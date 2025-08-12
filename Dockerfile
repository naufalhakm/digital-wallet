FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates build-base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server

FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app
RUN mkdir -p logs

COPY --from=builder /app/server .
COPY --from=builder /app/pkg/database/migrations /app/pkg/database/migrations

EXPOSE 8080

CMD ["./server"]
