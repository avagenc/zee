package handlers

import (
	"net/http"
)

type RootHandler struct {
	Version string
}

func NewRootHandler(version string) *RootHandler {
	return &RootHandler{Version: version}
}

func (h *RootHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const action = "Health Check"

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", action)
		return
	}

	healthCheckPayload := struct {
		Service string `json:"service"`
		Version string `json:"version"`
		Status  string `json:"status"`
	}{
		Service: "avagenc-agentic-tuya-smart",
		Version: h.Version,
		Status:  "ok",
	}

	writeSuccessResponse(w, http.StatusOK, healthCheckPayload, action)
}
