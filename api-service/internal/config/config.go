// Package config đọc cấu hình từ biến môi trường (và file .env nếu có).
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppName    string
	AppVersion string
	Port       string
	LogLevel   string

	DatabaseURL string

	JWTSecret   string
	TokenTTL    time.Duration
	AllowSignup bool

	// Admin đầu tiên được tạo khi DB chưa có admin nào.
	AdminEmail    string
	AdminPassword string
	AdminName     string

	MLServiceURL   string
	MLServiceToken string
	MLTimeout      time.Duration

	// URL của web-app, dùng để tạo link trong email (mời thành viên, reset mật khẩu)
	WebURL string
	// Để trống = không đẩy sự kiện sang notification service (sự kiện vẫn nằm trong outbox)
	RabbitMQURL string
	// Số worker chạy job phân tích cả project
	JobWorkers int

	CORSAllowedOrigins []string
	// TrustProxy: lấy IP client từ X-Real-IP (khi chạy sau nginx của web-app).
	TrustProxy bool
}

func Load() (Config, error) {
	loadDotEnv(".env")

	var err error
	cfg := Config{
		AppName:            env("APP_NAME", "ReqSentinel API"),
		AppVersion:         env("APP_VERSION", "0.2.0"),
		Port:               env("PORT", "8000"),
		LogLevel:           env("LOG_LEVEL", "INFO"),
		DatabaseURL:        normalizeDatabaseURL(env("DATABASE_URL", "")),
		JWTSecret:          env("JWT_SECRET", ""),
		AllowSignup:        envBool("ALLOW_SIGNUP", true),
		AdminEmail:         env("ADMIN_EMAIL", ""),
		AdminPassword:      env("ADMIN_PASSWORD", ""),
		AdminName:          env("ADMIN_NAME", "Administrator"),
		MLServiceURL:       strings.TrimRight(env("ML_SERVICE_URL", "http://localhost:8001"), "/"),
		MLServiceToken:     env("ML_SERVICE_TOKEN", ""),
		CORSAllowedOrigins: splitList(env("CORS_ALLOWED_ORIGINS", "")),
		TrustProxy:         envBool("TRUST_PROXY", false),
		WebURL:             strings.TrimRight(env("WEB_URL", "http://localhost:5173"), "/"),
		RabbitMQURL:        env("RABBITMQ_URL", ""),
	}
	if cfg.JobWorkers, err = strconv.Atoi(env("JOB_WORKERS", "1")); err != nil || cfg.JobWorkers < 1 {
		return cfg, errors.New("JOB_WORKERS must be a positive integer")
	}

	if cfg.TokenTTL, err = time.ParseDuration(env("TOKEN_TTL", "24h")); err != nil {
		return cfg, fmt.Errorf("TOKEN_TTL: %w", err)
	}
	if cfg.MLTimeout, err = time.ParseDuration(env("ML_TIMEOUT", "120s")); err != nil {
		return cfg, fmt.Errorf("ML_TIMEOUT: %w", err)
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("DATABASE_URL is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return cfg, errors.New("JWT_SECRET is required and must be at least 32 characters")
	}
	return cfg, nil
}

// normalizeDatabaseURL cho phép dùng lại DATABASE_URL kiểu SQLAlchemy
// (postgresql+asyncpg://...) từ bản Python cũ.
func normalizeDatabaseURL(u string) string {
	for _, prefix := range []string{"postgresql+asyncpg://", "postgresql+psycopg2://", "postgresql+psycopg://"} {
		if strings.HasPrefix(u, prefix) {
			return "postgres://" + strings.TrimPrefix(u, prefix)
		}
	}
	return u
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// loadDotEnv nạp KEY=VALUE từ file; không ghi đè biến đã có trong môi trường.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
}
