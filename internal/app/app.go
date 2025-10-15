package app

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"time"

	"github.com/DenisOzindzheDev/furniture-shop/internal/auth"
	"github.com/DenisOzindzheDev/furniture-shop/internal/config"
	"github.com/DenisOzindzheDev/furniture-shop/internal/infra/kafka"
	"github.com/DenisOzindzheDev/furniture-shop/internal/infra/postgres"
	"github.com/DenisOzindzheDev/furniture-shop/internal/infra/redis"
	"github.com/DenisOzindzheDev/furniture-shop/internal/infra/s3"
	"github.com/DenisOzindzheDev/furniture-shop/internal/migrate"
	"github.com/DenisOzindzheDev/furniture-shop/internal/service"
	router "github.com/DenisOzindzheDev/furniture-shop/internal/transport/http"
	redisClient "github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type App struct {
	server *http.Server
	db     *sql.DB
	cache  *redisClient.Client
	prod   *kafka.Producer
	log    *zap.SugaredLogger
}

func New(cfg *config.Config, log *zap.SugaredLogger) (*App, error) {
	// Подключение к Postgres
	db, err := sql.Open("postgres", cfg.DBUrl)
	if err != nil {
		return nil, err
	}

	if err := waitForDB(db, 30*time.Second); err != nil {
		return nil, err
	}

	// Миграции
	if err := runMigrations(db); err != nil {
		return nil, err
	}

	// Redis
	rdb := redisClient.NewClient(&redisClient.Options{Addr: cfg.RedisAddr})
	if err := waitForRedis(rdb, 30*time.Second); err != nil {
		return nil, err
	}

	// Kafka
	producer := kafka.NewProducer(cfg.KafkaBrokers, "furniture-events")

	// S3
	s3Storage, err := s3.NewS3Storage(&cfg.AWS)
	if err != nil {
		log.Warnw("Failed to init S3 storage", "error", err)
	}

	// Сервисы и репозитории
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, 24*time.Hour)
	imageService := service.NewImageService(s3Storage, cfg)

	userRepo := postgres.NewUserRepo(db)
	productRepo := postgres.NewProductRepo(db)
	cacheRepo := redis.NewCache(cfg.RedisAddr, 30*time.Minute)

	userService := service.NewUserService(userRepo, jwtManager, producer)
	productService := service.NewProductService(productRepo, imageService, cacheRepo)
	pdfService := service.NewPDFService("http://localhost:8080")

	// HTTP маршрутизатор
	mux := router.New(cfg, db, rdb, jwtManager, userService, productService, pdfService)

	server := &http.Server{
		Addr:         cfg.HTTPPort,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return &App{
		server: server,
		db:     db,
		cache:  rdb,
		prod:   producer,
		log:    log,
	}, nil
}

func (a *App) Run() error {
	return a.server.ListenAndServe()
}

func (a *App) Stop(ctx context.Context) error {
	a.log.Infow("closing resources...")
	if err := a.server.Shutdown(ctx); err != nil {
		return err
	}
	if a.db != nil {
		_ = a.db.Close()
	}
	if a.cache != nil {
		_ = a.cache.Close()
	}
	if a.prod != nil {
		a.prod.Close()
	}
	return nil
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

func waitForRedis(redisClient *redisClient.Client, timeout time.Duration) error {
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
