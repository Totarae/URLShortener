package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/spf13/viper"
)

// Config хранит конфигурацию сервера
type Config struct {
	ServerAddress    string `json:"server_address"`
	BaseURL          string `json:"base_url"`
	FileStoragePath  string `json:"file_storage_path"`
	DatabaseDSN      string `json:"database_dsn"`
	PgMigrationsPath string `json:"pg_migrations_path"`
	EnableHTTPS      bool   `json:"enable_https"`
	TLSCertPath      string `json:"tls_cert_path"`
	TLSKeyPath       string `json:"tls_key_path"`
	Mode             string `json:"-"`
	TrustedSubnet    string `json:"trusted_subnet"`
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
	viper.SetDefault("TRUSTED_SUBNET", "")

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
	configPath := flag.String("c", "", "path to JSON config file")
	trustedSubnet := flag.String("t", "", "trusted subnet in CIDR format")
	flag.StringVar(configPath, "config", "", "path to JSON config file")

	flag.Parse()

	// Загружаем JSON-конфигурацию (если указана)
	if *configPath == "" {
		*configPath = os.Getenv("CONFIG")
	}

	type rawJSON Config
	jsonCfg := &rawJSON{}
	if *configPath != "" {
		data, err := os.ReadFile(*configPath)
		if err != nil {
			log.Printf("Не удалось прочитать JSON-файл конфигурации %q: %v", *configPath, err)
		} else if err := json.Unmarshal(data, jsonCfg); err != nil {
			log.Printf("Ошибка разбора JSON-файла конфигурации: %v", err)
		}
	}

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
		TrustedSubnet:    viper.GetString("TRUSTED_SUBNET"),
	}

	// Переопределяем значениями из переменных окружения (viper)
	override := func(env string, target *string) {
		if val := viper.GetString(env); val != "" {
			*target = val
		}
	}
	override("SERVER_ADDRESS", &cfg.ServerAddress)
	override("BASE_URL", &cfg.BaseURL)
	override("FILE_STORAGE_PATH", &cfg.FileStoragePath)
	override("DATABASE_DSN", &cfg.DatabaseDSN)
	override("PG_MIGRATIONS_PATH", &cfg.PgMigrationsPath)
	override("TLS_CERT_PATH", &cfg.TLSCertPath)
	override("TLS_KEY_PATH", &cfg.TLSKeyPath)
	override("TRUSTED_SUBNET", &cfg.TrustedSubnet)
	cfg.EnableHTTPS = viper.GetBool("ENABLE_HTTPS")

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

	if *trustedSubnet != "" {
		cfg.TrustedSubnet = *trustedSubnet
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
	log.Printf("Инициализация конфигурации: EnableHTTPS=%v", cfg.EnableHTTPS)

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
