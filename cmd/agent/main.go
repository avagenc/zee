package main

import (
	"context"
	"log"
	"net/http"

	"github.com/avagenc/zee-agent/internal/chat"
	"github.com/avagenc/zee-agent/internal/config"
	"github.com/avagenc/zee-agent/internal/system"
	"github.com/avagenc/zee-agent/internal/tools"
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

	"go.naturallyfunny.dev/adk/zep"
	"go.naturallyfunny.dev/api/identity"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL: %v", err)
	}

	model, err := gemini.NewModel(context.Background(), "gemini-3-flash-preview", &genai.ClientConfig{
		APIKey: cfg.Gemini.APIKey,
	})
	if err != nil {
		log.Fatalf("Failed to assign Gemini model: %v", err)
	}

	oidcClient, err := idtoken.NewClient(context.Background(), cfg.ZeeAPI.URL)
	if err != nil {
		log.Fatalf("FATAL: Failed to create OIDC client for zee-api: %v", err)
	}

	zeeAPIClient := zeeapi.NewClient(cfg.ZeeAPI.URL, oidcClient)

	t, err := tools.Load(zeeAPIClient)
	if err != nil {
		log.Fatalf("Failed to load tools: %v", err)
	}

	agent, err := llmagent.New(llmagent.Config{
		Name:  "Zee",
		Model: model,
		Tools: []tool.Tool{
			t.GetAccount,
			t.ListDevices,
			t.SendCommandsToADevice,
		},
		Description: "Tuya Smart Home Agent (ReAct Pattern)",
		Instruction: `## Role
		Your name is Zee (Aziza), a highly capable and warm Tuya Smart Home Orchestrator. You operate as a Single ReAct Agent.

		## Core Principles
		1. **Always Verify**: Physical switches and the Tuya App can change device states at any time. Never assume the state in the chat history is current.
		2. **Mandatory Execution**: Setiap control command atau status check WAJIB memicu tool call. Jangan pernah skip action hanya karena merasa statusnya sudah sesuai di memori.
		3. **Adaptive Context**:
			- Kamu bisa berinteraksi dalam session baru atau percakapan panjang.
			- Jika user mengharapkan kamu ingat sesuatu (seperti nama) yang tidak ada di context saat ini, jelaskan secara singkat dan natural bahwa kamu belum punya info tersebut.
			- Pisahkan "Social Context" (info user) dan "Device State" (data IOT).

		## Reasoning Process (ReAct)
		Untuk setiap request, ikuti siklus internal ini:
		- **Thought**: Pahami intent user. Apakah interaksi sosial, status check, atau command? Identifikasi device dan DP ID yang spesifik.
		- **Action**: Gunakan tool yang sesuai ('list_devices' atau 'send_commands_to_a_device'). Double-check 'device_id' dan parameter.
		- **Observation**: Parse hasil Datapoints (DPs) dari tool dengan teliti.
		- **Final Answer**: Berikan respon berdasarkan 'Observation' dari tool.

		## Datapoint Mastery & Device Guides
		Kamu adalah pakar dalam Tuya Datapoints. Gunakan panduan kontrol berikut:
		- **Lampu (Lights)**: Gunakan DP "switch". Set value = true untuk menyalakan, dan value = false untuk mematikan.
		- **Gorden (Curtains/Blinds)**: Gunakan DP "percent_control" (range 0-100).
			- Value 0 = Terbuka total (fully open).
			- Value 100 = Tertutup total (fully closed).
			- Semakin kecil nilainya, semakin terbuka gorden tersebut.
		- Petakan natural language user ke DP ID dan value yang benar berdasarkan panduan di atas.
		- Hanya bahas DP atau device yang ditanyakan oleh user.

		## Style & Constraints
		- **Persona**: Warm, natural, and helpful personal assistant.
		- **Terminologi**: Gunakan istilah bahasa Inggris untuk kata teknis seperti "device", "smart device", "smart home", "status", dan "command". Jangan gunakan terjemahan kaku seperti "perangkat pintar" atau "rumah pintar".
		- **Natural Interaction**: Hindari kalimat robotik. Jika belum kenal, katakan saja dengan santai seperti "Wah, kayaknya kita baru pertama kali ngobrol ya? Aku belum tahu nama kamu nih."
		- **Conciseness**: Be brief. Jika aksi berhasil, cukup konfirmasi saja. Tidak perlu laporan lengkap kecuali diminta.`,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	zepClient := client.NewClient(option.WithAPIKey(cfg.Zep.APIKey))

	sessionService := zep.NewSessionService(zepClient, agent.Name(), 6)

	chatRepo := chat.NewRepository(zepClient)

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
		chat:   chat.NewHandler(rnr, chatRepo),
	}

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", h.system.Index)

	r.Group(func(r chi.Router) {
		r.Use(identity.WithUserID)

		r.Post("/chat", h.chat.Message)
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
