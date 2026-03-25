package config

import (
	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	Keys     KeysConfig
	NATS     NATSConfig
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

type KeysConfig struct {
	JwtRSAPrivateKey string `env:"JWT_RSA_PRIVATE_KEY"`
	JwtActiveKID     string `env:"JWT_ACTIVE_KID" envDefault:"key-1"`
}

type NATSConfig struct {
	URL string `env:"NATS_URL" envDefault:"nats://nats:4222"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
