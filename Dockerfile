# Build Go binary
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o vaulthook ./cmd/api
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Minimal runtime
FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/vaulthook .
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY migrations/ ./migrations/
COPY deploy/entrypoint.sh .
RUN chmod +x entrypoint.sh
EXPOSE 8080
ENTRYPOINT ["./entrypoint.sh"]
