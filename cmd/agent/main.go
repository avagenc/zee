package main

import (
	"context"
	"log"
	"net/http"

	_ "time/tzdata"

	"github.com/avagenc/zee-agent/internal/account"
	"github.com/avagenc/zee-agent/internal/chat"
	"github.com/avagenc/zee-agent/internal/config"
	"github.com/avagenc/zee-agent/internal/device"
	"github.com/avagenc/zee-agent/internal/system"
	"github.com/avagenc/zee-agent/internal/tools"
	"github.com/avagenc/zee-agent/internal/tuya"
	"github.com/avagenc/zee-agent/internal/zee"
	"github.com/avagenc/zee-agent/internal/zeedb"

	"github.com/getzep/zep-go/v3/client"
	"github.com/getzep/zep-go/v3/option"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"google.golang.org/genai"

	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/tool"

	adksession "go.naturallyfunny.dev/adk/session"
	"go.naturallyfunny.dev/adk/zep"
	apises  "go.naturallyfunny.dev/api/session"
	apitime "go.naturallyfunny.dev/api/time"
	apiuser "go.naturallyfunny.dev/api/user"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL: %v", err)
	}

	ctx := context.Background()

	pgPool, err := zeedb.NewPool(
		cfg.Database.URL,
		cfg.Database.MaxConns,
		cfg.Database.MinConns,
		cfg.Database.MaxConnLifetime,
		cfg.Database.MaxConnIdleTime,
	)
	if err != nil {
		log.Fatalf("FATAL: Failed to connect to database: %v", err)
	}
	defer pgPool.Close()

	if err := zeedb.ValidateSchema(ctx, pgPool); err != nil {
		log.Fatalf("FATAL: Schema validation failed: %v", err)
	}

	tuyaClient, err := tuya.NewClient(
		cfg.Tuya.AccessID,
		cfg.Tuya.AccessSecret,
		cfg.Tuya.BaseURL,
	)
	if err != nil {
		log.Fatalf("FATAL: Failed to create Tuya client: %v", err)
	}

	accountRepo := account.NewRepository(pgPool)
	accountSvc := account.NewService(accountRepo)

	tuyaIoTClient := device.NewTuyaIoTClient(tuyaClient)
	deviceSvc := device.NewService(accountSvc.GetTuyaUID, tuyaIoTClient)

	t, err := tools.Load(tools.Services{
		Account: accountSvc,
		Device:  deviceSvc,
		Sender:  deviceSvc,
	})
	if err != nil {
		log.Fatalf("Failed to load tools: %v", err)
	}

	model, err := gemini.NewModel(ctx, "gemini-3-flash-preview", &genai.ClientConfig{
		APIKey: cfg.Gemini.APIKey,
	})
	if err != nil {
		log.Fatalf("Failed to assign Gemini model: %v", err)
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

	conversationHistory := 6
	zepClient := client.NewClient(option.WithAPIKey(cfg.Zep.APIKey))

	humanSessSvc := adksession.Wrap(
		zep.NewSessionService(zepClient, zee.Name,
			zep.WithContextHistoryLength(conversationHistory),
			zep.WithKnowledgeContext(nil),
			zep.WithUserDisplayName("Human"),
			zep.WithTimeHarnessFromContext(),
		),
		adksession.WithTimezoneFromContext(apitime.ContextKey),
	)

	avaSessSvc := adksession.Wrap(
		zep.NewSessionService(zepClient, zee.Name,
			zep.WithContextHistoryLength(conversationHistory),
			zep.WithKnowledgeContext(nil),
			zep.WithUserDisplayName("Ava"),
			zep.WithTimeHarnessFromContext(),
		),
		adksession.WithTimezoneFromContext(apitime.ContextKey),
	)

	humanRunner, err := runner.New(runner.Config{
		AppName:           cfg.App.Name,
		Agent:             zeeForUser,
		SessionService:    humanSessSvc,
		AutoCreateSession: true,
	})
	if err != nil {
		log.Fatalf("Failed to create user-channel runner: %v", err)
	}

	avaRunner, err := runner.New(runner.Config{
		AppName:           cfg.App.Name,
		Agent:             zeeForAva,
		SessionService:    avaSessSvc,
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
		r.Use(apiuser.HTTPWithID)
		r.Use(apises.HTTPWithID)
		r.Use(apitime.HTTPWithZone)

		r.Post("/chat", chat.Handle(humanRunner))
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
