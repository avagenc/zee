package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/avagenc/agentic-tuya-smart/internal/models"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}

func writeSuccessResponse(w http.ResponseWriter, status int, result any, action string) {
	writeJSON(w, status, models.BaseResponse{
		Success:   true,
		Action:    action,
		Result:    result,
		Timestamp: time.Now().Unix(),
	})
}

func writeErrorResponse(w http.ResponseWriter, status int, message, action string) {
	writeJSON(w, status, models.BaseResponse{
		Success:   false,
		Action:    action,
		Error:     message,
		Timestamp: time.Now().Unix(),
	})
}
