package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
)

// Переменные для версии сборки
var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {

	logger, err := zap.NewProduction()
	if err != nil {
		panic("Не удалось инициализировать логгер")
	}
	defer logger.Sync()

	logger.Info("Информация о сборке",
		zap.String("Build version", buildVersion),
		zap.String("Build date", buildDate),
		zap.String("Build commit", buildCommit),
	)

	// Инициализация конфигурации
	cfg := config.NewConfig()

	var db *database.DB
	var store *util.URLStore
	var repo *repositories.URLRepository

	if cfg.Mode == "database" {
		db, err = database.NewDB(logger)
		if err != nil {
			logger.Error("Ошибка подключения к базе данных", zap.Error(err))
			return
		} else {
			logger.Info("DSN: ", zap.String("DB", cfg.DatabaseDSN))
		}
		defer db.Close()

		// run Postgres migrations
		if err := runPgMigrations(cfg); err != nil {

			logger.Error("runPgMigrations failed", zap.Error(err))
			return
		}
		repo = repositories.NewURLRepository(db)
	} else {
		store = util.NewURLStore(cfg.FileStoragePath)
	}

	authService := auth.New("rainbow-secret-key") // секрет должен быть из .env или конфигурации

	// Передача базового URL в обработчики
	handler := handlers.NewHandler(store, cfg.BaseURL, repo, logger, cfg.Mode, authService)

	r := router.NewRouter(handler, logger)

	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: r,
	}

	// Контекст завершения по сигналу
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	logger.Info("Сервер запущен на ", zap.String("address", cfg.ServerAddress))

	// Запуск сервера
	go func() {
		logger.Info("Сервер запущен", zap.String("address", cfg.ServerAddress))
		var err error
		if cfg.EnableHTTPS {
			if _, err := os.Stat(cfg.TLSCertPath); os.IsNotExist(err) {
				logger.Fatal("Файл сертификата не найден", zap.String("path", cfg.TLSCertPath))
			}
			if _, err := os.Stat(cfg.TLSKeyPath); os.IsNotExist(err) {
				logger.Fatal("Файл ключа не найден", zap.String("path", cfg.TLSKeyPath))
			}
			logger.Info("HTTPS включён", zap.String("cert", cfg.TLSCertPath), zap.String("key", cfg.TLSKeyPath))
			err = server.ListenAndServeTLS(cfg.TLSCertPath, cfg.TLSKeyPath)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("Ошибка сервера", zap.Error(err))
		}
	}()

	// Ждём завершения
	<-ctx.Done()
	stop()
	logger.Info("Получен сигнал завершения. Завершаем сервер...")

	// Таймаут на graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Ошибка при завершении сервера", zap.Error(err))
	}

	// Сохраняем данные из хранилища
	if cfg.Mode == "file" && store != nil {
		if err := store.SaveToFile(); err != nil {
			logger.Error("Ошибка при сохранении в файл", zap.Error(err))
		} else {
			logger.Info("Данные успешно сохранены в файл")
		}
	}

	logger.Info("Сервер завершён корректно")

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
