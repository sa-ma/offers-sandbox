# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/api

# Run stage
FROM alpine:3.20
RUN adduser -D -H appuser
USER appuser
ENV PORT=8080
EXPOSE 8080
COPY --from=builder /bin/api /bin/api
ENTRYPOINT ["/bin/api"]
