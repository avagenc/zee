package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/avagenc/zee-api/internal/tuya"
	"github.com/avagenc/zee-api/internal/config"
	"github.com/avagenc/zee-api/internal/handlers"
	"github.com/avagenc/zee-api/internal/middleware"
	"github.com/avagenc/zee-api/internal/postgres"
	"github.com/avagenc/zee-api/internal/services"
	"github.com/avagenc/zee-api/internal/system"
)

func main() {
	log.Println("In the name of Allah, The Most Compassionate, The Most Merciful")

	appCfg, err := config.LoadApp()
	if err != nil {
		log.Fatalf("FATAL: Failed to load app config: %v", err)
	}

	serverCfg, err := config.LoadServer()
	if err != nil {
		log.Fatalf("FATAL: Failed to load server config: %v", err)
	}

	securityCfg, err := config.LoadSecurity()
	if err != nil {
		log.Fatalf("FATAL: Failed to load security config: %v", err)
	}

	tuyaCfg, err := config.LoadTuya()
	if err != nil {
		log.Fatalf("FATAL: Failed to load tuya config: %v", err)
	}

	pgCfg, err := config.LoadPGPool()
	if err != nil {
		log.Fatalf("FATAL: Failed to load postgres config: %v", err)
	}

	pool, err := postgres.NewPool(pgCfg)
	if err != nil {
		log.Fatalf("FATAL: Failed to create database pool: %v", err)
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := postgres.ValidateSchema(ctx, pool); err != nil {
		log.Fatalf("FATAL: Schema validation failed: %v", err)
	}

	tuyaClient, err := tuya.NewClient(tuyaCfg)
	if err != nil {
		log.Fatalf("FATAL: Failed to create Tuya client: %v", err)
	}

	deviceService := services.NewDeviceService(tuyaClient)

	systemHandler := system.NewHandler(appCfg)
	deviceHandler := handlers.NewDeviceHandler(deviceService, "/v0")
	homeHandler := handlers.NewHomeHandler(deviceService, "/v0")

	apiKeyMiddleware := middleware.NewAPIKey(securityCfg)

	mux := http.NewServeMux()

	mux.HandleFunc("/", systemHandler.Index)
	mux.Handle("/v0/devices/", apiKeyMiddleware.Authenticate(deviceHandler))
	mux.Handle("/v0/homes/", apiKeyMiddleware.Authenticate(homeHandler))

	server := &http.Server{
		Addr:         ":" + serverCfg.Port,
		Handler:      mux,
		ReadTimeout:  serverCfg.ReadTimeout,
		WriteTimeout: serverCfg.WriteTimeout,
		IdleTimeout:  serverCfg.IdleTimeout,
	}

	go func() {
		log.Printf("Starting %s v%s on port %s (env: %s)", appCfg.Name, appCfg.Version, serverCfg.Port, appCfg.Env)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("FATAL: Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server gracefully...")

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("FATAL: Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
