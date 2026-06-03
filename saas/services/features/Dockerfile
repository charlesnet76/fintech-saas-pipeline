# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o features .

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.20

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY --from=builder /app/features .

USER appuser
EXPOSE 8082

CMD ["./features"]
