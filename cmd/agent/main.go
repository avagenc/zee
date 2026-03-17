package main

import (
	"log"
	"net/http"

	"github.com/avagenc/zee-agent/internal/chat"
	"github.com/avagenc/zee-agent/internal/config"
	"github.com/avagenc/zee-agent/internal/identity"
	"github.com/avagenc/zee-agent/internal/system"
	zepclient "github.com/getzep/zep-go/v3/client"
	"github.com/getzep/zep-go/v3/option"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/sashabaranov/go-openai"

	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/runner"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL: %v", err)
	}

	openaiConfig := openai.DefaultConfig(cfg.Groq.APIKey)
	openaiConfig.BaseURL = cfg.Groq.BaseURL
	openaiClient := openai.NewClientWithConfig(openaiConfig)

	model := chat.NewModel(openaiClient, "openai/gpt-oss-20b")

	agent, err := llmagent.New(llmagent.Config{
		Name:        "Zee",
		Model:       model,
		Description: "Starter agent",
		Instruction: "You are a simple llm chatbot.",
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	zepOpts := []option.RequestOption{
		option.WithAPIKey(cfg.Zep.APIKey),
	}
	if cfg.Zep.URL != "" {
		zepOpts = append(zepOpts, option.WithBaseURL(cfg.Zep.URL))
	}
	zepClient := zepclient.NewClient(zepOpts...)

	sessionService := chat.NewZepSessionService(zepClient)

	rnr, err := runner.New(runner.Config{
		AppName:        cfg.App.Name,
		Agent:          agent,
		SessionService: sessionService,
	})
	if err != nil {
		log.Fatalf("Failed to create agent runner: %v", err)
	}

	h := struct {
		system *system.Handler
		chat   *chat.Handler
	}{
		system: system.NewHandler(cfg.App.Name, cfg.App.Version, cfg.App.Env),
		chat:   chat.NewHandler(rnr, zepClient, sessionService),
	}

	r := chi.NewRouter()

	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	r.Get("/", h.system.Index)

	r.Group(func(r chi.Router) {
		r.Use(identity.RequireUserID)

		r.Post("/chat", h.chat.HandleChat)
	})

	s := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	log.Printf("In the name of Allah, The Most Compassionate, The Most Merciful")
	log.Printf("Starting %s (%s) on port %s\n", cfg.App.Name, cfg.App.Version, cfg.Server.Port)

	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("FATAL: Failed to start API: %v", err)
	}
}
