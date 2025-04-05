package models

// ErrorResponse represents the error structure returned by the API.
// swagger:model
type ErrorResponse struct {
    // Error is the error message.
    Error string `json:"error" example:"Invalid request payload"`
}