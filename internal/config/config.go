package config

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/spf13/viper"
)

// Config хранит конфигурацию сервера
type Config struct {
	ServerAddress    string
	BaseURL          string
	FileStoragePath  string
	DatabaseDSN      string
	PgMigrationsPath string
	Mode             string
	EnableHTTPS      bool
	TLSCertPath      string
	TLSKeyPath       string
}

// NewConfig инициализирует конфигурацию на основе аргументов командной строки
func NewConfig() *Config {

	viper.SetDefault("SERVER_ADDRESS", "localhost:8080") // Значения по умолчанию
	viper.SetDefault("BASE_URL", "http://localhost:8080")
	viper.SetDefault("FILE_STORAGE_PATH", "data.json")
	viper.SetDefault("DATABASE_DSN", "")
	viper.SetDefault("PG_MIGRATIONS_PATH", "internal/migrations")
	viper.SetDefault("ENABLE_HTTPS", false)
	viper.SetDefault("TLS_CERT_PATH", "cert.pem")
	viper.SetDefault("TLS_KEY_PATH", "key.pem")

	viper.AutomaticEnv()

	// Читаем .env, если есть (не переопределяет переменные окружения!)
	viper.SetConfigFile(".env")
	_ = viper.ReadInConfig() // Ошибку игнорируем, если файла нет

	// Определяем флаги, но НЕ задаем в них значения по умолчанию
	serverAddress := flag.String("a", "", "server address")
	baseURL := flag.String("b", "", "base URL")
	fileStoragePath := flag.String("f", "", "file storage path (JSON file)")
	databaseDSN := flag.String("d", "", "PostgreSQL DSN")
	enableHTTPS := flag.Bool("s", false, "enable HTTPS")
	tlsCertPath := flag.String("cert", "", "path to TLS certificate")
	tlsKeyPath := flag.String("key", "", "path to TLS key")

	flag.Parse()

	// Если переменные окружения заданы — они имеют высший приоритет
	cfg := &Config{
		ServerAddress:    viper.GetString("SERVER_ADDRESS"),
		BaseURL:          viper.GetString("BASE_URL"),
		FileStoragePath:  viper.GetString("FILE_STORAGE_PATH"),
		DatabaseDSN:      viper.GetString("DATABASE_DSN"),
		PgMigrationsPath: viper.GetString("PG_MIGRATIONS_PATH"),
		EnableHTTPS:      viper.GetBool("ENABLE_HTTPS"),
		TLSCertPath:      viper.GetString("TLS_CERT_PATH"),
		TLSKeyPath:       viper.GetString("TLS_KEY_PATH"),
	}

	// Если флаг передан, но переменной окружения нет — используем флаг
	if *serverAddress != "" {
		cfg.ServerAddress = *serverAddress
	}
	if *baseURL != "" {
		cfg.BaseURL = *baseURL
	}
	if *fileStoragePath != "" {
		cfg.FileStoragePath = *fileStoragePath
	}
	if *databaseDSN != "" {
		cfg.DatabaseDSN = *databaseDSN
		os.Setenv("DATABASE_DSN", cfg.DatabaseDSN)
	}

	// Определяем режим работы
	if cfg.DatabaseDSN != "" {
		cfg.Mode = "database"
	} else if cfg.FileStoragePath != "" {
		cfg.Mode = "file"
	} else {
		cfg.Mode = "in-memory"
	}

	// Включаем TLS
	if *enableHTTPS {
		cfg.EnableHTTPS = true
	}
	if *tlsCertPath != "" {
		cfg.TLSCertPath = *tlsCertPath
	}
	if *tlsKeyPath != "" {
		cfg.TLSKeyPath = *tlsKeyPath
	}

	log.Printf("Инициализация конфигурации: ServerAddress=%s", cfg.ServerAddress)
	log.Printf("Инициализация конфигурации: BaseURL=%s", cfg.BaseURL)
	log.Printf("Инициализация конфигурации: FileStoragePath=%s", cfg.FileStoragePath)
	log.Printf("Инициализация конфигурации: DatabaseDSN=%s", cfg.DatabaseDSN)
	log.Printf("Инициализация конфигурации: PgMigrationsPath=%s", cfg.PgMigrationsPath)
	log.Printf("Инициализация конфигурации: Mode=%s", cfg.Mode)
	// Проверка корректности конфигурации
	if err := cfg.Validate(); err != nil {
		fmt.Printf("Ошибка конфигурации: %v\n", err)
	}

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
	if cfg.FileStoragePath == "" {
		return fmt.Errorf("путь к файлу хранилища не может быть пустым")
	}
	/*	if cfg.DatabaseDSN == "" || cfg.PgMigrationsPath == "" {
		return fmt.Errorf("адрес подключения к БД не может быть пустым")
	}*/
	return nil
}
