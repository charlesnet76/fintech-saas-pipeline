# cache-bust: v4
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download -x 2>&1 || (go mod download -x && exit 1)
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o pipeline .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates python3 py3-pip
WORKDIR /app
COPY --from=builder /app/pipeline .
COPY validator.py .
USER nobody
EXPOSE 8086
CMD ["./pipeline"]
