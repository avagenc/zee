package main

import (
	"context"
	_ "embed"
	"fmt"
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
	"github.com/avagenc/zee-agent/internal/tuya"

	"github.com/getzep/zep-go/v3/client"
	"github.com/getzep/zep-go/v3/option"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"google.golang.org/genai"

	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"go.naturallyfunny.dev/adk/zep"
	apises "go.naturallyfunny.dev/api/session"
	apitime "go.naturallyfunny.dev/api/time"
	apiuser "go.naturallyfunny.dev/api/user"
)

//go:embed base-instruction.txt
var baseInstruction string

//go:embed for-human-instruction.txt
var humanInstruction string

//go:embed for-ava-instruction.txt
var avaInstruction string

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Info: .env file not found, using system environment variables")
	}

	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("FATAL: DATABASE_URL is required")
	}
	dbCfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Fatalf("FATAL: Failed to parse database URL: %v", err)
	}
	dbCfg.MaxConns = 20
	if v := os.Getenv("DATABASE_MAX_CONNS"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			log.Fatalf("FATAL: invalid DATABASE_MAX_CONNS: %v", err)
		}
		dbCfg.MaxConns = int32(n)
	}
	dbCfg.MinConns = 0
	if v := os.Getenv("DATABASE_MIN_CONNS"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			log.Fatalf("FATAL: invalid DATABASE_MIN_CONNS: %v", err)
		}
		dbCfg.MinConns = int32(n)
	}
	dbCfg.MaxConnLifetime = time.Hour
	if v := os.Getenv("DATABASE_MAX_CONN_LIFETIME"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			log.Fatalf("FATAL: invalid DATABASE_MAX_CONN_LIFETIME: %v", err)
		}
		dbCfg.MaxConnLifetime = d
	}
	dbCfg.MaxConnIdleTime = 30 * time.Minute
	if v := os.Getenv("DATABASE_MAX_CONN_IDLE_TIME"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			log.Fatalf("FATAL: invalid DATABASE_MAX_CONN_IDLE_TIME: %v", err)
		}
		dbCfg.MaxConnIdleTime = d
	}

	dbCtx, dbCancel := context.WithTimeout(ctx, 30*time.Second)
	defer dbCancel()

	pgPool, err := pgxpool.NewWithConfig(dbCtx, dbCfg)
	if err != nil {
		log.Fatalf("FATAL: Unable to create connection pool: %v", err)
	}
	defer pgPool.Close()

	if err := pgPool.Ping(dbCtx); err != nil {
		log.Fatalf("FATAL: Unable to connect to database: %v", err)
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

	deviceSvc := device.NewService(accountSvc.GetTuyaUID, tuyaClient)

	getAccount, err := functiontool.New(
		functiontool.Config{
			Name:        "get_account",
			Description: "Use this tool to retrieve the Tuya account linked to the current authenticated avagenc user.",
		},
		func(ctx tool.Context, args struct{}) (any, error) {
			result, err := accountSvc.Get(ctx, ctx.UserID())
			if err != nil {
				fmt.Printf("[tool:get_account] error: %v\n", err)
				return nil, err
			}
			return result, nil
		},
	)
	if err != nil {
		log.Fatalf("Failed to create get_account tool: %v", err)
	}

	listDevices, err := functiontool.New(
		functiontool.Config{
			Name:        "list_devices",
			Description: "Use this tool to retrieve all Tuya IoT devices linked to the linked user's Tuya account.",
		},
		func(ctx tool.Context, args struct{}) (map[string]any, error) {
			result, err := deviceSvc.List(ctx, ctx.UserID())
			if err != nil {
				fmt.Printf("[tool:list_devices] error: %v\n", err)
				return nil, err
			}
			return map[string]any{"devices": result}, nil
		},
	)
	if err != nil {
		log.Fatalf("Failed to create list_devices tool: %v", err)
	}

	sendCommandsToADevice, err := functiontool.New(
		functiontool.Config{
			Name:        "send_commands_to_a_device",
			Description: `Use this tool to send commands to a device by the device "id". You can get "id" of devices from list_devices tool. Always pay attention to the "id" of the device, make sure you input the accurate device "id" or else it would be fatal. in string form but array of maps`,
		},
		func(ctx tool.Context, args struct {
			DeviceID string             `json:"device_id"`
			Commands []device.DataPoint `json:"commands"`
		}) (any, error) {
			result, err := deviceSvc.SendCommands(ctx, ctx.UserID(), args.DeviceID, args.Commands)
			if err != nil {
				fmt.Printf("[tool:send_commands_to_a_device] error: %v\n", err)
				return nil, err
			}
			return result, nil
		},
	)
	if err != nil {
		log.Fatalf("Failed to create send_commands_to_a_device tool: %v", err)
	}

	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	if geminiAPIKey == "" {
		log.Fatal("FATAL: GEMINI_API_KEY is required")
	}
	model, err := gemini.NewModel(ctx, "gemini-3.1-flash", &genai.ClientConfig{
		APIKey: geminiAPIKey,
	})
	if err != nil {
		log.Fatalf("Failed to assign Gemini model: %v", err)
	}

	tuyaTools := []tool.Tool{getAccount, listDevices, sendCommandsToADevice}

	const name = "Zee"

	agentForHuman, err := llmagent.New(llmagent.Config{
		Name:        name,
		Model:       model,
		Tools:       tuyaTools,
		Description: "Avagenc Tuya Smart Agent, Human triggered processing",
		Instruction: "[SYSTEM_INSTRUCTION]" + baseInstruction + "\n" + humanInstruction + "\n[/SYSTEM_INSTRUCTION]",
	})
	if err != nil {
		log.Fatalf("Failed to create user-channel agent: %v", err)
	}

	agentForAva, err := llmagent.New(llmagent.Config{
		Name:        name,
		Model:       model,
		Tools:       tuyaTools,
		Description: "Avagenc Tuya Smart Agent, Ava triggered processing",
		Instruction: "[SYSTEM_INSTRUCTION]" + baseInstruction + "\n" + avaInstruction + "\n[/SYSTEM_INSTRUCTION]",
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

	humanSessSvc := zep.NewSessionService(zepClient, name,
		zep.WithMessagesHistoryLength(conversationHistory),
		zep.WithKnowledgeContext(nil),
		zep.WithUserDisplayName("Human (Avagenc User)"),
		zep.WithTimeHarnessFromContext(apitime.ContextKey),
	)

	avaSessSvc := zep.NewSessionService(zepClient, name,
		zep.WithMessagesHistoryLength(conversationHistory),
		zep.WithKnowledgeContext(nil),
		zep.WithUserDisplayName("Ava (Avagenc Agent)"),
		zep.WithTimeHarnessFromContext(apitime.ContextKey),
	)

	humanRunner, err := runner.New(runner.Config{
		AppName:           "zee-agent",
		Agent:             agentForHuman,
		SessionService:    humanSessSvc,
		AutoCreateSession: true,
	})
	if err != nil {
		log.Fatalf("Failed to create user-channel runner: %v", err)
	}

	avaRunner, err := runner.New(runner.Config{
		AppName:           "zee-agent",
		Agent:             agentForAva,
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

		r.Post("/chat", chat.Handle(humanRunner, ""))
		r.Post("/chat/agent", chat.Handle(avaRunner, "@zee "))
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
