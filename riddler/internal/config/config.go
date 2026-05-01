package config

import (
	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	NATS     NATSConfig
	OTel     OTelConfig
	Doorman  DoormanConfig
}

type AppConfig struct {
	Port int `env:"APP_PORT" envDefault:"80"`
}

type PostgresConfig struct {
	DSN string `env:"POSTGRES_DSN,required"`
}

type NATSConfig struct {
	URL string `env:"NATS_URL" envDefault:"nats://nats:4222"`
}

type OTelConfig struct {
	Endpoint    string `env:"OTEL_ENDPOINT" envDefault:"jaeger:4317"`
	ServiceName string `env:"OTEL_SERVICE_NAME" envDefault:"riddler"`
}

type RedisConfig struct {
	URL string `env:"REDIS_URL,required"`
}

type DoormanConfig struct {
	JWKSEndpoint string `env:"DOORMAN_JWKS_URL" envDefault:"http://doorman/api/v1/.well-known/jwks.json"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
