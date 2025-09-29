FROM golang:1.25-alpine AS builder


# Устанавливаем swag
RUN go install github.com/swaggo/swag/cmd/swag@latest

WORKDIR /app

# Устанавливаем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Генерируем Swagger документацию
RUN swag init -g cmd/api/main.go -o docs

# Собираем основное приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api ./cmd/api

# Собираем утилиту миграций
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o migrate ./cmd/migrate

# Финальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем бинарники
COPY --from=builder /app/api .
COPY --from=builder /app/migrate .

# Копируем конфигурацию
COPY --from=builder /app/config/config.yaml /var/furniture-shop-api/config.yaml

# Копируем миграции в правильную директорию
COPY --from=builder /app/migrations ./migrations/

# Копируем документацию
COPY --from=builder /app/docs ./docs/

# Создаем не-root пользователя
RUN addgroup -S app && adduser -S app -G app

# Меняем владельца файлов
RUN chown -R app:app /root/

USER app

EXPOSE 8080

CMD ["./api"]