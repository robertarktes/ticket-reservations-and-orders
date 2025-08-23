package config

import (
	"time"

	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	CRDBDSN      string
	MongoURI     string
	RedisAddr    string
	RabbitURL    string
	JWTPublicKey string
	HoldTTL      time.Duration
	OTLPEndpoint string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	holdTTL, _ := time.ParseDuration(os.Getenv("HOLD_TTL"))
	if holdTTL == 0 {
		holdTTL = 5 * time.Minute
	}

	return &Config{
		CRDBDSN:      os.Getenv("CRDB_DSN"),
		MongoURI:     os.Getenv("MONGO_URI"),
		RedisAddr:    os.Getenv("REDIS_ADDR"),
		RabbitURL:    os.Getenv("RABBIT_URL"),
		JWTPublicKey: os.Getenv("JWT_PUBLIC_KEY"),
		HoldTTL:      holdTTL,
		OTLPEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}, nil
}
