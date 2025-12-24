package model

// UpdateThemeRequest is the payload from React
type UpdateThemeRequest struct {
	Theme UserTheme `json:"theme" example:"DARK" enums:"LIGHT,DARK" binding:"required"`
}
