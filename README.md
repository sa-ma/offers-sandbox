# Offers Sandbox

A minimal full-stack sandbox demonstrating an event-driven offer engine.

- Go services: API (rules CRUD + Kafka), Engine (consumer/producer), Simulator (producer), Gateway (WebSocket relay)
- React + Vite frontend: Rules editor + live awards feed
- Infra: Postgres, Redpanda (Kafka), migrations via `migrate`

## Quickstart

1. Copy env and start

```
cp .env.example .env
docker compose up --build
```

2. Open the app

- Frontend: http://localhost:3000
- API: http://localhost:8080
- Gateway WS: ws://localhost:8083/ws

3. Try it out

- Create an offer (e.g., name: "10% bonus >= £10", minStake: 10, bonusPct: 0.10)
- Watch "Live Feed" as the simulator emits `events.bet_placed` and the engine awards bonuses

## Services & Topics

- Topics

  - `events.bet_placed` (simulator → engine)
  - `offers.rules` (API → engine)
  - `events.offer_awarded` (engine → gateway/web)
  - `events.bet_qualified` (Flink → engine)

- Services
  - `api`: CRUD offers in Postgres; on change publishes to `offers.rules`
  - `engine`: consumes qualified bets, applies rules (cached + hot-reload on `offers.rules`), writes awards, emits `events.offer_awarded`
  - `simulator`: ticks every N seconds and produces random bets
  - `gateway`: relays award events to browser via WebSocket
  - `web`: static build of React app
  - `stake-gate` (Flink): windows bets per user (2-min tumbling), forwards only those from windows whose sum ≥ threshold

## Environment

See `.env.example` for defaults.

- `KAFKA_BROKERS` (e.g., `redpanda:9092` in Docker)
- `POSTGRES_DSN` (e.g., `postgres://postgres:postgres@postgres:5432/offers?sslmode=disable`)
- `VITE_API_URL` and `VITE_WS_URL` for the frontend
- `MIN_WINDOW_STAKE` (default `50`): threshold for Flink gate
- `BET_TOPIC` (engine, default `events.bet_qualified`)

## Flink stake gate

This job computes a 2-minute tumbling window sum of stakes per user from `events.bet_placed` and forwards all bets in windows whose sum meets/exceeds `MIN_WINDOW_STAKE` to `events.bet_qualified`.

Build and run (requires network for Maven and image pulls):

```
docker compose build stake-gate
docker compose up -d flink-jobmanager flink-taskmanager stake-gate
```

You can inspect the Flink UI at http://localhost:8081.

## Notes

- Idempotency enforced via `awards.event_id UNIQUE`
- Redpanda is configured with auto-create-topics for local simplicity
- CORS is open (`*`) by default for the API
