package model

// Common Response structure for all API calls
type Response struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"Update successful"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// DefaultResponse is a generic wrapper for Huma responses
type DefaultResponse struct {
	Body Response
}

// Helper methods for DefaultResponse could be here or in controller
