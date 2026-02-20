package controller

import "backend/model"

func NewTypedResponse[T any](data T, message string) *model.TypedResponse[T] {
	return &model.TypedResponse[T]{
		Body: model.Payload[T]{
			Success: true,
			Message: message,
			Data:    data,
		},
	}
}

func NewTypedError[T any](err string) *model.TypedResponse[T] {
	return &model.TypedResponse[T]{
		Body: model.Payload[T]{
			Success: false,
			Error:   err,
		},
	}
}
