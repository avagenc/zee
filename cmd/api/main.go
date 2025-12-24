package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/avagenc/agentic-tuya-smart/internal/clients/tuya"
	"github.com/avagenc/agentic-tuya-smart/internal/config"
	"github.com/avagenc/agentic-tuya-smart/internal/handlers"
	"github.com/avagenc/agentic-tuya-smart/internal/middleware"
	"github.com/avagenc/agentic-tuya-smart/internal/services"
)

const version = "0.3.1"

var (
	versionMajor = strings.Split(version, ".")[0]
	APIPrefix    = "/v" + versionMajor
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
	deviceHandler := handlers.NewDeviceHandler(deviceService, APIPrefix)
	homeHandler := handlers.NewHomeHandler(deviceService, APIPrefix)
	rootHandler := handlers.NewRootHandler(version)

	// 4. Create Middleware Authenticator
	authenticator := middleware.NewAuthenticator(cfg.AvagencAPIKey)

	// --- Register Routes ---
	mux := http.NewServeMux()

	mux.Handle("/", rootHandler)
	mux.Handle(APIPrefix+"/devices/", authenticator.Middleware(deviceHandler))
	mux.Handle(APIPrefix+"/homes/", authenticator.Middleware(homeHandler))

	// --- Start Server ---
	log.Printf("In the name of Allah, The Most Compassionate, The Most Merciful")
	log.Printf("Starting Avagenc Agentic Tuya Smart API on port %s\n", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatalf("FATAL: Failed to start server: %v", err)
	}
}
