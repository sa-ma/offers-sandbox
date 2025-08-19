package main

import (
    "context"
    "encoding/json"
    "log"
    "math"
    "os"
    "sort"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/segmentio/kafka-go"
)

type Offer struct {
    ID       int
    MinStake float64
    BonusPct float64
    Active   bool
}

type BetPlaced struct {
    EventID  string  `json:"eventId"`
    Ts       int64   `json:"ts"`
    UserID   string  `json:"userId"`
    Market   string  `json:"market"`
    Stake    float64 `json:"stake"`
    Currency string  `json:"currency"`
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

type AwardEvent struct {
    AwardID     string  `json:"awardId"`
    EventID     string  `json:"eventId"`
    UserID      string  `json:"userId"`
    OfferID     int     `json:"offerId"`
    Stake       float64 `json:"stake"`
    BonusAmount float64 `json:"bonusAmount"`
    Ts          int64   `json:"ts"`
}

func main() {
    ctx := context.Background()

    brokers := strings.Split(getenv("KAFKA_BROKERS", "redpanda:9092"), ",")
    groupID := getenv("GROUP_ID", "offer-engine")
    dsn := getenv("POSTGRES_DSN", "postgres://postgres:postgres@postgres:5432/offers?sslmode=disable")

    db, err := pgxpool.New(ctx, dsn)
    if err != nil {
        log.Fatalf("db connect error: %v", err)
    }
    defer db.Close()

    offers := loadOffers(ctx, db)
    log.Printf("loaded %d offers", len(offers))

    // Kafka readers
    betTopic := getenv("BET_TOPIC", "events.bet_qualified")
    betReader := kafka.NewReader(kafka.ReaderConfig{
        Brokers:  brokers,
        GroupID:  groupID,
        Topic:    betTopic,
        MinBytes: 1,
        MaxBytes: 10e6,
    })
    defer betReader.Close()

    ruleReader := kafka.NewReader(kafka.ReaderConfig{
        Brokers:  brokers,
        GroupID:  groupID + "-rules",
        Topic:    "offers.rules",
        MinBytes: 1,
        MaxBytes: 10e6,
    })
    defer ruleReader.Close()

    writer := &kafka.Writer{Addr: kafka.TCP(brokers...), Topic: "events.offer_awarded", Balancer: &kafka.LeastBytes{}}
    defer writer.Close()

    // Apply rule updates in background
    ruleCh := make(chan RuleEvent, 16)
    go func() {
        for {
            m, err := ruleReader.ReadMessage(ctx)
            if err != nil {
                log.Printf("rule reader error: %v", err)
                time.Sleep(time.Second)
                continue
            }
            var re RuleEvent
            if err := json.Unmarshal(m.Value, &re); err != nil {
                log.Printf("rule unmarshal: %v", err)
                continue
            }
            ruleCh <- re
        }
    }()

    // Processing loop
    for {
        select {
        case re := <-ruleCh:
            // Upsert rule in cache
            found := false
            for i := range offers {
                if offers[i].ID == re.Rule.ID {
                    offers[i].MinStake = re.Rule.MinStake
                    offers[i].BonusPct = re.Rule.BonusPct
                    offers[i].Active = re.Rule.Active
                    found = true
                    break
                }
            }
            if !found {
                offers = append(offers, Offer{ID: re.Rule.ID, MinStake: re.Rule.MinStake, BonusPct: re.Rule.BonusPct, Active: re.Rule.Active})
            }
        default:
        }

        m, err := betReader.FetchMessage(ctx)
        if err != nil {
            log.Printf("bet reader error: %v", err)
            time.Sleep(time.Second)
            continue
        }
        var bet BetPlaced
        if err := json.Unmarshal(m.Value, &bet); err != nil {
            log.Printf("bet unmarshal: %v", err)
            _ = betReader.CommitMessages(ctx, m)
            continue
        }

        // Apply rules: pick the best bonus among active and matching minStake
        bestIdx := -1
        var bestAmt float64
        for i := range offers {
            o := offers[i]
            if !o.Active || bet.Stake < o.MinStake {
                continue
            }
            amt := round2(bet.Stake * o.BonusPct)
            if amt > bestAmt {
                bestAmt = amt
                bestIdx = i
            }
        }

        if bestIdx >= 0 && bestAmt > 0 {
            best := offers[bestIdx]
            // Insert award idempotently (event_id unique)
            awardID := uuid.New().String()
            _, err := db.Exec(ctx, `INSERT INTO awards (id, event_id, user_id, stake, bonus_amount, offer_id) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (event_id) DO NOTHING`,
                awardID, bet.EventID, bet.UserID, bet.Stake, bestAmt, best.ID)
            if err != nil {
                log.Printf("award insert: %v", err)
            } else {
                evt := AwardEvent{
                    AwardID:     awardID,
                    EventID:     bet.EventID,
                    UserID:      bet.UserID,
                    OfferID:     best.ID,
                    Stake:       bet.Stake,
                    BonusAmount: bestAmt,
                    Ts:          time.Now().UnixMilli(),
                }
                b, _ := json.Marshal(evt)
                if err := writer.WriteMessages(ctx, kafka.Message{Value: b}); err != nil {
                    log.Printf("produce award: %v", err)
                }
            }
        }

        if err := betReader.CommitMessages(ctx, m); err != nil {
            log.Printf("commit error: %v", err)
        }
    }
}

func loadOffers(ctx context.Context, db *pgxpool.Pool) []Offer {
    rows, err := db.Query(ctx, `SELECT id, min_stake, bonus_pct, active FROM offers WHERE active = TRUE`)
    if err != nil {
        log.Printf("load offers: %v", err)
        return nil
    }
    defer rows.Close()
    var out []Offer
    for rows.Next() {
        var o Offer
        if err := rows.Scan(&o.ID, &o.MinStake, &o.BonusPct, &o.Active); err != nil {
            log.Printf("scan offer: %v", err)
            return out
        }
        out = append(out, o)
    }
    // stable ordering
    sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
    return out
}

func getenv(k, def string) string { v := os.Getenv(k); if v == "" { return def }; return v }
func round2(v float64) float64    { return math.Round(v*100) / 100 }
