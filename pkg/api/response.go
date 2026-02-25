package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

type ErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

type Meta struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"perPage,omitempty"`
	TotalItems int `json:"totalItems,omitempty"`
	TotalPages int `json:"totalPages,omitempty"`
}

type Response[T any] struct {
	Success bool          `json:"success"`
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Data    T             `json:"data,omitempty"`
	Meta    *Meta         `json:"meta,omitempty"`
	Errors  []ErrorDetail `json:"errors,omitempty"`
}

type Error struct {
	Status  int
	Code    string
	Message string
	Details []ErrorDetail
	Err     error
}

func NewError(status int, code, message string) *Error {
	return &Error{
		Status:  status,
		Code:    code,
		Message: message,
	}
}

func (e *Error) WithDetails(details []ErrorDetail) *Error {
	e.Details = details
	return e
}

func (e *Error) WithError(err error) *Error {
	e.Err = err
	return e
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("api: %s (%d): %s - %v", e.Code, e.Status, e.Message, e.Err)
	}
	return fmt.Sprintf("api: %s (%d): %s", e.Code, e.Status, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Err
}

func writeJSON[T any](w http.ResponseWriter, status int, payload Response[T]) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("api: failed to encode response: %v", err)
	}
}

func WriteSuccess[T any](w http.ResponseWriter, status int, code, message string, data T, meta *Meta) {
	writeJSON(w, status, Response[T]{
		Success: true,
		Code:    code,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

func writeFailure(w http.ResponseWriter, status int, code, message string, details []ErrorDetail) {
	writeJSON(w, status, Response[struct{}]{
		Success: false,
		Code:    code,
		Message: message,
		Errors:  details,
	})
}

func WriteError(w http.ResponseWriter, err error) {
	var apiErr *Error

	if errors.As(err, &apiErr) {
		if apiErr.Status >= http.StatusInternalServerError && apiErr.Err != nil {
			log.Printf("api: wrapped internal error: %v", apiErr.Err)
		}
		writeFailure(w, apiErr.Status, apiErr.Code, apiErr.Message, apiErr.Details)
		return
	}

	log.Printf("api: unhandled internal error: %v", err)
	writeFailure(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", nil)
}
