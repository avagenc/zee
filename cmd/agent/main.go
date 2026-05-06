package main

import (
	"context"
	"log"
	"net/http"

	"github.com/avagenc/zee-agent/internal/chat"
	"github.com/avagenc/zee-agent/internal/config"
	"github.com/avagenc/zee-agent/internal/system"
	"github.com/avagenc/zee-agent/internal/tools"
	"github.com/avagenc/zee-agent/internal/zee"
	"github.com/avagenc/zee-agent/internal/zeeapi"

	"github.com/getzep/zep-go/v3/client"
	"github.com/getzep/zep-go/v3/option"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"google.golang.org/api/idtoken"
	"google.golang.org/genai"

	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/tool"

	zepadk "go.naturallyfunny.dev/adk/zep"
	"go.naturallyfunny.dev/api/identity"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL: %v", err)
	}

	ctx := context.Background()

	model, err := gemini.NewModel(ctx, "gemini-3-flash-preview", &genai.ClientConfig{
		APIKey: cfg.Gemini.APIKey,
	})
	if err != nil {
		log.Fatalf("Failed to assign Gemini model: %v", err)
	}

	oidcClient, err := idtoken.NewClient(ctx, cfg.ZeeAPI.URL)
	if err != nil {
		log.Fatalf("FATAL: Failed to create OIDC client for zee-api: %v", err)
	}

	zeeAPIClient := zeeapi.NewClient(cfg.ZeeAPI.URL, oidcClient)

	t, err := tools.Load(zeeAPIClient)
	if err != nil {
		log.Fatalf("Failed to load tools: %v", err)
	}

	tuyaTools := []tool.Tool{
		t.GetAccount,
		t.ListDevices,
		t.SendCommandsToADevice,
	}

	zeeForUser, err := llmagent.New(llmagent.Config{
		Name:        zee.Name,
		Model:       model,
		Tools:       tuyaTools,
		Description: "Tuya Smart Home Agent — direct user channel",
		Instruction: zee.SystemInstruction(),
	})
	if err != nil {
		log.Fatalf("Failed to create user-channel agent: %v", err)
	}

	zeeForAva, err := llmagent.New(llmagent.Config{
		Name:        zee.Name,
		Model:       model,
		Tools:       tuyaTools,
		Description: "Tuya Smart Home Agent — Ava orchestrator channel",
		Instruction: zee.SystemInstruction(zee.ForAva()),
	})
	if err != nil {
		log.Fatalf("Failed to create ava-channel agent: %v", err)
	}

	zepClient := client.NewClient(option.WithAPIKey(cfg.Zep.APIKey))

	sessionService := zepadk.NewSessionService(
		zepClient,
		zee.Name,
		zepadk.WithConversationHistory(6),
		zepadk.WithKnowledgeContext(nil),
		zepadk.WithUserDisplayName("human"),
	)

	userRunner, err := runner.New(runner.Config{
		AppName:           cfg.App.Name,
		Agent:             zeeForUser,
		SessionService:    sessionService,
		AutoCreateSession: true,
	})
	if err != nil {
		log.Fatalf("Failed to create user-channel runner: %v", err)
	}

	avaRunner, err := runner.New(runner.Config{
		AppName:           cfg.App.Name,
		Agent:             zeeForAva,
		SessionService:    sessionService,
		AutoCreateSession: true,
	})
	if err != nil {
		log.Fatalf("Failed to create ava-channel runner: %v", err)
	}

	sys := system.NewHandler(cfg.App.Name, cfg.App.Version, cfg.App.Env)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", sys.Index)

	r.Group(func(r chi.Router) {
		r.Use(identity.WithUserID)

		r.Post("/chat", chat.Handle(userRunner))
		r.Post("/chat/ava", chat.Handle(avaRunner))
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
