package main

import (
	"github.com/Totarae/URLShortener/internal/config"
	"github.com/Totarae/URLShortener/internal/database"
	"github.com/Totarae/URLShortener/internal/handlers"
	"github.com/Totarae/URLShortener/internal/router"
	"github.com/Totarae/URLShortener/internal/util"
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
	logger.Info("Подключение к БД установлено", zap.String("DSN", cfg.DatabaseDSN))

	store := util.NewURLStore(cfg.FileStoragePath)

	// Передача базового URL в обработчики
	handler := handlers.NewHandler(store, cfg.BaseURL, db, logger)

	r := router.NewRouter(handler, logger)

	logger.Info("Сервер запущен на ", zap.String("address", cfg.ServerAddress))
	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		logger.Fatal("Ошибка при запуске сервера: ", zap.Error(err))
	}

}
