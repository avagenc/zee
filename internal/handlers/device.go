package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/avagenc/agentic-tuya-smart/internal/models"
	"github.com/avagenc/agentic-tuya-smart/internal/services"
)

type DeviceHandler struct {
	deviceService *services.DeviceService
}

func NewDeviceHandler(deviceService *services.DeviceService) *DeviceHandler {
	return &DeviceHandler{deviceService: deviceService}
}

func (h *DeviceHandler) HandleDeviceCommands(w http.ResponseWriter, r *http.Request) {
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
