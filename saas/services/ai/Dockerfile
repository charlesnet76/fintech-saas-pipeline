FROM golang:1.23-alpine AS builder
WORKDIR /app
ENV GOTOOLCHAIN=auto
COPY go.mod go.sum ./
RUN go mod download
COPY ai-main.go ./main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o ai-service .
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
ENV GOTOOLCHAIN=auto
COPY --from=builder /app/ai-service .
USER nobody
EXPOSE 8085
CMD ["./ai-service"]
