package chat

import (
	"encoding/json"
	"net/http"
	"strings"

	adkagent "google.golang.org/adk/agent"
	adkrunner "google.golang.org/adk/runner"
	"google.golang.org/genai"

	apihttp "go.naturallyfunny.dev/api/http"
	apises  "go.naturallyfunny.dev/api/session"
	apiuser "go.naturallyfunny.dev/api/user"
)

type Request struct {
	Message string `json:"message"`
}

type Response struct {
	Response string `json:"response"`
}

func Handle(runner *adkrunner.Runner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apihttp.WriteProblem(w, http.StatusBadRequest, map[string]any{"detail": "Invalid JSON body"})
			return
		}
		if req.Message == "" {
			apihttp.WriteProblem(w, http.StatusBadRequest, map[string]any{
				"detail": "Field 'message' is required",
				"errors": []map[string]string{{"field": "message", "message": "must not be empty"}},
			})
			return
		}

		userID, err := apiuser.IDFromContext(r.Context())
		if err != nil {
			apihttp.WriteProblem(w, http.StatusUnauthorized, map[string]any{"detail": err.Error()})
			return
		}
		sessionID, err := apises.IDFromContext(r.Context())
		if err != nil {
			apihttp.WriteProblem(w, http.StatusUnauthorized, map[string]any{"detail": err.Error()})
			return
		}

		msg := genai.NewContentFromText(req.Message, "user")
		events := runner.Run(r.Context(), userID, sessionID, msg, adkagent.RunConfig{})

		var b strings.Builder
		for event, err := range events {
			if err != nil {
				apihttp.WriteProblem(w, http.StatusInternalServerError, map[string]any{"detail": "Failed to process message"})
				return
			}
			if event.Content == nil {
				continue
			}
			for _, part := range event.Content.Parts {
				if part.Text != "" {
					b.WriteString(part.Text)
				}
			}
		}

		apihttp.WriteJSON(w, http.StatusOK, Response{Response: b.String()})
	}
}
