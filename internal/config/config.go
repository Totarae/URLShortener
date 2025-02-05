package config

import (
	"flag"
	"fmt"
	"github.com/spf13/viper"
	"log"
)

// Config хранит конфигурацию сервера
type Config struct {
	ServerAddress string
	BaseURL       string
}

// NewConfig инициализирует конфигурацию на основе аргументов командной строки
func NewConfig() *Config {

	viper.SetDefault("SERVER_ADDRESS", "localhost:8080") // Значения по умолчанию
	viper.SetDefault("BASE_URL", "http://localhost:8080")

	viper.AutomaticEnv()

	// Читаем .env, если есть (не переопределяет переменные окружения!)
	viper.SetConfigFile(".env")
	_ = viper.ReadInConfig() // Ошибку игнорируем, если файла нет

	// Определяем флаги, но НЕ задаем в них значения по умолчанию
	serverAddress := flag.String("a", "", "server address")
	baseURL := flag.String("b", "", "base URL")

	flag.Parse()

	// Если переменные окружения заданы — они имеют высший приоритет
	cfg := &Config{
		ServerAddress: viper.GetString("SERVER_ADDRESS"),
		BaseURL:       viper.GetString("BASE_URL"),
	}

	// Если флаг передан, но переменной окружения нет — используем флаг
	if *serverAddress != "" {
		cfg.ServerAddress = *serverAddress
	}
	if *baseURL != "" {
		cfg.BaseURL = *baseURL
	}

	log.Printf("Инициализация конфигурации: ServerAddress=%s", cfg.ServerAddress)
	log.Printf("Инициализация конфигурации: BaseURL=%s", cfg.BaseURL)

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
