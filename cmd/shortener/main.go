package main

import (
	"errors"
	"fmt"
	"github.com/Totarae/URLShortener/internal/auth"
	"github.com/Totarae/URLShortener/internal/config"
	"github.com/Totarae/URLShortener/internal/database"
	"github.com/Totarae/URLShortener/internal/handlers"
	"github.com/Totarae/URLShortener/internal/repositories"
	"github.com/Totarae/URLShortener/internal/router"
	"github.com/Totarae/URLShortener/internal/util"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
	"net/http"
)

func main() {

	logger, err := zap.NewProduction()
	if err != nil {
		panic("Не удалось инициализировать логгер")
	}
	defer logger.Sync()

	// Инициализация конфигурации
	cfg := config.NewConfig()

	var db *database.DB
	var store *util.URLStore
	var repo *repositories.URLRepository

	if cfg.Mode == "database" {
		db, err = database.NewDB(logger)
		if err != nil {
			logger.Fatal("Ошибка подключения к базе данных", zap.Error(err))
		} else {
			logger.Info("DSN: ", zap.String("DB", cfg.DatabaseDSN))
		}
		defer db.Close()

		// run Postgres migrations
		if err := runPgMigrations(cfg); err != nil {
			logger.Fatal("runPgMigrations failed: %w", zap.Error(err))
		}
		repo = repositories.NewURLRepository(db)
	} else if cfg.Mode == "file" {
		store = util.NewURLStore(cfg.FileStoragePath)
	} else {
		store = util.NewURLStore(cfg.FileStoragePath)
	}

	authService := auth.New("rainbow-secret-key") // секрет должен быть из .env или конфигурации

	// Передача базового URL в обработчики
	handler := handlers.NewHandler(store, cfg.BaseURL, repo, logger, cfg.Mode, authService)

	r := router.NewRouter(handler, logger)

	logger.Info("Сервер запущен на ", zap.String("address", cfg.ServerAddress))
	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		logger.Fatal("Ошибка при запуске сервера: ", zap.Error(err))
	}

}

// runPgMigrations runs Postgres migrations
func runPgMigrations(cfg *config.Config) error {

	if cfg.PgMigrationsPath == "" {
		return nil
	}

	if cfg.DatabaseDSN == "" {
		return errors.New("no cfg.PgURL provided")
	}

	m, err := migrate.New(
		"file://"+cfg.PgMigrationsPath,
		cfg.DatabaseDSN,
	)
	if err != nil {
		return fmt.Errorf("ошибка при создании миграции: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("ошибка при применении миграции: %w", err)
	}

	return nil
}
