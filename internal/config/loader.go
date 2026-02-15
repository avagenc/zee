package config

import (
	"fmt"
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("Info: .env file not found, using system environment variables")
	}

	cfg := &Config{
		App: &App{
			Name:    "zee-api",
			Version: "v0.3.0",
		},
		Server: &Server{
			Port:         "8080",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		Security: &Security{},
		Tuya:     &Tuya{},
		Database: &Database{
			MaxConns:        20,
			MinConns:        0,
			MaxConnLifetime: 1 * time.Hour,
			MaxConnIdleTime: 30 * time.Minute,
		},
	}

	if err := cleanenv.ReadEnv(cfg.App); err != nil {
		return nil, fmt.Errorf("failed to load app config: %w", err)
	}

	if err := cleanenv.ReadEnv(cfg.Server); err != nil {
		return nil, fmt.Errorf("failed to load server config: %w", err)
	}

	if err := cleanenv.ReadEnv(cfg.Security); err != nil {
		return nil, fmt.Errorf("failed to load security config: %w", err)
	}

	if err := cleanenv.ReadEnv(cfg.Tuya); err != nil {
		return nil, fmt.Errorf("failed to load tuya config: %w", err)
	}

	if err := cleanenv.ReadEnv(cfg.Database); err != nil {
		return nil, fmt.Errorf("failed to load database config: %w", err)
	}

	return cfg, nil
}
