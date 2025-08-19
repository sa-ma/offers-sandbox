package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/segmentio/kafka-go"
)

type Offer struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    MinStake  float64   `json:"minStake"`
    BonusPct  float64   `json:"bonusPct"`
    Active    bool      `json:"active"`
    CreatedAt time.Time `json:"createdAt"`
}

type Award struct {
    ID          string    `json:"awardId"`
    EventID     string    `json:"eventId"`
    UserID      string    `json:"userId"`
    OfferID     int       `json:"offerId"`
    Stake       float64   `json:"stake"`
    BonusAmount float64   `json:"bonusAmount"`
    CreatedAt   time.Time `json:"createdAt"`
}

type RuleEvent struct {
    Type string `json:"type"`
    Rule struct {
        ID        int     `json:"id"`
        MinStake  float64 `json:"minStake"`
        BonusPct  float64 `json:"bonusPct"`
        Active    bool    `json:"active"`
    } `json:"rule"`
}

type Server struct {
    db     *pgxpool.Pool
    writer *kafka.Writer
}

func main() {
    ctx := context.Background()

    port := getenv("PORT", "8080")
    dsn := getenv("POSTGRES_DSN", "postgres://postgres:postgres@postgres:5432/offers?sslmode=disable")
    brokers := strings.Split(getenv("KAFKA_BROKERS", "redpanda:9092"), ",")

    db, err := pgxpool.New(ctx, dsn)
    if err != nil {
        log.Fatalf("failed to connect db: %v", err)
    }
    defer db.Close()

    writer := &kafka.Writer{
        Addr:         kafka.TCP(brokers...),
        Topic:        "offers.rules",
        Balancer:     &kafka.LeastBytes{},
        RequiredAcks: kafka.RequireAll,
    }
    defer writer.Close()

    s := &Server{db: db, writer: writer}

    r := gin.Default()
    setupCORS(r)

    r.GET("/offers", s.listOffers)
    r.POST("/offers", s.createOffer)
    r.PUT("/offers/:id", s.updateOffer)
    r.GET("/awards", s.listAwards)

    log.Printf("Offer API listening on :%s", port)
    if err := r.Run(":" + port); err != nil {
        log.Fatal(err)
    }
}

func setupCORS(r *gin.Engine) {
    origins := getenv("CORS_ORIGINS", "*")
    r.Use(func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", origins)
        c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        if c.Request.Method == http.MethodOptions {
            c.AbortWithStatus(http.StatusNoContent)
            return
        }
        c.Next()
    })
}

func (s *Server) listOffers(c *gin.Context) {
    rows, err := s.db.Query(c.Request.Context(), `SELECT id, name, min_stake, bonus_pct, active, created_at FROM offers ORDER BY id`)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()
    out := make([]Offer, 0)
    for rows.Next() {
        var o Offer
        if err := rows.Scan(&o.ID, &o.Name, &o.MinStake, &o.BonusPct, &o.Active, &o.CreatedAt); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        out = append(out, o)
    }
    c.JSON(http.StatusOK, out)
}

type offerPayload struct {
    Name     string  `json:"name"`
    MinStake float64 `json:"minStake"`
    BonusPct float64 `json:"bonusPct"`
    Active   *bool   `json:"active"`
}

func (s *Server) createOffer(c *gin.Context) {
    var p offerPayload
    if err := c.BindJSON(&p); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    active := true
    if p.Active != nil {
        active = *p.Active
    }
    var id int
    err := s.db.QueryRow(c.Request.Context(), `INSERT INTO offers (name, min_stake, bonus_pct, active) VALUES ($1,$2,$3,$4) RETURNING id`,
        p.Name, p.MinStake, p.BonusPct, active).Scan(&id)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    // Publish rule change
    evt := RuleEvent{Type: "UPSERT"}
    evt.Rule.ID = id
    evt.Rule.MinStake = p.MinStake
    evt.Rule.BonusPct = p.BonusPct
    evt.Rule.Active = active
    if err := s.publishRule(c.Request.Context(), evt); err != nil {
        log.Printf("publish rule error: %v", err)
    }
    c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (s *Server) updateOffer(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    var p offerPayload
    if err := c.BindJSON(&p); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    // Keep active as is if nil
    var active bool
    if p.Active == nil {
        if err := s.db.QueryRow(c.Request.Context(), `SELECT active FROM offers WHERE id=$1`, id).Scan(&active); err != nil {
            c.JSON(http.StatusNotFound, gin.H{"error": "offer not found"})
            return
        }
    } else {
        active = *p.Active
    }
    ct, err := s.db.Exec(c.Request.Context(), `UPDATE offers SET name=$1, min_stake=$2, bonus_pct=$3, active=$4 WHERE id=$5`, p.Name, p.MinStake, p.BonusPct, active, id)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    if ct.RowsAffected() == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "offer not found"})
        return
    }
    evt := RuleEvent{Type: "UPSERT"}
    evt.Rule.ID = id
    evt.Rule.MinStake = p.MinStake
    evt.Rule.BonusPct = p.BonusPct
    evt.Rule.Active = active
    if err := s.publishRule(c.Request.Context(), evt); err != nil {
        log.Printf("publish rule error: %v", err)
    }
    c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) listAwards(c *gin.Context) {
    limitStr := c.DefaultQuery("limit", "100")
    limit, err := strconv.Atoi(limitStr)
    if err != nil || limit <= 0 || limit > 1000 {
        limit = 100
    }
    rows, err := s.db.Query(c.Request.Context(), `SELECT id, event_id, user_id, offer_id, stake, bonus_amount, created_at FROM awards ORDER BY created_at DESC LIMIT $1`, limit)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()
    out := make([]Award, 0)
    for rows.Next() {
        var a Award
        if err := rows.Scan(&a.ID, &a.EventID, &a.UserID, &a.OfferID, &a.Stake, &a.BonusAmount, &a.CreatedAt); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        out = append(out, a)
    }
    c.JSON(http.StatusOK, out)
}

func (s *Server) publishRule(ctx context.Context, evt RuleEvent) error {
    b, err := json.Marshal(evt)
    if err != nil {
        return err
    }
    msg := kafka.Message{
        Time:  time.Now(),
        Value: b,
        Headers: []kafka.Header{{Key: "content-type", Value: []byte("application/json")}},
    }
    return s.writer.WriteMessages(ctx, msg)
}

func getenv(k, def string) string { v := os.Getenv(k); if v == "" { return def }; return v }
