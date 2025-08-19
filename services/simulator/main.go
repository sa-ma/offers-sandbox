package main

import (
    "context"
    "encoding/json"
    "log"
    "math/rand"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/segmentio/kafka-go"
)

type BetPlaced struct {
    EventID  string  `json:"eventId"`
    Ts       int64   `json:"ts"`
    UserID   string  `json:"userId"`
    Market   string  `json:"market"`
    Stake    float64 `json:"stake"`
    Currency string  `json:"currency"`
}

func main() {
    ctx := context.Background()
    brokers := strings.Split(getenv("KAFKA_BROKERS", "redpanda:9092"), ",")
    tickSeconds := getenvInt("TICK_SECONDS", 2)

    writer := &kafka.Writer{Addr: kafka.TCP(brokers...), Topic: "events.bet_placed", Balancer: &kafka.LeastBytes{}}
    defer writer.Close()

    markets := []string{"EPL", "UCL", "NFL", "NBA"}
    users := 2000
    r := rand.New(rand.NewSource(time.Now().UnixNano()))

    log.Printf("simulator producing to %v every %ds", brokers, tickSeconds)
    t := time.NewTicker(time.Duration(tickSeconds) * time.Second)
    for {
        <-t.C
        bet := BetPlaced{
            EventID:  uuid.New().String(),
            Ts:       time.Now().UnixMilli(),
            UserID:   "u-" + strconv.Itoa(r.Intn(users)+1),
            Market:   markets[r.Intn(len(markets))],
            Stake:    float64(5+r.Intn(95)) + r.Float64(),
            Currency: "GBP",
        }
        b, _ := json.Marshal(bet)
        if err := writer.WriteMessages(ctx, kafka.Message{Value: b}); err != nil {
            log.Printf("produce error: %v", err)
        }
    }
}

func getenv(k, def string) string { v := os.Getenv(k); if v == "" { return def }; return v }
func getenvInt(k string, def int) int {
    v := os.Getenv(k)
    if v == "" { return def }
    if n, err := strconv.Atoi(v); err == nil { return n }
    return def
}
