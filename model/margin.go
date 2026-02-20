package model

import "github.com/danielgtaylor/huma/v2"

type UploadMarginInput struct {
	RawBody huma.MultipartFormFiles[struct {
		File huma.FormFile `form:"file" contentType:"text/csv" required:"true"`
	}]
}

type Margin struct {
	Symbol string  `bson:"_id" json:"symbol"`
	Name   string  `bson:"name" json:"name"`
	Margin float32 `bson:"margin" json:"margin"`
	Token  string  `bson:"token" json:"token"`
}

type StockMarginDto struct {
	Name            string  `json:"name"`
	Symbol          string  `json:"symbol"`
	Margin          float32 `json:"margin"`
	Close           float32 `json:"close"`
	DeliveryPercent float32 `json:"deliveryPercent,omitempty"`
}

type GetMarginInput struct {
	Symbol string `path:"symbol" doc:"Stock Symbol" example:"RELIANCE"`
}

type MTFInput struct {
	RawBody huma.MultipartFormFiles[struct {
		File huma.FormFile `form:"file" required:"true"`
	}]
}
