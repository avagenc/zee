package chat

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	adkagent "google.golang.org/adk/agent"
	adkrunner "google.golang.org/adk/runner"
	"google.golang.org/genai"

	"go.naturallyfunny.dev/api"
	"go.naturallyfunny.dev/api/identity"
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
			api.WriteError(w, api.NewError(api.InvalidArgument, "Invalid JSON body").WithError(err))
			return
		}
		if req.Message == "" {
			api.WriteError(w, api.NewError(api.InvalidArgument, "Field 'message' is required").
				WithDetails([]api.ErrorDetail{{Field: "message", Message: "must not be empty"}}))
			return
		}

		userID, err := identity.GetUserIDFromContext(r.Context())
		if err != nil {
			api.WriteError(w, api.NewError(api.Unauthenticated, "Missing user identity").WithError(err))
			return
		}
		sessionID, err := identity.GetSessionIDFromContext(r.Context())
		if err != nil {
			api.WriteError(w, api.NewError(api.Unauthenticated, "Missing session identity").WithError(err))
			return
		}

		msg := genai.NewContentFromText(req.Message, "user")
		events := runner.Run(r.Context(), userID, sessionID, msg, adkagent.RunConfig{})

		var b strings.Builder
		for event, err := range events {
			if err != nil {
				var apiErr *api.Error
				if errors.As(err, &apiErr) {
					api.WriteError(w, apiErr)
				} else {
					api.WriteError(w, api.NewError(api.Internal, "Failed to process message").WithError(err))
				}
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

		api.WriteSuccess(w, api.OK, "Message processed", Response{Response: b.String()}, nil)
	}
}
