package main

import (
	"context"
	_ "embed"
	"log"
	"os"
	"time"

	_ "time/tzdata"

	"github.com/joho/godotenv"
	"google.golang.org/genai"

	"go.naturallyfunny.dev/tuya"
	"go.naturallyfunny.dev/tuya/cloud"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/console"
	"google.golang.org/adk/cmd/launcher/universal"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/session"

	zee "go.avagenc.com/zee"
)

//go:embed instruction.txt
var devInstruction string

// staticAccountStore links any ownerID to a fixed Tuya UID (DEV_TUYA_UID).
// No postgres needed for local dev, and whatever user_id the session starts
// with is treated as linked.
type staticAccountStore struct {
	tuyaUID string
}

func (s *staticAccountStore) Get(_ context.Context, ownerID string) (tuya.Account, error) {
	return tuya.Account{OwnerID: ownerID, TuyaUID: s.tuyaUID, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}
func (s *staticAccountStore) Link(_ context.Context, ownerID, tuyaUID string) (tuya.Account, error) {
	return tuya.Account{OwnerID: ownerID, TuyaUID: tuyaUID}, nil
}
func (s *staticAccountStore) Unlink(_ context.Context, _ string) error { return nil }

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("FATAL: %s is required", key)
	}
	return v
}

func main() {
	ctx := context.Background()

	if err := godotenv.Load(); err != nil {
		log.Println("Info: .env not found, using system environment")
	}

	tuyaCloudClient, err := cloud.New(mustEnv("TUYA_ACCESS_ID"), mustEnv("TUYA_ACCESS_SECRET"), mustEnv("TUYA_BASE_URL"))
	if err != nil {
		log.Fatalf("FATAL: tuya cloud client: %v", err)
	}

	tuyaClient := tuya.New(cloud.NewIoT(tuyaCloudClient), &staticAccountStore{
		tuyaUID: mustEnv("DEV_TUYA_UID"),
	})

	model, err := gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{
		APIKey: mustEnv("GEMINI_API_KEY"),
	})
	if err != nil {
		log.Fatalf("FATAL: gemini model: %v", err)
	}

	zeeAgent, err := zee.NewAgent(zee.Config{
		Name:               "Zee",
		Description:        "Avagenc Tuya Smart Home Agent — dev mode (CLI)",
		ChannelInstruction: devInstruction,
		Model:              model,
		TuyaClient:         tuyaClient,
	})
	if err != nil {
		log.Fatalf("FATAL: zee agent: %v", err)
	}

	config := &launcher.Config{
		AgentLoader:    agent.NewSingleLoader(zeeAgent),
		SessionService: session.InMemoryService(),
	}

	// console is ADK's CLI run mode (equivalent to Python's `adk run`).
	// Prepend the "console" keyword so any extra flags (e.g. -streaming_mode)
	// are forwarded to it.
	l := universal.NewLauncher(console.NewLauncher())
	args := append([]string{"console"}, os.Args[1:]...)
	if err = l.Execute(ctx, config, args); err != nil {
		log.Fatalf("FATAL: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
