FROM golang:1.23-bullseye AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN --mount=type=cache,target="/root/.cache/go-build" go build -o bot cmd/main.go

FROM ubuntu:22.04
WORKDIR /app
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/bot .

COPY cmd/.env .env

ENV ENV_PATH=/app/.env

ENTRYPOINT ["/app/bot"]