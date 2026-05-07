package config

import (
	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Postgres PostgresConfig
	MinIO    MinIOConfig
	OTel     OTelConfig
	Doorman  DoormanConfig
	Image    ImageConfig
}

type AppConfig struct {
	Port int `env:"APP_PORT" envDefault:"80"`
}

type PostgresConfig struct {
	DSN string `env:"POSTGRES_DSN,required"`
}

type MinIOConfig struct {
	Endpoint  string `env:"MINIO_ENDPOINT,required"`
	AccessKey string `env:"MINIO_ROOT_USER,required"`
	SecretKey string `env:"MINIO_ROOT_PASSWORD,required"`
	UseSSL    bool   `env:"MINIO_USE_SSL" envDefault:"false"`
	Bucket    string `env:"MINIO_BUCKET" envDefault:"edium-images"`
}

type OTelConfig struct {
	Endpoint    string `env:"OTEL_ENDPOINT" envDefault:"jaeger:4317"`
	ServiceName string `env:"OTEL_SERVICE_NAME" envDefault:"louvre"`
}

type DoormanConfig struct {
	JWKSEndpoint string `env:"DOORMAN_JWKS_URL" envDefault:"http://doorman/doorman/v1/.well-known/jwks.json"`
}

type ImageConfig struct {
	MaxFileSize       int64  `env:"IMAGE_MAX_SIZE" envDefault:"10485760"`
	MaxWidth          int    `env:"IMAGE_MAX_WIDTH" envDefault:"1920"`
	MaxHeight         int    `env:"IMAGE_MAX_HEIGHT" envDefault:"1080"`
	AllowedMimeTypes  string `env:"IMAGE_ALLOWED_TYPES" envDefault:"image/jpeg,image/png,image/webp"`
	MaxUploadsPerHour int    `env:"IMAGE_MAX_UPLOADS_PER_HOUR" envDefault:"100"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
