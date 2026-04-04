package config

import (
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

type Config struct {
	App        AppConfig
	Postgres   PostgresConfig
	Redis      RedisConfig
	NATS       NATSConfig
	OTel       OTelConfig
	DeepSeek   DeepSeekConfig
	ClickHouse ClickHouseConfig
	RateLimit  RateLimitConfig
}

type AppConfig struct {
	Port int `env:"APP_PORT" envDefault:"80"`
}

type PostgresConfig struct {
	DSN string `env:"POSTGRES_DSN,required"`
}

type RedisConfig struct {
	URL string `env:"REDIS_URL,required"`
}

type NATSConfig struct {
	URL string `env:"NATS_URL" envDefault:"nats://nats:4222"`
}

type OTelConfig struct {
	Endpoint    string `env:"OTEL_ENDPOINT" envDefault:"jaeger:4317"`
	ServiceName string `env:"OTEL_SERVICE_NAME" envDefault:"charon"`
}

type DeepSeekConfig struct {
	BaseURL string        `env:"DEEPSEEK_BASE_URL" envDefault:"https://api.deepseek.com"`
	APIKey  string        `env:"DEEPSEEK_API_KEY,required"`
	Timeout time.Duration `env:"DEEPSEEK_TIMEOUT" envDefault:"60s"`
}

type ClickHouseConfig struct {
	DSN string `env:"CLICKHOUSE_DSN" envDefault:""`
}

type RateLimitConfig struct {
	Riddler int `env:"RATE_LIMIT_RIDDLER" envDefault:"100"`
	Yoda    int `env:"RATE_LIMIT_YODA" envDefault:"50"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
