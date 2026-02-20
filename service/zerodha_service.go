package service

import (
	"backend/cache"
	"backend/database"
	"backend/util"
	"context"
	"fmt"
	"strconv"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

type ZerodhaService interface {
	InitiateKiteConnect(ctx context.Context, accessToken string, userId int64) (*kiteconnect.Client, error)
	GenerateAccessToken(requestToken string, userId int64) (string, error)
	PlaceMTFOrder(kc *kiteconnect.Client, symbol string, qty int, price float64, transactionType, orderType string) (kiteconnect.OrderResponse, error)
	PlaceMTFStopLossOrder(kc *kiteconnect.Client, symbol string, qty int, price float64, triggerPrice float64) (string, error)
	GetOrderDetails(kc *kiteconnect.Client, orderID string) (kiteconnect.Order, error)
	UpdateMTFStopLossOrder(kc *kiteconnect.Client, orderID string, newPrice float64, newTriggerPrice float64) error
	CancelOrder(kc *kiteconnect.Client, orderID string) (kiteconnect.OrderResponse, error)
	ConvertSLToMarket(kc *kiteconnect.Client, orderID string, quantity int, price float64) (kiteconnect.OrderResponse, error)
	GetKiteClient(ctx context.Context, userID int64) (*kiteconnect.Client, error)
}

type ZerodhaServiceImpl struct {
	userSvc UserService
}

func NewZerodhaService(userSvc UserService) ZerodhaService {
	return &ZerodhaServiceImpl{
		userSvc: userSvc,
	}
}

func (s *ZerodhaServiceImpl) GenerateAccessToken(requestToken string, userId int64) (string, error) {

	user, err := s.userSvc.FindUser(context.Background(), 0, "", userId)
	if err != nil {
		return "", err
	}

	if user.ZerodhaConfig.ApiKey == "" || user.ZerodhaConfig.ApiSecret == "" {
		return "", fmt.Errorf("zerodha config not found")
	}

	kc := kiteconnect.New(user.ZerodhaConfig.ApiKey)

	userSession, err := kc.GenerateSession(requestToken, user.ZerodhaConfig.ApiSecret)
	if err != nil {
		return "", err
	}

	return userSession.AccessToken, nil
}

func (s *ZerodhaServiceImpl) InitiateKiteConnect(ctx context.Context, accessToken string, userId int64) (*kiteconnect.Client, error) {
	user, err := s.userSvc.FindUser(context.Background(), 0, "", userId)

	if err != nil {
		return nil, err
	}

	if user.ZerodhaConfig.ApiKey == "" || user.ZerodhaConfig.ApiSecret == "" {
		return nil, fmt.Errorf("zerodha config not found")
	}

	kc := kiteconnect.New(user.ZerodhaConfig.ApiKey)
	kc.SetAccessToken(accessToken)
	return kc, nil
}

func (s *ZerodhaServiceImpl) PlaceMTFOrder(kc *kiteconnect.Client, symbol string, qty int, price float64, transactionType, orderType string) (kiteconnect.OrderResponse, error) {
	orderParams := kiteconnect.OrderParams{
		Exchange:        kiteconnect.ExchangeNSE,
		Tradingsymbol:   symbol,
		TransactionType: transactionType,
		Quantity:        qty,
		Price:           price,
		Product:         kiteconnect.ProductMTF,
		OrderType:       orderType,
		Validity:        kiteconnect.ValidityDay,
		Tag:             "Shahbaz Trades",
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

func (s *ZerodhaServiceImpl) CancelOrder(kc *kiteconnect.Client, orderID string) (kiteconnect.OrderResponse, error) {
	orderResponse, err := kc.CancelOrder(kiteconnect.VarietyRegular, orderID, nil)
	if err != nil {
		return kiteconnect.OrderResponse{}, fmt.Errorf("failed to cancel order %s: %w", orderID, err)
	}

	return orderResponse, nil
}

func (s *ZerodhaServiceImpl) ConvertSLToMarket(kc *kiteconnect.Client, orderID string, quantity int, price float64) (kiteconnect.OrderResponse, error) {
	params := kiteconnect.OrderParams{
		OrderType: kiteconnect.OrderTypeMarket,
		Quantity:  quantity,
		Price:     price,
	}

	return kc.ModifyOrder(kiteconnect.VarietyRegular, orderID, params)
}

func (s *ZerodhaServiceImpl) GetKiteClient(ctx context.Context, userID int64) (*kiteconnect.Client, error) {
	userIdStr := strconv.FormatInt(userID, 10)

	// Check local memory cache
	if val, ok := cache.KiteClientCache.Get(userIdStr); ok {
		if kc, ok := val.(*kiteconnect.Client); ok {
			return kc, nil
		}
	}

	// Check Redis for access token
	var accessToken string
	ok, err := database.RedisHelper.GetAsStruct("zerodha_token_"+userIdStr, &accessToken)
	if err != nil || !ok || accessToken == "" {
		return nil, fmt.Errorf("access token not found in redis for user %d", userID)
	}

	// Initiate KiteConnect
	kc, err := s.InitiateKiteConnect(ctx, accessToken, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate kite connect: %w", err)
	}

	// Cache the client
	cache.KiteClientCache.Set(userIdStr, kc, util.ZerodhaTokenExpiry())

	return kc, nil
}
