package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config содержит настройки, общие для компонентов приложения
type Config struct {
	HTTP        HTTPConfig      `mapstructure:"http"`
	Frontend    FrontendConfig  `mapstructure:"frontend"`
	Cookie      CookieConfig    `mapstructure:"cookie"`
	Media       MediaConfig     `mapstructure:"media"`
	Logging     LoggingConfig   `mapstructure:"logging"`
	DatabaseURL string          `mapstructure:"database_url"`
	SMTP        SMTPConfig      `mapstructure:"smtp"`
	Local       LocalConfig     `mapstructure:"local"`
	Redis       RedisConfig     `mapstructure:"redis"`
	Auth        AuthConfig      `mapstructure:"auth"`
	User        UserConfig      `mapstructure:"user"`
	Posts       PostsConfig     `mapstructure:"posts"`
	Community   CommunityConfig `mapstructure:"community"`
}

// HTTPConfig содержит настройки HTTP-сервера
type HTTPConfig struct {
	Addr string `mapstructure:"addr"`
}

// FrontendConfig содержит адреса frontend для CORS и ссылок
type FrontendConfig struct {
	Origin string `mapstructure:"origin"`
	URL    string `mapstructure:"url"`
}

// CookieConfig содержит публичные настройки cookie
type CookieConfig struct {
	Secure   bool   `mapstructure:"secure"`
	SameSite string `mapstructure:"same_site"`
}

// MediaConfig содержит путь к файлам и публичный URL media
type MediaConfig struct {
	Dir       string `mapstructure:"dir"`
	PublicURL string `mapstructure:"public_url"`
}

// LoggingConfig содержит режим логирования
type LoggingConfig struct {
	Mode string `mapstructure:"mode"`
}

// SMTPConfig содержит настройки SMTP и URL логотипа письма
type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	FromName string `mapstructure:"from_name"`
	LogoURL  string `mapstructure:"logo_url"`
}

// LocalConfig содержит переопределения настроек для локального запуска
type LocalConfig struct {
	IsLocal       bool            `mapstructure:"is_local"`
	FrontendLocal FrontendConfig  `mapstructure:"frontend_local"`
	SMTP          LocalSMTPConfig `mapstructure:"smtp"`
}

// LocalSMTPConfig содержит локальные настройки SMTP без секретов
type LocalSMTPConfig struct {
	LogoURL string `mapstructure:"logo_url"`
}

// RedisConfig содержит настройки Redis для хранения сессий.
type RedisConfig struct {
	Addr       string `mapstructure:"addr"`
	DB         int    `mapstructure:"db"`
	SessionTTL string `mapstructure:"session_ttl"`
	Password   string `mapstructure:"password"`
}

type AuthConfig struct {
	HTTPAddr string `mapstructure:"http_addr"`
	GRPCAddr string `mapstructure:"grpc_addr"`
}

type UserConfig struct {
	HTTPAddr string `mapstructure:"http_addr"`
	GRPCAddr string `mapstructure:"grpc_addr"`
}

type PostsConfig struct {
	HTTPAddr string `mapstructure:"http_addr"`
}

type CommunityConfig struct {
	HTTPAddr string `mapstructure:"http_addr"`
	GRPCAddr string `mapstructure:"grpc_addr"`
}

// TTLDuration возвращает срок жизни сессии в формате time.Duration.
func (c RedisConfig) TTLDuration() (time.Duration, error) {
	return time.ParseDuration(c.SessionTTL)
}

// LogConfig содержит безопасное представление настроек для стартового лога
type LogConfig struct {
	HTTP        HTTPConfig     `json:"http"`
	Frontend    FrontendConfig `json:"frontend"`
	Cookie      CookieConfig   `json:"cookie"`
	Media       MediaConfig    `json:"media"`
	Logging     LoggingConfig  `json:"logging"`
	DatabaseURL string         `json:"database_url"`
	SMTP        SMTPLogConfig  `json:"smtp"`
	Local       LocalConfig    `json:"local"`
	Redis       RedisLogConfig `json:"redis"`
}

// SMTPLogConfig содержит SMTP-настройки
type SMTPLogConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	FromName string `json:"from_name"`
	LogoURL  string `json:"logo_url"`
}

// RedisLogConfig содержит безопасное представление настроек Redis.
type RedisLogConfig struct {
	Addr       string `json:"addr"`
	DB         int    `json:"db"`
	SessionTTL string `json:"session_ttl"`
	Password   string `json:"password"`
}

// SafeForLog возвращает настройки, безопасные для записи в лог
func (c Config) SafeForLog() LogConfig {
	databaseURL := ""
	if c.DatabaseURL != "" {
		databaseURL = "*hiden*"
	}
	username := ""
	if c.SMTP.Username != "" {
		username = "*hiden*"
	}
	password := ""
	if c.SMTP.Password != "" {
		password = "*hiden*"
	}
	return LogConfig{
		HTTP:        c.HTTP,
		Frontend:    c.Frontend,
		Cookie:      c.Cookie,
		Media:       c.Media,
		Logging:     c.Logging,
		DatabaseURL: databaseURL,
		SMTP: SMTPLogConfig{
			Host:     c.SMTP.Host,
			Port:     c.SMTP.Port,
			Username: username,
			Password: password,
			From:     c.SMTP.From,
			FromName: c.SMTP.FromName,
			LogoURL:  c.SMTP.LogoURL,
		},
		Local: c.Local,
		Redis: RedisLogConfig{
			Addr:       c.Redis.Addr,
			DB:         c.Redis.DB,
			SessionTTL: c.Redis.SessionTTL,
			Password:   redactIfSet(c.Redis.Password),
		},
	}
}

func redactIfSet(value string) string {
	if value == "" {
		return ""
	}
	return "*hiden*"
}

// Default возвращает настройки приложения по умолчанию
func Default() Config {
	return Config{
		HTTP:      HTTPConfig{Addr: "0.0.0.0:8080"},
		Frontend:  FrontendConfig{Origin: "http://localhost:8001", URL: "http://localhost:8001"},
		Cookie:    CookieConfig{Secure: false, SameSite: "lax"},
		Media:     MediaConfig{Dir: "/data/media", PublicURL: "/api/media"},
		Logging:   LoggingConfig{Mode: "prod"},
		Redis:     RedisConfig{Addr: "redis:6379", SessionTTL: "720h"},
		Auth:      AuthConfig{HTTPAddr: ":8081", GRPCAddr: ":9090"},
		User:      UserConfig{HTTPAddr: ":8082", GRPCAddr: ":9091"},
		Posts:     PostsConfig{HTTPAddr: ":8083"},
		Community: CommunityConfig{HTTPAddr: ":8084", GRPCAddr: ":9092"},
	}
}

const configPath = "./deploy/config.yml"

// Load загружает настройки из deploy/config.yml через Viper
func Load() (Config, error) {
	settings := viper.New()
	settings.SetConfigFile(configPath)
	if err := bindEnvironment(settings); err != nil {
		return Config{}, fmt.Errorf("bind environment: %w", err)
	}
	if err := settings.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", configPath, err)
	}

	cfg := Default()
	if err := settings.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config %s: %w", configPath, err)
	}
	cfg.DatabaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		settings.GetString("postgres.user"),
		settings.GetString("postgres.password"),
		settings.GetString("postgres.host"),
		settings.GetString("postgres.port"),
		settings.GetString("postgres.db"),
	)
	cfg.SMTP.Host = settings.GetString("smtp.host")
	cfg.SMTP.Port = settings.GetString("smtp.port")
	cfg.SMTP.Username = settings.GetString("smtp.username")
	cfg.SMTP.Password = settings.GetString("smtp.password")
	cfg.SMTP.From = settings.GetString("smtp.from")
	cfg.SMTP.FromName = settings.GetString("smtp.from_name")
	cfg.Redis.Password = settings.GetString("redis.password")
	if cfg.Local.IsLocal {
		if strings.TrimSpace(cfg.Local.FrontendLocal.Origin) != "" {
			cfg.Frontend.Origin = cfg.Local.FrontendLocal.Origin
		}
		if strings.TrimSpace(cfg.Local.FrontendLocal.URL) != "" {
			cfg.Frontend.URL = cfg.Local.FrontendLocal.URL
		}
		if strings.TrimSpace(cfg.Local.SMTP.LogoURL) != "" {
			cfg.SMTP.LogoURL = cfg.Local.SMTP.LogoURL
		}
	}
	if strings.TrimSpace(cfg.HTTP.Addr) == "" {
		return Config{}, errors.New("http.addr is required")
	}
	if strings.TrimSpace(cfg.Frontend.Origin) == "" {
		cfg.Frontend.Origin = cfg.Frontend.URL
	}
	if strings.TrimSpace(cfg.Media.Dir) == "" || strings.TrimSpace(cfg.Media.PublicURL) == "" {
		return Config{}, errors.New("media.dir and media.public_url are required")
	}
	if strings.TrimSpace(cfg.Logging.Mode) == "" {
		cfg.Logging.Mode = "prod"
	}
	return cfg, nil
}

// bindEnvironment связывает приватные переменные окружения с настройками Viper
func bindEnvironment(settings *viper.Viper) error {
	bindings := map[string]string{
		"postgres.db":       "POSTGRES_DB",
		"postgres.user":     "POSTGRES_USER",
		"postgres.password": "POSTGRES_PASSWORD",
		"postgres.host":     "POSTGRES_HOST",
		"postgres.port":     "POSTGRES_PORT",
		"smtp.host":         "SMTP_HOST",
		"smtp.port":         "SMTP_PORT",
		"smtp.username":     "SMTP_USERNAME",
		"smtp.password":     "SMTP_PASSWORD",
		"smtp.from":         "SMTP_FROM",
		"smtp.from_name":    "SMTP_FROM_NAME",
		"redis.password":    "REDIS_PASSWORD",
	}
	for key, environment := range bindings {
		if err := settings.BindEnv(key, environment); err != nil {
			return err
		}
	}
	return nil
}
