package main

import (
	"log"
	"net/http"

	"github.com/avagenc/agentic-tuya-smart/internal/config"
	"github.com/avagenc/agentic-tuya-smart/internal/handlers"
	"github.com/avagenc/agentic-tuya-smart/internal/middleware"
	"github.com/avagenc/agentic-tuya-smart/internal/services"
	"github.com/avagenc/agentic-tuya-smart/internal/clients/tuya"
)

func main() {
	// --- Get Configuration ---
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("FATAL: Failed to load configuration: %v", err)
	}

	// --- Dependency Injection ---

	// 1. Create Clients
	tuyaClient, err := tuya.NewClient(cfg.TuyaAccessID, cfg.TuyaAccessSecret, cfg.TuyaBaseURL)
	if err != nil {
		log.Fatalf("FATAL: Failed to create Tuya client: %v", err)
	}

	// 2. Create Services
	deviceService := services.NewDeviceService(tuyaClient)

	// 3. Create Handlers
	deviceHandler := handlers.NewDeviceHandler(deviceService)
	homeHandler := handlers.NewHomeHandler(deviceService)

	// 4. Create Middleware Authenticator
	authenticator := middleware.NewAuthenticator(cfg.AvagencAPIKey)

	// --- Register Routes ---
	mux := http.NewServeMux()

	mux.HandleFunc("/", handlers.RootHandler)
	mux.Handle("/v0.1/devices/commands", authenticator.Middleware(http.HandlerFunc(deviceHandler.HandleDeviceCommands)))
	mux.Handle("/v0.1/homes/devices", authenticator.Middleware(http.HandlerFunc(homeHandler.HandleGetHomeDevices)))

	// --- Start Server ---
	log.Printf("Starting Avagenc Tuya IoT Service on port %s\n", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatalf("FATAL: Failed to start server: %v", err)
	}
}
