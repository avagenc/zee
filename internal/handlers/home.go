package handlers

import (
	"net/http"
	"strings"

	"github.com/avagenc/agentic-tuya-smart/internal/services"
)

type HomeHandler struct {
	deviceService *services.DeviceService
}

func NewHomeHandler(deviceService *services.DeviceService) *HomeHandler {
	return &HomeHandler{deviceService: deviceService}
}

func (h *HomeHandler) HandleGetHomeDevices(w http.ResponseWriter, r *http.Request) {
	const action = "Get Home Devices"

	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed", action)
		return
	}

	homeID := strings.TrimSpace(r.URL.Query().Get("homeId"))
	if homeID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "missing homeId", action)
		return
	}

	devices, err := h.deviceService.GetAllByHomeId(homeID)
	if err != nil {
		writeErrorResponse(w, http.StatusBadGateway, err.Error(), action)
		return
	}

	writeSuccessResponse(w, http.StatusOK, devices, action)
}
