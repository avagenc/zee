package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "time/tzdata"

	"github.com/avagenc/zee-agent/internal/account"
	"github.com/avagenc/zee-agent/internal/chat"
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
	"github.com/joho/godotenv"
	"google.golang.org/genai"

	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/tool"

	adksession "go.naturallyfunny.dev/adk/session"
	"go.naturallyfunny.dev/adk/zep"
	apises "go.naturallyfunny.dev/api/session"
	apitime "go.naturallyfunny.dev/api/time"
	apiuser "go.naturallyfunny.dev/api/user"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Info: .env file not found, using system environment variables")
	}

	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("FATAL: DATABASE_URL is required")
	}
	maxConns := int32(20)
	if v := os.Getenv("DATABASE_MAX_CONNS"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			log.Fatalf("FATAL: invalid DATABASE_MAX_CONNS: %v", err)
		}
		maxConns = int32(n)
	}
	minConns := int32(0)
	if v := os.Getenv("DATABASE_MIN_CONNS"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			log.Fatalf("FATAL: invalid DATABASE_MIN_CONNS: %v", err)
		}
		minConns = int32(n)
	}
	maxConnLifetime := time.Hour
	if v := os.Getenv("DATABASE_MAX_CONN_LIFETIME"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			log.Fatalf("FATAL: invalid DATABASE_MAX_CONN_LIFETIME: %v", err)
		}
		maxConnLifetime = d
	}
	maxConnIdleTime := 30 * time.Minute
	if v := os.Getenv("DATABASE_MAX_CONN_IDLE_TIME"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			log.Fatalf("FATAL: invalid DATABASE_MAX_CONN_IDLE_TIME: %v", err)
		}
		maxConnIdleTime = d
	}
	pgPool, err := zeedb.NewPool(databaseURL, maxConns, minConns, maxConnLifetime, maxConnIdleTime)
	if err != nil {
		log.Fatalf("FATAL: Failed to connect to database: %v", err)
	}
	defer pgPool.Close()

	if err := zeedb.ValidateSchema(ctx, pgPool); err != nil {
		log.Fatalf("FATAL: Schema validation failed: %v", err)
	}

	tuyaAccessID := os.Getenv("TUYA_ACCESS_ID")
	if tuyaAccessID == "" {
		log.Fatal("FATAL: TUYA_ACCESS_ID is required")
	}
	tuyaAccessSecret := os.Getenv("TUYA_ACCESS_SECRET")
	if tuyaAccessSecret == "" {
		log.Fatal("FATAL: TUYA_ACCESS_SECRET is required")
	}
	tuyaBaseURL := os.Getenv("TUYA_BASE_URL")
	if tuyaBaseURL == "" {
		log.Fatal("FATAL: TUYA_BASE_URL is required")
	}
	tuyaClient, err := tuya.NewClient(tuyaAccessID, tuyaAccessSecret, tuyaBaseURL)
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

	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	if geminiAPIKey == "" {
		log.Fatal("FATAL: GEMINI_API_KEY is required")
	}
	model, err := gemini.NewModel(ctx, "gemini-3-flash-preview", &genai.ClientConfig{
		APIKey: geminiAPIKey,
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
		Description: "Avagenc Tuya Smart Agent, Human triggered processing",
		Instruction: zee.SystemInstruction(),
	})
	if err != nil {
		log.Fatalf("Failed to create user-channel agent: %v", err)
	}

	zeeForAva, err := llmagent.New(llmagent.Config{
		Name:        zee.Name,
		Model:       model,
		Tools:       tuyaTools,
		Description: "Avagenc Tuya Smart Agent, Ava triggered processing",
		Instruction: zee.SystemInstruction(zee.ForAva()),
	})
	if err != nil {
		log.Fatalf("Failed to create ava-channel agent: %v", err)
	}

	zepAPIKey := os.Getenv("ZEP_API_KEY")
	if zepAPIKey == "" {
		log.Fatal("FATAL: ZEP_API_KEY is required")
	}
	conversationHistory := 8
	zepClient := client.NewClient(option.WithAPIKey(zepAPIKey))

	humanSessSvc := adksession.Wrap(
		zep.NewSessionService(zepClient, zee.Name,
			zep.WithMessagesHistoryLength(conversationHistory),
			zep.WithKnowledgeContext(nil),
			zep.WithUserDisplayName("Human (Avagenc User)"),
			zep.WithTimeHarnessFromContext(),
		),
		adksession.WithTimezoneFromContext(apitime.ContextKey),
	)

	avaSessSvc := adksession.Wrap(
		zep.NewSessionService(zepClient, zee.Name,
			zep.WithMessagesHistoryLength(conversationHistory),
			zep.WithKnowledgeContext(nil),
			zep.WithUserDisplayName("Ava (Avagenc Agent)"),
			zep.WithTimeHarnessFromContext(),
		),
		adksession.WithTimezoneFromContext(apitime.ContextKey),
	)

	humanRunner, err := runner.New(runner.Config{
		AppName:           "zee-agent",
		Agent:             zeeForUser,
		SessionService:    humanSessSvc,
		AutoCreateSession: true,
	})
	if err != nil {
		log.Fatalf("Failed to create user-channel runner: %v", err)
	}

	avaRunner, err := runner.New(runner.Config{
		AppName:           "zee-agent",
		Agent:             zeeForAva,
		SessionService:    avaSessSvc,
		AutoCreateSession: true,
	})
	if err != nil {
		log.Fatalf("Failed to create ava-channel runner: %v", err)
	}

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		log.Fatal("FATAL: APP_ENV is required")
	}
	sys := system.NewHandler("zee-agent", "v0.2.0", appEnv)

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
		r.Post("/chat/agent", chat.Handle(avaRunner))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	const readTimeout = 16 * time.Second
	const writeTimeout = 120 * time.Second
	const idleTimeout = 120 * time.Second
	
	s := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	log.Printf("In the name of Allah, The Most Compassionate, The Most Merciful")
	log.Printf("Starting zee-agent (v0.2.0) on port %s\n", port)

	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("FATAL: Failed to start API: %v", err)
	}
}
