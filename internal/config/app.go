package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type App struct {
	Name    string
	Version string
	Env     string `env:"APP_ENV" env-required:"true"`
}

func LoadApp() (*App, error) {
	cfg := App{
		Name:    "zee-api",
		Version: "v0.2.0",
	}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("failed to load app config: %w", err)
	}

	return &cfg, nil
}
