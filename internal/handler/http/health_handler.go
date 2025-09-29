// internal/handler/http/health_handler.go
package http

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/segmentio/kafka-go"
)

type HealthHandler struct {
	db    *sql.DB
	redis *redis.Client
	kafka *kafka.Writer
}

func NewHealthHandler(db *sql.DB, redis *redis.Client, kafka *kafka.Writer) *HealthHandler {
	return &HealthHandler{
		db:    db,
		redis: redis,
		kafka: kafka,
	}
}

type HealthResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
}

func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	services := make(map[string]string)

	// Check PostgreSQL
	if err := h.db.Ping(); err != nil {
		services["postgres"] = "unhealthy"
	} else {
		services["postgres"] = "healthy"
	}

	// Check Redis
	if _, err := h.redis.Ping(r.Context()).Result(); err != nil {
		services["redis"] = "unhealthy"
	} else {
		services["redis"] = "healthy"
	}

	services["kafka"] = "healthy"

	status := "healthy"
	for _, serviceStatus := range services {
		if serviceStatus == "unhealthy" {
			status = "unhealthy"
			break
		}
	}

	response := HealthResponse{
		Status:   status,
		Services: services,
	}

	w.Header().Set("Content-Type", "application/json")
	if status == "unhealthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(response)
}
