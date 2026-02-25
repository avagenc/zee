package main

import (
	"log"
	"net/http"
	"time"

	"github.com/avagenc/zee-agent/internal/chat"
	"github.com/avagenc/zee-agent/internal/config"
	"github.com/avagenc/zee-agent/internal/middleware"
	"github.com/avagenc/zee-agent/system"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/sashabaranov/go-openai"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/server/adkrest"
	"google.golang.org/adk/session"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL: %v", err)
	}

	if cfg.Groq == nil || cfg.Groq.APIKey == "" {
		log.Fatalf("FATAL: GROQ_API_KEY environment variable is required")
	}

	groqCfg := openai.DefaultConfig(cfg.Groq.APIKey)
	groqCfg.BaseURL = cfg.Groq.BaseURL
	groqClient := openai.NewClientWithConfig(groqCfg)

	model := chat.NewModel(groqClient, "llama-3.3-70b-versatile")

	a, err := llmagent.New(llmagent.Config{
		Name:        "Zee",
		Model:       model,
		Description: "A simple baseline agent powered by Groq",
		Instruction: "You are a helpful, smart AI assistant. Answer the user's questions clearly.",
	})
	if err != nil {
		log.Fatalf("FATAL: Failed to create agent: %v", err)
	}

	adkCfg := &launcher.Config{
		AgentLoader:    agent.NewSingleLoader(a),
		SessionService: session.InMemoryService(),
	}

	hdl := struct {
		system *system.Handler
		chat   http.Handler
	}{
		system: system.NewHandler(cfg.App.Name, cfg.App.Version, cfg.App.Env),
		chat:   adkrest.NewHandler(adkCfg, 120*time.Second),
	}

	r := chi.NewRouter()

	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	r.Get("/", hdl.system.Index)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireUserIdentity)

		r.Mount("/chat", http.StripPrefix("/chat", hdl.chat))
	})

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	log.Printf("In the name of Allah, The Most Compassionate, The Most Merciful")
	log.Printf("Starting %s (%s) on port %s\n", cfg.App.Name, cfg.App.Version, cfg.Server.Port)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("FATAL: Failed to start API: %v", err)
	}
}
