# Makefile
.PHONY: build up down logs clean

build:
	docker-compose build

up:
	docker-compose up -d

down:
	docker-compose down

logs:
	docker-compose logs -f

clean:
	docker-compose down -v
	docker system prune -f

migrate:
	# docker-compose exec api ./migrate -path /migrations -database "postgres://postgres:postgres@postgres:5432/furniture?sslmode=disable" up

test:
	go test ./...

# Запуск в development режиме
dev:
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# Запуск в production режиме  
prod:
	docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d