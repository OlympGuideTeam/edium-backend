package config

import (
	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
}

type AppConfig struct {
	Port int `env:"APP_PORT" envDefault:"80"`
}

type PostgresConfig struct {
	DB       string `env:"POSTGRES_DB,required"`
	User     string `env:"POSTGRES_USER,required"`
	Password string `env:"POSTGRES_PASSWORD,required"`
	Port     int    `env:"POSTGRES_PORT" envDefault:"5432"`
	Host     string `env:"POSTGRES_HOST" envDefault:"doorman-db"`
}

type RedisConfig struct {
	Password string `env:"REDIS_PASSWORD,required"`
	Port     int    `env:"REDIS_PORT" envDefault:"6379"`
	Host     string `env:"REDIS_HOST" envDefault:"doorman-redis"`
}

type KafkaConfig struct {
	Brokers  []string `env:"KAFKA_BROKERS" envDefault:"kafka:9092"`
	ClientID string   `env:"KAFKA_CLIENT_ID" envDefault:"doorman"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
