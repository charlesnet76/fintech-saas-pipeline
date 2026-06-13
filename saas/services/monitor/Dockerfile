# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

WORKDIR /app
ENV GOTOOLCHAIN=auto
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o monitor .

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.20

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
ENV GOTOOLCHAIN=auto
COPY --from=builder /app/monitor .

USER appuser
EXPOSE 8083

CMD ["./monitor"]
