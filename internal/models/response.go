package models

type BaseResponse struct {
	Action  string `json:"action"`
	Success bool   `json:"success"`
	Timestamp int64  `json:"timestamp"`

	// Exists when Success is true
	Result  any    `json:"result,omitempty"`

	// Exists when Success is false
	Error   string `json:"error,omitempty"`
}
