package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server     Server
	DB         DB
	Redis      Redis
	JWT        JWT
	Moderation Moderation
}

type Server struct {
	Host              string
	Port              string
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
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

type Moderation struct {
	AutoHideThreshold int
}

func Load() (Config, error) {
	_ = godotenv.Load()

	serverHost, err := requiredString("SERVER_HOST", false)
	if err != nil {
		return Config{}, err
	}

	serverPort, err := requiredString("SERVER_PORT", false)
	if err != nil {
		return Config{}, err
	}

	dbHost, err := requiredString("DB_HOST", false)
	if err != nil {
		return Config{}, err
	}

	dbPort, err := requiredString("DB_PORT", false)
	if err != nil {
		return Config{}, err
	}

	dbUser, err := requiredString("DB_USER", false)
	if err != nil {
		return Config{}, err
	}

	dbPassword, err := requiredString("DB_PASSWORD", false)
	if err != nil {
		return Config{}, err
	}

	dbName, err := requiredString("DB_NAME", false)
	if err != nil {
		return Config{}, err
	}

	dbSSLMode, err := requiredString("DB_SSL_MODE", false)
	if err != nil {
		return Config{}, err
	}

	redisAddr, err := requiredString("REDIS_ADDR", false)
	if err != nil {
		return Config{}, err
	}

	redisPassword, err := requiredString("REDIS_PASSWORD", true)
	if err != nil {
		return Config{}, err
	}

	jwtSecret, err := requiredString("JWT_SECRET", false)
	if err != nil {
		return Config{}, err
	}

	readTimeout, err := mustDuration("SERVER_READ_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	writeTimeout, err := mustDuration("SERVER_WRITE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := mustDuration("SERVER_SHUTDOWN_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	rateLimitWindow, err := mustDuration("SERVER_RATE_LIMIT_WINDOW")
	if err != nil {
		return Config{}, err
	}

	rateLimitRequests, err := mustInt64("SERVER_RATE_LIMIT_REQUESTS")
	if err != nil {
		return Config{}, err
	}

	redisDB, err := mustInt("REDIS_DB")
	if err != nil {
		return Config{}, err
	}

	jwtTTL, err := mustDuration("JWT_TTL")
	if err != nil {
		return Config{}, err
	}

	autoHideThreshold, err := mustInt("MODERATION_AUTO_HIDE_THRESHOLD")
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Server: Server{
			Host:              serverHost,
			Port:              serverPort,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			ShutdownTimeout:   shutdownTimeout,
			RateLimitRequests: rateLimitRequests,
			RateLimitWindow:   rateLimitWindow,
		},
		DB: DB{
			Host:     dbHost,
			Port:     dbPort,
			User:     dbUser,
			Password: dbPassword,
			Name:     dbName,
			SSLMode:  dbSSLMode,
		},
		Redis: Redis{
			Addr:     redisAddr,
			Password: redisPassword,
			DB:       redisDB,
		},
		JWT: JWT{
			Secret: jwtSecret,
			TTL:    jwtTTL,
		},
		Moderation: Moderation{
			AutoHideThreshold: autoHideThreshold,
		},
	}

	return cfg, nil
}

func (s Server) Address() string {
	return s.Host + ":" + s.Port
}

func (d DB) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode)
}

func requiredString(key string, allowEmpty bool) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("missing env %s", key)
	}
	if value == "" && !allowEmpty {
		return "", fmt.Errorf("empty env %s", key)
	}
	return value, nil
}

func mustInt(key string) (int, error) {
	value, err := requiredString(key, false)
	if err != nil {
		return 0, err
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

func mustInt64(key string) (int64, error) {
	value, err := requiredString(key, false)
	if err != nil {
		return 0, err
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

func mustDuration(key string) (time.Duration, error) {
	value, err := requiredString(key, false)
	if err != nil {
		return 0, err
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}
