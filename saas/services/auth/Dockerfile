# cache-bust: v2
FROM golang:1.23-alpine AS builder
WORKDIR /app
ENV GOTOOLCHAIN=auto
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o auth .
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
ENV GOTOOLCHAIN=auto
COPY --from=builder /app/auth .
USER nobody
EXPOSE 8081
CMD ["./auth"]
