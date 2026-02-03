package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Tuya struct {
	AccessID     string `env:"TUYA_ACCESS_ID" env-required:"true"`
	AccessSecret string `env:"TUYA_ACCESS_SECRET" env-required:"true"`
	BaseURL      string `env:"TUYA_BASE_URL" env-required:"true"`
}

func LoadTuya() (*Tuya, error) {
	cfg := Tuya{}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("failed to load tuya config: %w", err)
	}

	return &cfg, nil
}
