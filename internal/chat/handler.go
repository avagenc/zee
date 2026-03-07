package chat

import (
	"encoding/json"
	"net/http"

	"github.com/avagenc/zee-agent/pkg/api"
	"github.com/google/uuid"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Response string `json:"response"`
}

type Handler struct {
	rnr            *runner.Runner
	sessionService session.Service
}

func NewHandler(rnr *runner.Runner, sessionService session.Service) *Handler {
	return &Handler{
		rnr:            rnr,
		sessionService: sessionService,
	}
}

func (h *Handler) HandleChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, api.NewError(http.StatusBadRequest, "INVALID_REQUEST", "Failed to parse JSON body"))
		return
	}

	if req.Message == "" {
		api.WriteError(w, api.NewError(http.StatusBadRequest, "INVALID_REQUEST", "Field 'message' cannot be empty"))
		return
	}

	msg := genai.NewContentFromText(req.Message, "user")
	sessionID := uuid.New().String()
	userID := "system"

	_, err := h.sessionService.Create(r.Context(), &session.CreateRequest{
		AppName:   "zee-agent",
		UserID:    userID,
		SessionID: sessionID,
	})
	if err != nil {
		api.WriteError(w, api.NewError(http.StatusInternalServerError, "AGENT_ERROR", "Failed to initialize stateless session: "+err.Error()))
		return
	}
	defer func() {
		_ = h.sessionService.Delete(r.Context(), &session.DeleteRequest{
			AppName:   "zee-agent",
			UserID:    userID,
			SessionID: sessionID,
		})
	}()

	events := h.rnr.Run(r.Context(), userID, sessionID, msg, agent.RunConfig{})

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
