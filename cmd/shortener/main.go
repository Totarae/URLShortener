package main

import (
	"fmt"
	"github.com/Totarae/URLShortener/internal/config"
	"github.com/Totarae/URLShortener/internal/handlers"
	"github.com/Totarae/URLShortener/internal/router"
	"github.com/Totarae/URLShortener/internal/util"
	"net/http"
)

func main() {

	// Инициализация конфигурации
	cfg := config.NewConfig()

	// Проверка корректности конфигурации
	if err := cfg.Validate(); err != nil {
		fmt.Printf("Ошибка конфигурации: %v\n", err)
		return
	}

	store := util.NewURLStore()

	// Передача базового URL в обработчики
	handler := handlers.NewHandler(store, cfg.BaseURL)

	r := router.NewRouter(handler)
	
	fmt.Printf("Сервер запущен на %s\n", cfg.ServerAddress)
	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		fmt.Printf("Ошибка при запуске сервера: %v\n", err)
	}

}
