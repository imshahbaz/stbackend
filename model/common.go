package model

// Payload is a generic version of the response body for better type safety and OpenAPI docs
type Payload[T any] struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message,omitempty" example:"Success"`
	Data    T      `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// TypedResponse is a generic Huma response wrapper
type TypedResponse[T any] struct {
	Body Payload[T]
}

// RequestBody is a generic Huma input wrapper for simple bodies
type RequestBody[T any] struct {
	Body T
}

// IDInput is a generic Huma input wrapper for requests with an ID path parameter and a body
type IDInput[T any] struct {
	ID   string `path:"id" required:"true" doc:"Identifier"`
	Body T
}

// HeaderResponse represents a response body with headers (like Set-Cookie)
type HeaderResponse[T any] struct {
	SetCookie string `header:"Set-Cookie"`
	Body      Payload[T]
}
