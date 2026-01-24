package service

import (
	"backend/model"
	"context"
	"fmt"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

type ZerodhaService interface {
	InitiateKiteConnect(ctx context.Context, accessToken string) (*kiteconnect.Client, error)
	GenerateAccessToken(requestToken string, userId int64) (string, error)
	PlaceMTFOrder(kc *kiteconnect.Client, symbol string, qty int, price float64) (kiteconnect.OrderResponse, error)
	PlaceMTFStopLossOrder(kc *kiteconnect.Client, symbol string, qty int, price float64, triggerPrice float64) (string, error)
	GetOrderDetails(kc *kiteconnect.Client, orderID string) (kiteconnect.Order, error)
	UpdateMTFStopLossOrder(kc *kiteconnect.Client, orderID string, newPrice float64, newTriggerPrice float64) error
}

type ZerodhaServiceImpl struct {
	zerodhaConfig *model.ZerodhaConfig
}

func NewZerodhaService(zc *model.ZerodhaConfig) ZerodhaService {
	fmt.Println(zc)
	return &ZerodhaServiceImpl{
		zerodhaConfig: zc,
	}
}

func (s *ZerodhaServiceImpl) GenerateAccessToken(requestToken string, userId int64) (string, error) {

	kc := kiteconnect.New(s.zerodhaConfig.ApiKey)

	userSession, err := kc.GenerateSession(requestToken, s.zerodhaConfig.ApiSecret)
	if err != nil {
		return "", err
	}

	return userSession.AccessToken, nil
}

func (s *ZerodhaServiceImpl) InitiateKiteConnect(ctx context.Context, accessToken string) (*kiteconnect.Client, error) {
	kc := kiteconnect.New(s.zerodhaConfig.ApiKey)
	kc.SetAccessToken(accessToken)
	return kc, nil
}

func (s *ZerodhaServiceImpl) PlaceMTFOrder(kc *kiteconnect.Client, symbol string, qty int, price float64) (kiteconnect.OrderResponse, error) {
	orderParams := kiteconnect.OrderParams{
		Exchange:        kiteconnect.ExchangeNSE,
		Tradingsymbol:   symbol,
		TransactionType: kiteconnect.TransactionTypeBuy,
		Quantity:        qty,
		Price:           price,
		Product:         kiteconnect.ProductMTF,
		OrderType:       kiteconnect.OrderTypeMarket,
		Validity:        kiteconnect.ValidityDay,
	}

	return kc.PlaceOrder(kiteconnect.VarietyRegular, orderParams)
}

func (s *ZerodhaServiceImpl) PlaceMTFStopLossOrder(kc *kiteconnect.Client, symbol string, qty int, price float64, triggerPrice float64) (string, error) {
	orderParams := kiteconnect.OrderParams{
		Exchange:        kiteconnect.ExchangeNSE,
		Tradingsymbol:   symbol,
		TransactionType: kiteconnect.TransactionTypeSell,
		Quantity:        qty,
		Price:           price,
		TriggerPrice:    triggerPrice,
		Product:         kiteconnect.ProductMTF,
		OrderType:       kiteconnect.OrderTypeSL,
		Validity:        kiteconnect.ValidityDay,
	}

	orderResponse, err := kc.PlaceOrder(kiteconnect.VarietyRegular, orderParams)
	if err != nil {
		return "", err
	}

	return orderResponse.OrderID, nil
}

func (s *ZerodhaServiceImpl) GetOrderDetails(kc *kiteconnect.Client, orderID string) (kiteconnect.Order, error) {
	history, err := kc.GetOrderHistory(orderID)
	if err != nil {
		return kiteconnect.Order{}, err
	}

	if len(history) == 0 {
		return kiteconnect.Order{}, fmt.Errorf("no history found for order id %s", orderID)
	}

	currentOrderState := history[len(history)-1]

	return currentOrderState, nil
}

func (s *ZerodhaServiceImpl) UpdateMTFStopLossOrder(kc *kiteconnect.Client, orderID string, newPrice float64, newTriggerPrice float64) error {
	modParams := kiteconnect.OrderParams{
		Price:        newPrice,
		TriggerPrice: newTriggerPrice,
	}

	_, err := kc.ModifyOrder(kiteconnect.VarietyRegular, orderID, modParams)
	return err
}
