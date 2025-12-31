package controller

import "backend/model"

func NewResponse(data any, message string) *model.DefaultResponse {
	return &model.DefaultResponse{
		Body: model.Response{
			Success: true,
			Message: message,
			Data:    data,
		},
	}
}

func NewErrorResponse(err string) *model.DefaultResponse {
	return &model.DefaultResponse{
		Body: model.Response{
			Success: false,
			Error:   err,
		},
	}
}
