FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install SSL certificates and timezone data for production
RUN apk --no-cache add ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o dez-cron .

FROM alpine:latest

# Copy SSL certs and timezone data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

WORKDIR /app
COPY --from=builder /app/dez-cron .

EXPOSE 8080

CMD ["./dez-cron"]
