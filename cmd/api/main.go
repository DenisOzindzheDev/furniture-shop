package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/DenisOzindzheDev/furniture-shop/docs"
	"github.com/DenisOzindzheDev/furniture-shop/internal/app"
	"github.com/DenisOzindzheDev/furniture-shop/internal/config"
	"go.uber.org/zap"
)

// @title Furniture Store API
// @version 1.0
// @description API для интернет-магазина мебели
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api
// @schemes http

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT токен в формате: "Bearer {token}"
func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	sugar := logger.Sugar()

	cfg := config.Load()

	application, err := app.New(cfg, sugar)
	if err != nil {
		sugar.Fatalw("Failed to initialize application", "error", err)
	}

	go func() {
		sugar.Infow("starting server", "addr", cfg.HTTPPort)
		if err := application.Run(); err != nil && err != http.ErrServerClosed {
			sugar.Fatalw("server exited with error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	sugar.Infow("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := application.Stop(ctx); err != nil {
		sugar.Fatalw("failed to shutdown gracefully", "error", err)
	}

	sugar.Infow("server stopped cleanly")
}
