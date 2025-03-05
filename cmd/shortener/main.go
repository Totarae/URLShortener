package main

import (
	"errors"
	"fmt"
	"github.com/Totarae/URLShortener/internal/config"
	"github.com/Totarae/URLShortener/internal/database"
	"github.com/Totarae/URLShortener/internal/handlers"
	"github.com/Totarae/URLShortener/internal/router"
	"github.com/Totarae/URLShortener/internal/util"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
	"net/http"
	"os"
)

func main() {

	logger, err := zap.NewProduction()
	if err != nil {
		panic("Не удалось инициализировать логгер")
	}
	defer logger.Sync()

	// Инициализация конфигурации
	cfg := config.NewConfig()

	// Устанавливаем DATABASE_DSN
	if cfg.DatabaseDSN == "" {
		logger.Fatal("DATABASE_DSN is not set")
	}

	os.Setenv("DATABASE_DSN", cfg.DatabaseDSN)

	db, err := database.NewDB(logger)
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

	store := util.NewURLStore(cfg.FileStoragePath)

	// Передача базового URL в обработчики
	handler := handlers.NewHandler(store, cfg.BaseURL, db, logger)

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
		return errors.New("No cfg.PgURL provided")
	}

	m, err := migrate.New(
		"file://"+cfg.PgMigrationsPath,
		cfg.DatabaseDSN,
	)
	if err != nil {
		return fmt.Errorf("ошибка при создании миграции: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("ошибка при применении миграции: %w", err)
	}

	return nil
}
