package main

import (
	"context"
	"database/sql"
	serv "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/DenisOzindzheDev/furniture-shop/docs"
	"github.com/DenisOzindzheDev/furniture-shop/internal/auth"
	"github.com/DenisOzindzheDev/furniture-shop/internal/config"
	"github.com/DenisOzindzheDev/furniture-shop/internal/handler/http"
	"github.com/DenisOzindzheDev/furniture-shop/internal/kafka"
	"github.com/DenisOzindzheDev/furniture-shop/internal/migrate"
	"github.com/DenisOzindzheDev/furniture-shop/internal/repository/postgres"
	redisRepo "github.com/DenisOzindzheDev/furniture-shop/internal/repository/redis"
	"github.com/DenisOzindzheDev/furniture-shop/internal/service"
	"github.com/DenisOzindzheDev/furniture-shop/internal/storage"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
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

	db, err := sql.Open("postgres", cfg.DBUrl)
	if err != nil {
		sugar.Fatalw("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := waitForDB(db, 30*time.Second); err != nil {
		sugar.Fatalw("Failed to connect to database after retry:", err)
	}

	migrations, err := sql.Open("postgres", cfg.DBUrl)
	if err != nil {
		sugar.Fatalw("Failed to connect to database:", err)
	}
	defer migrations.Close()

	if err := runMigrations(migrations); err != nil {
		sugar.Fatalw("Failed to run migrations:", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
	defer redisClient.Close()

	s3Storage, err := storage.NewS3Storage(&cfg.AWS)
	if err != nil {
		sugar.Infof("Warning: Failed to initialize S3 storage: %v", err)
		sugar.Infow("Continuing without S3 storage...")
	}

	if err := waitForRedis(redisClient, 30*time.Second); err != nil {
		sugar.Fatalw("Failed to connect to Redis after retry:", err)
	}

	cache := redisRepo.NewCache(cfg.RedisAddr, 30*time.Minute)
	defer cache.Close()

	producer := kafka.NewProducer(cfg.KafkaBrokers, "furniture-events")
	defer producer.Close()

	jwtManager := auth.NewJWTManager(cfg.JWTSecret, 24*time.Hour)
	imageService := service.NewImageService(s3Storage, cfg)
	userRepo := postgres.NewUserRepo(db)
	productRepo := postgres.NewProductRepo(db)
	userService := service.NewUserService(userRepo, jwtManager, producer)
	productService := service.NewProductService(productRepo, imageService, cache)
	userHandler := http.NewUserHandler(userService)
	productHandler := http.NewProductHandler(productService)
	healthHandler := http.NewHealthHandler(db, redisClient, nil)
	productAdminHandler := http.NewProductAdminHandler(productService)
	pdfService := service.NewPDFService("http://localhost:8080")
	productPDFHandler := http.NewProductPDFHandler(productService, pdfService)

	mux := serv.NewServeMux()

	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"), // URL pointing to API definition
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	mux.HandleFunc("GET /api/health", healthHandler.HealthCheck)
	mux.HandleFunc("POST /api/register", userHandler.Register)
	mux.HandleFunc("POST /api/login", userHandler.Login)
	mux.HandleFunc("GET /api/products", productHandler.ListProducts)
	mux.HandleFunc("GET /api/products/{id}", productHandler.GetProduct)
	mux.HandleFunc("GET /api/products/{id}/download", productPDFHandler.DownloadProductPDF)
	mux.HandleFunc("GET /api/products/{id}/preview", productPDFHandler.PreviewProductPDF)

	authMiddleware := auth.AuthMiddleware(jwtManager)
	mux.Handle("GET /api/profile", authMiddleware(serv.HandlerFunc(userHandler.Profile)))

	adminMiddleware := auth.AuthMiddleware(jwtManager)
	mux.Handle("POST /api/admin/products", adminMiddleware(serv.HandlerFunc(productAdminHandler.CreateProduct)))
	mux.Handle("PUT /api/admin/products/{id}", adminMiddleware(serv.HandlerFunc(productAdminHandler.UpdateProduct)))
	mux.Handle("DELETE /api/admin/products/{id}", adminMiddleware(serv.HandlerFunc(productAdminHandler.DeleteProduct)))
	mux.Handle("GET /api/admin/products", adminMiddleware(serv.HandlerFunc(productAdminHandler.ListProducts)))

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000"}, // Nuxt dev server
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		Debug:            cfg.CorsDebug,
	})
	handlerWithCORS := c.Handler(mux)

	server := &serv.Server{
		Addr:         cfg.HTTPPort,
		Handler:      handlerWithCORS,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	go func() {
		sugar.Infow("starting server", "port", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != serv.ErrServerClosed {
			sugar.Fatalw("server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	sugar.Infow("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		sugar.Fatalw("server forced to shutdown", "error", err)
	}

	sugar.Infow("server exited properly")
}

func waitForDB(db *sql.DB, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := db.Ping(); err == nil {
				return nil
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func waitForRedis(redisClient *redis.Client, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if _, err := redisClient.Ping(ctx).Result(); err == nil {
				return nil
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func runMigrations(db *sql.DB) error {
	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "./migrations"
	}

	migrator := migrate.NewMigrator(migrationsPath)
	return migrator.Run(db)
}
