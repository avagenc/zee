package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	AvagencAPIKey    string
	TuyaAccessID     string
	TuyaAccessSecret string
	TuyaBaseURL      string
}

func NewConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found. Reading credentials from environment.")
	}

	cfg := &Config{
		Port:             os.Getenv("PORT"),
		AvagencAPIKey:    os.Getenv("AVAGENC_API_KEY"),
		TuyaAccessID:     os.Getenv("TUYA_ACCESS_ID"),
		TuyaAccessSecret: os.Getenv("TUYA_ACCESS_SECRET"),
		TuyaBaseURL:      os.Getenv("TUYA_BASE_URL"),
	}

	if cfg.Port == "" {
		log.Println("Port not configured, defaulting to 8080")
		cfg.Port = "8080"
	}

	if cfg.AvagencAPIKey == "" || cfg.TuyaAccessID == "" || cfg.TuyaAccessSecret == "" || cfg.TuyaBaseURL == "" {
		return nil, fmt.Errorf("one or more required environment variables are not set")
	}

	return cfg, nil
}
