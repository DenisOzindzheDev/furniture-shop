// cmd/migrate/main.go
package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/DenisOzindzheDev/furniture-shop/internal/config"
	"github.com/DenisOzindzheDev/furniture-shop/internal/migrate"
	_ "github.com/lib/pq"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := sql.Open("postgres", cfg.DBUrl)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Run migrations
	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "./migrations"
	}

	log.Printf("Running migrations from: %s", migrationsPath)

	migrator := migrate.NewMigrator(migrationsPath)
	if err := migrator.Run(db); err != nil {
		log.Fatal("Migration failed:", err)
	}

	log.Println("Migrations completed successfully")
}
