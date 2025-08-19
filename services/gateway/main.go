package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "strings"
    "sync"
    "time"

    "github.com/gorilla/websocket"
    "github.com/segmentio/kafka-go"
)

type Hub struct {
    mu      sync.Mutex
    clients map[*websocket.Conn]struct{}
}

func (h *Hub) add(c *websocket.Conn) {
    h.mu.Lock(); defer h.mu.Unlock()
    if h.clients == nil { h.clients = make(map[*websocket.Conn]struct{}) }
    h.clients[c] = struct{}{}
}
func (h *Hub) remove(c *websocket.Conn) {
    h.mu.Lock(); defer h.mu.Unlock()
    delete(h.clients, c)
}
func (h *Hub) broadcast(msg []byte) {
    h.mu.Lock(); defer h.mu.Unlock()
    for c := range h.clients {
        c.SetWriteDeadline(time.Now().Add(2 * time.Second))
        if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
            c.Close()
            delete(h.clients, c)
        }
    }
}

func main() {
    port := getenv("PORT", "8083")
    topic := getenv("TOPIC", "events.offer_awarded")
    brokers := strings.Split(getenv("KAFKA_BROKERS", "redpanda:9092"), ",")

    hub := &Hub{clients: make(map[*websocket.Conn]struct{})}

    upgrader := websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool { return true },
    }

    http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        conn, err := upgrader.Upgrade(w, r, nil)
        if err != nil { return }
        hub.add(conn)
        go func() {
            for {
                if _, _, err := conn.NextReader(); err != nil {
                    hub.remove(conn)
                    conn.Close()
                    return
                }
            }
        }()
    })

    // Kafka consumer (single shared reader)
    ctx := context.Background()
    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers: brokers,
        GroupID: "web-gateway",
        Topic:   topic,
    })
    go func() {
        defer reader.Close()
        for {
            m, err := reader.ReadMessage(ctx)
            if err != nil { log.Printf("kafka read: %v", err); time.Sleep(time.Second); continue }
            hub.broadcast(m.Value)
        }
    }()

    log.Printf("Web Gateway listening on :%s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getenv(k, def string) string { v := os.Getenv(k); if v == "" { return def }; return v }

