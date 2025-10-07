# Builder stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git
WORKDIR /app

# Копируем зависимости first для лучшего кэширования
COPY go.mod go.sum ./
RUN go mod download

# Устанавливаем swag и копируем исходный код
RUN go install github.com/swaggo/swag/cmd/swag@latest
COPY . .

# Генерируем swagger и собираем бинарники
RUN swag init -g cmd/api/main.go -o docs
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -a -installsuffix cgo -o api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -a -installsuffix cgo -o migrate ./cmd/migrate

# Production stage
FROM gcr.io/distroless/static-debian11

WORKDIR /app

# Копируем только необходимые файлы
COPY --from=builder /app/api .
COPY --from=builder /app/migrate .
COPY --from=builder /app/config/config.yaml /var/furniture-shop-api/config.yaml
COPY --from=builder /app/migrations ./migrations/
COPY --from=builder /app/docs ./docs/

EXPOSE 8080

CMD ["./api"]