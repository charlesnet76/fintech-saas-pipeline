# cache-bust: v5
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
COPY *.go ./
RUN go build -o pipeline .
FROM alpine:3.19
RUN apk --no-cache add ca-certificates python3
WORKDIR /app
COPY --from=builder /app/pipeline .
COPY validator.py .
USER nobody
EXPOSE 8086
CMD ["./pipeline"]