# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# ── Final stage ───────────────────────────────────────────────────────────────
FROM alpine:3.21

RUN addgroup -S lyo && adduser -S lyo -G lyo
WORKDIR /app

COPY --from=builder /server /app/server

USER lyo
EXPOSE 8080

ENTRYPOINT ["/app/server"]
