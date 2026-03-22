package chat

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/avagenc/zee-agent/pkg/api"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/genai"
)

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Response string `json:"response"`
}

type Repository interface {
	UpsertUser(ctx context.Context, userID string) error
	GetOrCreateThreadID(ctx context.Context, userID string) (string, error)
}

type Handler struct {
	rnr  *runner.Runner
	repo Repository
}

func NewHandler(rnr *runner.Runner, repo Repository) *Handler {
	return &Handler{
		rnr:  rnr,
		repo: repo,
	}
}

func (h *Handler) Message(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, api.NewError(http.StatusBadRequest, "INVALID_REQUEST", "Failed to parse JSON body"))
		return
	}

	if req.Message == "" {
		api.WriteError(w, api.NewError(http.StatusBadRequest, "INVALID_REQUEST", "Field 'message' cannot be empty"))
		return
	}

	userID, err := api.GetUserIDFromContext(r.Context())
	if err != nil || userID == "" {
		api.WriteError(w, api.NewError(http.StatusUnauthorized, "UNAUTHORIZED", "Missing user identity"))
		return
	}

	go func() {
		if err := h.repo.UpsertUser(context.Background(), userID); err != nil {
			log.Printf("zep user upsert (non-fatal): %v", err)
		}
	}()

	threadID, err := h.repo.GetOrCreateThreadID(r.Context(), userID)
	if err != nil {
		api.WriteError(w, api.NewError(http.StatusInternalServerError, "AGENT_ERROR", "Failed to setup conversation thread: "+err.Error()))
		return
	}

	msg := genai.NewContentFromText(req.Message, "user")
	events := h.rnr.Run(r.Context(), userID, threadID, msg, agent.RunConfig{})

	var fullResponse string
	for event, err := range events {
		if err != nil {
			api.WriteError(w, api.NewError(http.StatusInternalServerError, "AGENT_ERROR", err.Error()))
			return
		}
		if event.Content != nil {
			for _, part := range event.Content.Parts {
				if part.Text != "" {
					fullResponse += part.Text
				}
			}
		}
	}

	api.WriteSuccess(w, http.StatusOK, "SUCCESS", "Message processed", ChatResponse{
		Response: fullResponse,
	}, nil)
}
