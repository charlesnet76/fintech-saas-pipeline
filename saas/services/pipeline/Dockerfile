# cache-bust: v6
FROM golang:1.23-alpine AS builder
WORKDIR /app
ENV GOTOOLCHAIN=auto
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o pipeline .
FROM alpine:3.19
RUN apk --no-cache add ca-certificates python3
WORKDIR /app
COPY --from=builder /app/pipeline .
COPY validator.py .
COPY predict.py .
USER nobody
EXPOSE 8086
CMD ["./pipeline"]