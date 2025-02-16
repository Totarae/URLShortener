package main

import (
	"github.com/Totarae/URLShortener/internal/config"
	"github.com/Totarae/URLShortener/internal/handlers"
	"github.com/Totarae/URLShortener/internal/router"
	"github.com/Totarae/URLShortener/internal/util"
	"go.uber.org/zap"
	"net/http"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Инициализация конфигурации
	cfg := config.NewConfig()

	store := util.NewURLStore(cfg.FileStoragePath)

	// Передача базового URL в обработчики
	handler := handlers.NewHandler(store, cfg.BaseURL)

	r := router.NewRouter(handler, logger)

	logger.Info("Сервер запущен на ", zap.String("address", cfg.ServerAddress))
	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		logger.Fatal("Ошибка при запуске сервера: ", zap.Error(err))
	}

}
