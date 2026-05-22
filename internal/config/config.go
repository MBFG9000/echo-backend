package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env           string
	OnionAddress  string
	PublicAppURL  string
	Server        Server
	DB            DB
	Redis         Redis
	JWT           JWT
	CORS          CORS
	Moderation    Moderation
}

type Server struct {
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	Host              string
	Port              string
	ShutdownTimeout   time.Duration
	RateLimitRequests int64
	RateLimitWindow   time.Duration
}

type DB struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type Redis struct {
	Addr     string
	Password string
	DB       int
}

type JWT struct {
	Secret string
	TTL    time.Duration
}

type CORS struct {
	AllowedOrigins []string
}

type Moderation struct {
	AutoHideThreshold int
}

func Load() (Config, error) {
	_ = godotenv.Load()
	var errs []error
	var cfg Config

	cfg.Env = strings.TrimSpace(os.Getenv("APP_ENV"))
	if cfg.Env == "" {
		cfg.Env = "development"
	}
	if !strings.EqualFold(cfg.Env, "production") {
		cfg.Env = "development"
	} else {
		cfg.Env = "production"
	}

	cfg.OnionAddress = strings.TrimSpace(os.Getenv("ONION_ADDRESS"))
	cfg.PublicAppURL = strings.TrimRight(strings.TrimSpace(EnvOrDefault("APP_PUBLIC_URL", "http://localhost:5173")), "/")

	//Server
	cfg.Server.Host = RequireEnv(&errs, "SERVER_HOST", false)
	cfg.Server.Port = RequireEnv(&errs, "SERVER_PORT", false)
	cfg.Server.ReadTimeout = RequireDuration(&errs, "SERVER_READ_TIMEOUT", false)
	cfg.Server.WriteTimeout = RequireDuration(&errs, "SERVER_WRITE_TIMEOUT", false)
	cfg.Server.ShutdownTimeout = RequireDuration(&errs, "SERVER_SHUTDOWN_TIMEOUT", false)
	cfg.Server.RateLimitRequests = RequireInt64(&errs, "SERVER_RATE_LIMIT_REQUESTS", false)
	cfg.Server.RateLimitWindow = RequireDuration(&errs, "SERVER_RATE_LIMIT_WINDOW", false)

	// DB
	cfg.DB.Host = RequireEnv(&errs, "DB_HOST", false)
	cfg.DB.Port = RequireEnv(&errs, "DB_PORT", false)
	cfg.DB.User = RequireEnv(&errs, "DB_USER", false)
	cfg.DB.Password = RequireEnv(&errs, "DB_PASSWORD", true)
	cfg.DB.Name = RequireEnv(&errs, "DB_NAME", false)
	cfg.DB.SSLMode = RequireEnv(&errs, "DB_SSL_MODE", true)

	cfg.Redis.Addr = RequireEnv(&errs, "REDIS_ADDR", false)
	cfg.Redis.Password = RequireEnv(&errs, "REDIS_PASSWORD", true)
	cfg.Redis.DB = RequireInt(&errs, "REDIS_DB", false)

	// JWT
	cfg.JWT.Secret = RequireEnv(&errs, "JWT_SECRET", false)
	cfg.JWT.TTL = RequireDuration(&errs, "JWT_TTL", false)

	// Moderation
	cfg.Moderation.AutoHideThreshold = RequireInt(&errs, "MODERATION_AUTO_HIDE_THRESHOLD", false)

	corsAllowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if strings.TrimSpace(corsAllowedOrigins) == "" {
		corsAllowedOrigins = "*"
	}
	cfg.CORS.AllowedOrigins = splitAndTrim(corsAllowedOrigins)

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}

	return cfg, nil
}

func (s Server) Address() string {
	return s.Host + ":" + s.Port
}

func (d DB) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode)
}

// Try to get enviroment variable by specified key, if error occur write
// errors to errs array, allowEmpty defines is empty string causes error or not
func RequireEnv(errs *[]error, key string, allowEmpty bool) string {
	value, ok := os.LookupEnv(key)

	if !ok {
		*errs = append(*errs, fmt.Errorf("Env variable %s does not exist", key))
		return ""
	}
	if value == "" && !allowEmpty {
		*errs = append(*errs, fmt.Errorf("Env variable %s is empty", key))
		return ""
	}

	return value
}

// return specified default value if enviroment variable
// with specified key does not exist or empty
func EnvOrDefault(key string, defaultValue string) string {
	value, ok := os.LookupEnv(key)

	if ok && value != "" {
		return value
	}

	return defaultValue
}

// Work as RequireEnv methods with additional check for integer
func RequireInt(errs *[]error, key string, allowEmpty bool) int {
	valueString := RequireEnv(errs, key, allowEmpty)
	if valueString == "" {
		return 0
	}

	parsed, err := strconv.Atoi(valueString)

	if err != nil {
		*errs = append(*errs, fmt.Errorf("cant parse %s to int, Error: %w", valueString, err))
		return 0
	}

	return parsed
}

// Work as RequireEnv methods with additional check for 64 bit integer
func RequireInt64(errs *[]error, key string, allowEmpty bool) int64 {
	valueString := RequireEnv(errs, key, allowEmpty)
	if valueString == "" {
		return 0
	}

	parsed, err := strconv.ParseInt(valueString, 10, 64)

	if err != nil {
		*errs = append(*errs, fmt.Errorf("cant parse %s to int64, Error: %w", valueString, err))
		return 0
	}

	return parsed
}

// Work as RequireEnv methods with additional check for Time.Duration which
// is under the hood actually 64 bit integer
func RequireDuration(errs *[]error, key string, allowEmpty bool) time.Duration {
	valueString := RequireEnv(errs, key, allowEmpty)

	if valueString == "" {
		return 0
	}

	parsed, err := time.ParseDuration(valueString)

	if err != nil {
		*errs = append(*errs, fmt.Errorf("cant parse %s to Time.Duration, Error: %w", valueString, err))
		return 0
	}
	return parsed
}

func splitAndTrim(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

func (c Config) IsProduction() bool {
	return c.Env == "production"
}
