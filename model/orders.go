package model

import "time"

type Order struct {
	ID            string    `json:"id" bson:"_id,omitempty"`
	UserID        int64     `json:"userId" bson:"userId"`
	Symbol        string    `json:"symbol" bson:"symbol"`
	Quantity      int       `json:"quantity" bson:"quantity"`
	Date          time.Time `json:"date" bson:"date"`
	BuyOrder      OrderInfo `json:"buyOrder" bson:"buyOrder"`
	StopLossOrder OrderInfo `json:"stopLossOrder" bson:"stopLossOrder"`
}

type OrderInfo struct {
	OrderID      string  `json:"orderId" bson:"orderId"`
	OrderStatus  string  `json:"orderStatus" bson:"orderStatus"`
	AveragePrice float64 `json:"averagePrice" bson:"averagePrice"`
}

type OrderDto struct {
	ID       string `json:"id"`
	UserID   int64  `json:"userId"`
	Symbol   string `json:"symbol"`
	Quantity int    `json:"quantity"`
	Date     string `json:"date"`
}

type GetOrderInput struct {
	ID string `path:"id" required:"true" doc:"Order ID"`
}

type GetOrdersByDateInput struct {
	Date string `query:"date" required:"true" doc:"Date in YYYY-MM-DD or RFC3339 format"`
}

type GetOrdersByUserIdInput struct {
	UserID int64 `path:"userId" required:"true" doc:"User ID"`
}

type CreateOrderInput struct {
	Body OrderDto
}

type UpdateOrderInput struct {
	ID   string `path:"id" required:"true" doc:"Order ID"`
	Body OrderDto
}
