package model

type Response struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"Update successful"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type DefaultResponse struct {
	Body Response
}

