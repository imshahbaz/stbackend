package model

import (
	"time"
)

type Order struct {
	ID            string    `json:"id" bson:"_id,omitempty"`
	UserID        int64     `json:"userId" bson:"userId"`
	Symbol        string    `json:"symbol" bson:"symbol"`
	Quantity      int       `json:"quantity" bson:"quantity"`
	Date          time.Time `json:"date" bson:"date"`
	BuyOrder      OrderInfo `json:"buyOrder" bson:"buyOrder"`
	StopLossOrder OrderInfo `json:"stopLossOrder" bson:"stopLossOrder"`
	Margin        Margin    `json:"margin" bson:"margin"`
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

type StrategyOrder struct {
	ID           string    `json:"id" bson:"_id,omitempty"`
	UserID       int64     `json:"userId" bson:"userId"`
	Date         time.Time `json:"date" bson:"date"`
	StrategyName string    `json:"strategyName" bson:"strategyName"`
	Amount       float64   `json:"amount" bson:"amount"`
}

type StrategyOrderDto struct {
	ID           string  `json:"id"`
	UserID       int64   `json:"userId"`                   // This will be set by the controller from ctx
	Date         string  `json:"date" validate:"required"` // Format: YYYY-MM-DD
	StrategyName string  `json:"strategyName" validate:"required"`
	Amount       float64 `json:"amount" validate:"required"`
}

type CreateStrategyOrderInput struct {
	Body StrategyOrderDto
}

type GetStrategyOrderInput struct {
	ID string `path:"id" required:"true" doc:"Strategy Order ID"`
}

type GetAllStrategyOrdersInput struct {
	StrategyName string `query:"strategyName" doc:"Filter by strategy name"`
}

func (s *StrategyOrder) ToDto() StrategyOrderDto {
	return StrategyOrderDto{
		ID:           s.ID,
		UserID:       s.UserID,
		Date:         s.Date.Format("2006-01-02"),
		StrategyName: s.StrategyName,
		Amount:       s.Amount,
	}
}

func (d *StrategyOrderDto) ToEntity() (StrategyOrder, error) {
	t, err := time.Parse("2006-01-02", d.Date)
	if err != nil {
		return StrategyOrder{}, err
	}
	return StrategyOrder{
		ID:           d.ID,
		UserID:       d.UserID,
		Date:         t,
		StrategyName: d.StrategyName,
		Amount:       d.Amount,
	}, nil
}
