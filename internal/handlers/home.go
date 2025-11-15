package handlers

import (
	"net/http"
	"strings"

	"github.com/avagenc/agentic-tuya-smart/internal/services"
)

type HomeHandler struct {
	deviceService *services.DeviceService
	APIPrefix     string
}

func NewHomeHandler(deviceService *services.DeviceService, apiPrefix string) *HomeHandler {
	return &HomeHandler{deviceService: deviceService, APIPrefix: apiPrefix}
}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	trimmedPath := strings.TrimPrefix(r.URL.Path, h.APIPrefix+"/homes/")
	pathSegments := strings.Split(trimmedPath, "/")

	if len(pathSegments) == 2 && pathSegments[1] == "devices" {
		homeID := pathSegments[0]
		h.handleGetHomeDevices(w, r, homeID)
	} else {
		http.NotFound(w, r)
	}
}

func (h *HomeHandler) handleGetHomeDevices(w http.ResponseWriter, r *http.Request, homeID string) {
	const action = "Get Home Devices"

	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed", action)
		return
	}

	if homeID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "missing homeId in path", action)
		return
	}

	devices, err := h.deviceService.GetAllByHomeId(homeID)
	if err != nil {
		writeErrorResponse(w, http.StatusBadGateway, err.Error(), action)
		return
	}

	writeSuccessResponse(w, http.StatusOK, devices, action)
}
