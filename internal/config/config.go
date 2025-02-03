package config

import (
	"flag"
	"fmt"
)

// Config хранит конфигурацию сервера
type Config struct {
	ServerAddress string
	BaseURL       string
}

// InitConfig инициализирует конфигурацию на основе аргументов командной строки
func InitConfig() *Config {
	cfg := &Config{}

	// Определение флагов
	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "server adress")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080", "host")

	// Парсинг флагов
	flag.Parse()
	fmt.Printf("Инициализация конфигурации: ServerAddress=%s\n", cfg.ServerAddress)

	// Логирование полученных значений флагов
	fmt.Printf("Инициализация конфигурации: BaseURL=%s\n", cfg.BaseURL)

	return cfg
}

// Validate проверяет корректность конфигурации
func (cfg *Config) Validate() error {
	if cfg.ServerAddress == "" {
		return fmt.Errorf("адрес сервера не может быть пустым")
	}
	if cfg.BaseURL == "" {
		return fmt.Errorf("базовый URL не может быть пустым")
	}
	return nil
}
