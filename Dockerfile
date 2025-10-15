# =======================
# Builder stage
# =======================
FROM golang:1.25-alpine AS builder

# Устанавливаем необходимые пакеты
RUN apk add --no-cache git bash

WORKDIR /app

# Копируем зависимости для кеширования
COPY go.mod go.sum ./
RUN go mod download

# Устанавливаем swag для генерации документации
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Копируем весь исходный код
COPY . .

# Генерируем swagger документацию
RUN swag init -g cmd/api/main.go -o docs

# Сборка бинарей для Linux без CGO
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o migrate ./cmd/migrate

# =======================
# Production stage
# =======================
FROM gcr.io/distroless/static-debian11

WORKDIR /app

# Копируем собранные бинарники
COPY --from=builder /app/api .
COPY --from=builder /app/migrate .

# Копируем конфиг в рабочую директорию приложения
COPY --from=builder /app/configs/config.yaml ./config.yaml

# Копируем миграции и swagger
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/docs ./docs

# Открываем порт для приложения
EXPOSE 8080

# Указываем команду запуска
CMD ["./api"]
