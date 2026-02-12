package main

import (
	"context"
	"log"
	"net/http"

	"github.com/avagenc/zee/internal/account"
	"github.com/avagenc/zee/internal/config"
	"github.com/avagenc/zee/internal/device"
	"github.com/avagenc/zee/internal/middleware"
	"github.com/avagenc/zee/internal/postgres"
	"github.com/avagenc/zee/internal/system"
	"github.com/avagenc/zee/internal/tuya"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg, err := config.LoadAPI()
	if err != nil {
		log.Fatalf("FATAL: %v", err)
	}

	pgPool, err := postgres.NewPool(
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

	if err := postgres.ValidateSchema(context.Background(), pgPool); err != nil {
		log.Fatalf("FATAL: Schema validation failed: %v", err)
	}

	repo := struct {
		account account.Repository
	}{
		account: account.NewRepository(pgPool),
	}

	tuyaClient, err := tuya.NewClient(
		cfg.Tuya.AccessID,
		cfg.Tuya.AccessSecret,
		cfg.Tuya.BaseURL,
	)
	if err != nil {
		log.Fatalf("FATAL: Failed to create Tuya client: %v", err)
	}

	tuyaIoTClient := struct {
		device device.TuyaIoTClient
	}{
		device: device.NewTuyaIoTClient(tuyaClient),
	}

	accountSvc := account.NewService(repo.account)

	svc := struct {
		account account.Service
		device  device.Service
	}{
		account: accountSvc,
		device:  device.NewService(accountSvc.GetTuyaUID, tuyaIoTClient.device),
	}

	hdl := struct {
		system  *system.Handler
		account *account.Handler
		device  *device.Handler
	}{
		system:  system.NewHandler(cfg.App.Name, cfg.App.Version, cfg.App.Env),
		account: account.NewHandler(svc.account),
		device:  device.NewHandler(svc.device),
	}

	r := chi.NewRouter()

	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.AuthenticateAPIKey(cfg.Security.APIKey))

	r.Get("/", hdl.system.Index)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireUserIdentity)

		r.Get("/account", hdl.account.Get)
		r.Get("/devices", hdl.device.List)

		r.Route("/devices/{deviceId}", func(r chi.Router) {
			r.Post("/commands", hdl.device.SendCommands)
		})
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
