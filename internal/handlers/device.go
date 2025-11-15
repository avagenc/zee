package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/avagenc/agentic-tuya-smart/internal/models"
	"github.com/avagenc/agentic-tuya-smart/internal/services"
)

type DeviceHandler struct {
	deviceService *services.DeviceService
	APIPrefix     string
}

func NewDeviceHandler(deviceService *services.DeviceService, apiPrefix string) *DeviceHandler {
	return &DeviceHandler{
		deviceService: deviceService,
		APIPrefix:     apiPrefix,
	}
}

func (h *DeviceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	subPath := strings.TrimPrefix(r.URL.Path, h.APIPrefix+"/devices/")

	switch subPath {
	case "commands":
		h.handleCommands(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *DeviceHandler) handleCommands(w http.ResponseWriter, r *http.Request) {
	const action = "Send Device Commands"

	if r.Method != http.MethodPost {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed", action)
		return
	}

	var payload models.DeviceCommandsRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid request body", action)
		return
	}

	if payload.DeviceID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "missing device_id", action)
		return
	}

	if len(payload.Commands) == 0 {
		writeErrorResponse(w, http.StatusBadRequest, "commands cannot be empty", action)
		return
	}

	result, err := h.deviceService.SendCommands(payload.DeviceID, payload.Commands)
	if err != nil {
		writeErrorResponse(w, http.StatusBadGateway, fmt.Sprintf("SendDeviceCommands error: %v", err), action)
		return
	}

	writeSuccessResponse(w, http.StatusOK, result, action)
}
