FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY auth-main.go ./main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o auth .
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/auth .
USER nobody
EXPOSE 8081
CMD ["./auth"]
