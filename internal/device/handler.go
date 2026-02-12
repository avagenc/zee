package device

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/avagenc/zee/internal/domain"
	"github.com/avagenc/zee/pkg/api"
	"github.com/go-chi/chi/v5"
)

type Service interface {
	List(ctx context.Context, userID string) ([]domain.Device, error)
	SendCommands(ctx context.Context, userID string, deviceID string, commands []domain.DataPoint) (json.RawMessage, error)
}

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := api.GetUserIDFromContext(r.Context())
	if err != nil {
		api.Respond(w, http.StatusUnauthorized, api.NewErrorResponse("UNAUTHORIZED", "Missing user identity", nil))
		return
	}

	devices, err := h.svc.List(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrAccountNotLinked) {
			api.Respond(w, http.StatusUnauthorized, api.NewErrorResponse("UNAUTHORIZED", "No Tuya App Account is linked to the user", nil))
			return
		}
		api.Respond(w, http.StatusBadGateway, api.NewErrorResponse("UPSTREAM_ERROR", err.Error(), nil))
		return
	}

	api.Respond(w, http.StatusOK, api.NewSuccessResponse("Devices retrieved successfully", devices, nil))
}

func (h *Handler) SendCommands(w http.ResponseWriter, r *http.Request) {
	userID, err := api.GetUserIDFromContext(r.Context())
	if err != nil {
		api.Respond(w, http.StatusUnauthorized, api.NewErrorResponse("UNAUTHORIZED", "Missing user identity", nil))
		return
	}

	deviceID := chi.URLParam(r, "deviceId")
	if deviceID == "" {
		api.Respond(w, http.StatusBadRequest, api.NewErrorResponse("INVALID_REQUEST", "Missing deviceId", nil))
		return
	}

	var req struct {
		Commands []domain.DataPoint `json:"commands"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Respond(w, http.StatusBadRequest, api.NewErrorResponse("INVALID_REQUEST", "Invalid request body", nil))
		return
	}

	if len(req.Commands) == 0 {
		api.Respond(w, http.StatusBadRequest, api.NewErrorResponse("INVALID_REQUEST", "Commands cannot be empty", nil))
		return
	}

	result, err := h.svc.SendCommands(r.Context(), userID, deviceID, req.Commands)
	if err != nil {
		if errors.Is(err, domain.ErrAccountNotLinked) {
			api.Respond(w, http.StatusUnauthorized, api.NewErrorResponse("UNAUTHORIZED", "No Tuya App Account is linked to the user", nil))
			return
		}
		if errors.Is(err, domain.ErrDeviceNotOwned) {
			api.Respond(w, http.StatusForbidden, api.NewErrorResponse("FORBIDDEN", "Device does not belong to user", nil))
			return
		}
		api.Respond(w, http.StatusBadGateway, api.NewErrorResponse("UPSTREAM_ERROR", err.Error(), nil))
		return
	}

	api.Respond(w, http.StatusOK, api.NewSuccessResponse("Commands sent successfully", result, nil))
}
