// internal/migrate/migrate.go
package migrate

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type Migrator struct {
	migrationsPath string
}

func NewMigrator(migrationsPath string) *Migrator {
	return &Migrator{
		migrationsPath: migrationsPath,
	}
}

func (m *Migrator) Run(db *sql.DB) error {
	// Проверяем существование директории с миграциями
	if _, err := os.Stat(m.migrationsPath); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory does not exist: %s", m.migrationsPath)
	}

	// Логируем информацию о пути
	log.Printf("Running migrations from: %s", m.migrationsPath)

	// Получаем список файлов в директории
	files, err := os.ReadDir(m.migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	log.Printf("Found %d files in migrations directory", len(files))
	for _, file := range files {
		log.Printf("  - %s", file.Name())
	}

	// Создаем драйвер для PostgreSQL
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Создаем мигратор
	// Используем относительный путь, так как в Docker мы уже в правильной директории
	migrator, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", m.migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer migrator.Close()

	// Выполняем миграции
	log.Println("Running database migrations...")
	err = migrator.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Println("No new migrations to apply")
	} else {
		log.Println("Migrations applied successfully")
	}

	// Проверяем версию миграции
	version, dirty, err := migrator.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	log.Printf("Current migration version: %d (dirty: %t)", version, dirty)

	return nil
}
