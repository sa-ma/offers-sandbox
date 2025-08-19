# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/gateway

# Run stage
FROM alpine:3.20
RUN adduser -D -H appuser
USER appuser
ENV PORT=8083
EXPOSE 8083
COPY --from=builder /bin/gateway /bin/gateway
ENTRYPOINT ["/bin/gateway"]
