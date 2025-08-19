# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/engine

# Run stage
FROM alpine:3.20
RUN adduser -D -H appuser
USER appuser
COPY --from=builder /bin/engine /bin/engine
ENTRYPOINT ["/bin/engine"]
