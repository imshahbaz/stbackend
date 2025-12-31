package controller

import "backend/model"

// NewResponse creates a success response with the given data and message.
func NewResponse(data any, message string) *model.DefaultResponse {
	return &model.DefaultResponse{
		Body: model.Response{
			Success: true,
			Message: message,
			Data:    data,
		},
	}
}

// NewErrorResponse creates an error response (conceptually, though Huma handles HTTP errors separately).
func NewErrorResponse(err string) *model.DefaultResponse {
	return &model.DefaultResponse{
		Body: model.Response{
			Success: false,
			Error:   err,
		},
	}
}
