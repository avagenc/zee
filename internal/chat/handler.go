package chat

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/avagenc/zee-agent/pkg/api"
	"github.com/getzep/zep-go/v3"
	zepclient "github.com/getzep/zep-go/v3/client"
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
	zepClient      *zepclient.Client
	sessionService session.Service
}

func NewHandler(rnr *runner.Runner, zepClient *zepclient.Client, sessionService session.Service) *Handler {
	return &Handler{
		rnr:            rnr,
		zepClient:      zepClient,
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

	userID, err := api.GetUserIDFromContext(r.Context())
	if err != nil || userID == "" {
		api.WriteError(w, api.NewError(http.StatusUnauthorized, "UNAUTHORIZED", "Missing user identity"))
		return
	}

	_, err = h.zepClient.User.Add(r.Context(), &zep.CreateUserRequest{
		UserID: userID,
	})
	if err != nil {
		log.Printf("zep user upsert (non-fatal): %v", err)
	}

	var threadID string
	threads, err := h.zepClient.User.GetThreads(r.Context(), userID)
	if err == nil && len(threads) > 0 {
		threadID = *threads[0].ThreadID
	} else {
		threadID = uuid.New().String()
		_, err = h.zepClient.Thread.Create(r.Context(), &zep.CreateThreadRequest{
			ThreadID: threadID,
			UserID:   userID,
		})
		if err != nil {
			log.Printf("failed to create zep thread: %v", err)
			api.WriteError(w, api.NewError(http.StatusInternalServerError, "AGENT_ERROR", "Failed to create conversation thread: "+err.Error()))
			return
		}
	}

	_, err = h.sessionService.Create(r.Context(), &session.CreateRequest{
		AppName:   "zee-agent",
		UserID:    userID,
		SessionID: threadID,
	})
	if err != nil {
		api.WriteError(w, api.NewError(http.StatusInternalServerError, "AGENT_ERROR", "Failed to initialize session: "+err.Error()))
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
