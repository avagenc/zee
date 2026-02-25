package config

import "time"

type Config struct {
	App    *App
	Server *Server

	Zep    *Zep
	ZeeAPI *ZeeAPI
	Groq   *Groq
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
}

type ZeeAPI struct {
	URL string `env:"ZEE_API_URL" env-required:"true"`
}

type Groq struct {
	APIKey  string `env:"GROQ_API_KEY" env-required:"true"`
	BaseURL string `env:"GROQ_BASE_URL"`
}
