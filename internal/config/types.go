package config

import "time"

type Config struct {
	App    *App
	Server *Server

	Zep      *Zep
	Gemini   *Gemini
	Tuya     *Tuya
	Database *Database
}

type App struct {
	Name    string
	Version string
	Env     string `env:"APP_ENV" env-required:"true"`
}

type Server struct {
	Port         string        `env:"PORT"`
	ReadTimeout  time.Duration `env:"SERVER_READ_TIMEOUT"`
	WriteTimeout time.Duration `env:"SERVER_WRITE_TIMEOUT"`
	IdleTimeout  time.Duration `env:"SERVER_IDLE_TIMEOUT"`
}

type Zep struct {
	APIKey string `env:"ZEP_API_KEY" env-required:"true"`
	URL    string `env:"ZEP_API_URL"`
}

type Gemini struct {
	APIKey string `env:"GEMINI_API_KEY" env-required:"true"`
}

type Tuya struct {
	AccessID     string `env:"TUYA_ACCESS_ID"     env-required:"true"`
	AccessSecret string `env:"TUYA_ACCESS_SECRET" env-required:"true"`
	BaseURL      string `env:"TUYA_BASE_URL"      env-required:"true"`
}

type Database struct {
	URL             string        `env:"DATABASE_URL"                env-required:"true"`
	MaxConns        int32         `env:"DATABASE_MAX_CONNS"`
	MinConns        int32         `env:"DATABASE_MIN_CONNS"`
	MaxConnLifetime time.Duration `env:"DATABASE_MAX_CONN_LIFETIME"`
	MaxConnIdleTime time.Duration `env:"DATABASE_MAX_CONN_IDLE_TIME"`
}
