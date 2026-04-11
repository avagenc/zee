package system

import (
	"net/http"

	"go.ibnfadl.com/api"
)

type Handler struct {
	name    string
	version string
	env     string
}

func NewHandler(name, version, env string) *Handler {
	return &Handler{
		name:    name,
		version: version,
		env:     env,
	}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Service     string `json:"service"`
		Status      string `json:"status"`
		Environment string `json:"environment"`
		Version     string `json:"version"`
	}{
		Service:     h.name,
		Status:      "UP",
		Environment: h.env,
		Version:     h.version,
	}

	api.WriteSuccess(w, http.StatusOK, "OK", "Service is healthy", data, nil)
}
