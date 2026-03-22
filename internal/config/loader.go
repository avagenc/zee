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
			Name:    "zee-agent",
			Version: "v0.2.0",
		},
		Server: &Server{
			Port:         "8080",
			ReadTimeout:  16 * time.Second,
			WriteTimeout: 120 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		Zep:    &Zep{},
		ZeeAPI: &ZeeAPI{},
		Gemini: &Gemini{},
	}

	if err := cleanenv.ReadEnv(cfg.App); err != nil {
		return nil, fmt.Errorf("failed to load app config: %w", err)
	}

	if err := cleanenv.ReadEnv(cfg.Server); err != nil {
		return nil, fmt.Errorf("failed to load server config: %w", err)
	}

	if err := cleanenv.ReadEnv(cfg.Zep); err != nil {
		return nil, fmt.Errorf("failed to load zep config: %w", err)
	}

	if err := cleanenv.ReadEnv(cfg.ZeeAPI); err != nil {
		return nil, fmt.Errorf("failed to load zee config: %w", err)
	}

	if err := cleanenv.ReadEnv(cfg.Gemini); err != nil {
		return nil, fmt.Errorf("failed to load gemini config: %w", err)
	}

	return cfg, nil
}
