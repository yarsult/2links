# Этап сборки (builder)
FROM golang:1.23-bullseye AS builder
WORKDIR /app

# Копируем файлы зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код в рабочую директорию
COPY . .

# Кэшируем сборку для ускорения
RUN --mount=type=cache,target="/root/.cache/go-build" go build -o bot cmd/main.go

# Этап финального контейнера
FROM ubuntu:22.04
WORKDIR /app
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
# Копируем скомпилированный бинарник из builder
COPY --from=builder /app/bot .

# Копируем .env файл в ту же директорию
COPY cmd/.env .env

# Указываем переменную окружения для пути к .env
ENV ENV_PATH=/app/.env

# Указываем точку входа
ENTRYPOINT ["/app/bot"]