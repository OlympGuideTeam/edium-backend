package config

import (
	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Postgres PostgresConfig
	NATS     NATSConfig
	Telegram TelegramConfig
	VK       VKConfig
	OTel     OTelConfig
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

type TelegramConfig struct {
	BotToken string `env:"TELEGRAM_BOT_TOKEN,required"`
}

type VKConfig struct {
	GroupToken string `env:"VK_GROUP_TOKEN,required"`
	GroupID    int    `env:"VK_GROUP_ID,required"`
}

type OTelConfig struct {
	Endpoint    string `env:"OTEL_ENDPOINT" envDefault:"jaeger:4317"`
	ServiceName string `env:"OTEL_SERVICE_NAME" envDefault:"herald"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
