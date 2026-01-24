package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"backend/cache"
	"backend/database"
	"backend/model"
	"backend/repository"
	"backend/util"

	"github.com/rs/zerolog/log"
	kiteconnect "github.com/zerodha/gokiteconnect/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrderService interface {
	Get(ctx context.Context, id string) (*model.OrderDto, error)
	GetAll(ctx context.Context) ([]model.OrderDto, error)
	GetAllByDate(ctx context.Context, date string) ([]model.OrderDto, error)
	GetAllByUserId(ctx context.Context, userId int64) ([]model.OrderDto, error)
	GetTodaysOrders(ctx context.Context) ([]model.Order, error)
	InitiateMtfOrders(ctx context.Context)
	Create(ctx context.Context, dto model.OrderDto) error
	Update(ctx context.Context, id string, dto model.OrderDto) error
	Delete(ctx context.Context, id string) error
	UpdateOrderStatus(ctx context.Context)
	StartTrading(ctx context.Context)
}

type OrderServiceImpl struct {
	repo       *repository.OrderRepo
	zerodhaSvc ZerodhaService
	angelSvc   AngelOneService
}

func NewOrderService(repo *repository.OrderRepo, zerodhaSvc ZerodhaService, angelSvc AngelOneService) OrderService {
	return &OrderServiceImpl{repo: repo, zerodhaSvc: zerodhaSvc, angelSvc: angelSvc}
}

func (s *OrderServiceImpl) Get(ctx context.Context, id string) (*model.OrderDto, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	order, err := s.repo.Get(ctx, objID)
	if err != nil || order == nil {
		return nil, err
	}

	return s.toDto(order), nil
}

func (s *OrderServiceImpl) GetAll(ctx context.Context) ([]model.OrderDto, error) {
	orders, err := s.repo.GetAll(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	dtos := make([]model.OrderDto, len(orders))
	for i, o := range orders {
		dtos[i] = *s.toDto(&o)
	}
	return dtos, nil
}

func (s *OrderServiceImpl) GetAllByDate(ctx context.Context, dateStr string) ([]model.OrderDto, error) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		t, err = time.Parse(time.RFC3339, dateStr)
	}

	if err != nil {
		return nil, err
	}

	// We want everything from the start of the day to the end of the day in IST
	startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, util.IstLocation)
	endOfDay := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, util.IstLocation)

	filter := bson.M{
		"date": bson.M{
			"$gte": startOfDay,
			"$lte": endOfDay,
		},
	}
	orders, err := s.repo.GetAll(ctx, filter)
	if err != nil {
		return nil, err
	}

	dtos := make([]model.OrderDto, len(orders))
	for i, o := range orders {
		dtos[i] = *s.toDto(&o)
	}
	return dtos, nil
}

func (s *OrderServiceImpl) GetAllByUserId(ctx context.Context, userId int64) ([]model.OrderDto, error) {
	filter := bson.M{"userId": userId}
	orders, err := s.repo.GetAll(ctx, filter)
	if err != nil {
		return nil, err
	}

	dtos := make([]model.OrderDto, len(orders))
	for i, o := range orders {
		dtos[i] = *s.toDto(&o)
	}
	return dtos, nil
}

func (s *OrderServiceImpl) GetTodaysOrders(ctx context.Context) ([]model.Order, error) {
	today := util.ToIST(time.Now())
	startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, util.IstLocation)
	endOfDay := time.Date(today.Year(), today.Month(), today.Day(), 23, 59, 59, 999999999, util.IstLocation)

	filter := bson.M{
		"date": bson.M{
			"$gte": startOfDay,
			"$lte": endOfDay,
		},
	}
	return s.repo.GetAll(ctx, filter)
}

func (s *OrderServiceImpl) processTodayOrdersPerUser(ctx context.Context, taskName string, processFn func(kc *kiteconnect.Client, ord *model.Order) error) {
	orders, err := s.GetTodaysOrders(ctx)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to fetch today's orders for %s", taskName)
		return
	}

	if len(orders) == 0 {
		log.Info().Msgf("No orders found for today to %s", taskName)
		return
	}

	userOrders := make(map[int64][]model.Order)
	for _, o := range orders {
		userOrders[o.UserID] = append(userOrders[o.UserID], o)
	}

	for userID, oList := range userOrders {
		go func(uid int64, list []model.Order) {
			var accessToken string
			ok, _ := database.RedisHelper.GetAsStruct("zerodha_token_"+strconv.FormatInt(uid, 10), &accessToken)
			if !ok || accessToken == "" {
				log.Warn().Int64("userId", uid).Msg("AccessToken not found in Redis for user")
				return
			}

			kc, err := s.zerodhaSvc.InitiateKiteConnect(context.Background(), accessToken, uid)
			if err != nil {
				log.Error().Err(err).Int64("userId", uid).Msg("Failed to initiate KiteConnect for user")
				return
			}

			for _, ord := range list {
				if err := processFn(kc, &ord); err != nil {
					log.Error().Err(err).Str("symbol", ord.Symbol).Int64("userId", uid).Msgf("Error processing order in %s", taskName)
					continue
				}

				objID, err := primitive.ObjectIDFromHex(ord.ID)
				if err != nil {
					log.Error().Err(err).Str("symbol", ord.Symbol).Int64("userId", uid).Msg("Invalid ObjectID during update")
					continue
				}

				_, err = s.repo.PatchStruct(context.Background(), objID, ord)
				if err != nil {
					log.Error().Err(err).Str("symbol", ord.Symbol).Int64("userId", uid).Msg("Failed to update order in database")
				}
			}
		}(userID, oList)
	}
}

func (s *OrderServiceImpl) InitiateMtfOrders(ctx context.Context) {
	s.processTodayOrdersPerUser(ctx, "Initiate MTF", func(kc *kiteconnect.Client, ord *model.Order) error {
		if ord.BuyOrder.OrderID != "" {
			log.Info().Str("orderId", ord.BuyOrder.OrderID).Str("symbol", ord.Symbol).Int64("userId", ord.UserID).Msg("MTF order already placed")
			return nil
		}

		orderResponse, err := s.zerodhaSvc.PlaceMTFOrder(kc, ord.Symbol, ord.Quantity, 0)
		if err != nil {
			return err
		}

		log.Info().Str("orderId", orderResponse.OrderID).Str("symbol", ord.Symbol).Int64("userId", ord.UserID).Msg("MTF order placed successfully")
		ord.BuyOrder = model.OrderInfo{
			OrderID: orderResponse.OrderID,
		}
		return nil
	})
}

func (s *OrderServiceImpl) Create(ctx context.Context, dto model.OrderDto) error {
	var orderDate time.Time
	if dto.Date != "" {
		orderDate, _ = time.Parse(time.RFC3339, dto.Date)
		if orderDate.IsZero() {
			orderDate, _ = time.Parse("2006-01-02", dto.Date)
		}
	}

	if orderDate.IsZero() {
		orderDate = util.GetISTMidnight()
	} else {
		orderDate = util.ToIST(orderDate)
	}

	order := model.Order{
		UserID:   dto.UserID,
		Symbol:   dto.Symbol,
		Quantity: dto.Quantity,
		Date:     orderDate,
	}
	return s.repo.Insert(ctx, order)
}

func (s *OrderServiceImpl) Update(ctx context.Context, id string, dto model.OrderDto) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	var orderDate time.Time
	if dto.Date != "" {
		orderDate, _ = time.Parse(time.RFC3339, dto.Date)
		if orderDate.IsZero() {
			orderDate, _ = time.Parse("2006-01-02", dto.Date)
		}
	}

	if !orderDate.IsZero() {
		orderDate = util.ToIST(orderDate)
	}

	updateData := model.Order{
		UserID:   dto.UserID,
		Symbol:   dto.Symbol,
		Quantity: dto.Quantity,
		Date:     orderDate,
	}

	_, err = s.repo.PatchStruct(ctx, objID, updateData)
	return err
}

func (s *OrderServiceImpl) Delete(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, objID)
}

func (s *OrderServiceImpl) toDto(o *model.Order) *model.OrderDto {
	return &model.OrderDto{
		ID:       o.ID,
		UserID:   o.UserID,
		Symbol:   o.Symbol,
		Quantity: o.Quantity,
		Date:     util.ToIST(o.Date).Format(time.RFC3339),
	}
}

func (s *OrderServiceImpl) UpdateOrderStatus(ctx context.Context) {
	s.processTodayOrdersPerUser(ctx, "Update MTF Status", func(kc *kiteconnect.Client, ord *model.Order) error {
		if ord.BuyOrder.OrderID == "" {
			log.Info().Str("symbol", ord.Symbol).Int64("userId", ord.UserID).Msg("MTF order not found, skipping status update")
			return nil
		}

		orderResponse, err := s.zerodhaSvc.GetOrderDetails(kc, ord.BuyOrder.OrderID)
		if err != nil {
			return err
		}

		if orderResponse.Status != "" {
			ord.BuyOrder.OrderStatus = orderResponse.Status
		}

		if orderResponse.AveragePrice > 0 {
			ord.BuyOrder.AveragePrice = orderResponse.AveragePrice
		}

		return nil
	})
}

func (s *OrderServiceImpl) StartTrading(ctx context.Context) {
	duration := 1 * time.Minute
	orders, err := s.GetTodaysOrders(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch today's orders")
		return
	}
	for _, order := range orders {
		bgCtx, cancel := context.WithTimeout(context.Background(), duration)
		go s.processOrder(bgCtx, cancel, order)
	}
}

func (s *OrderServiceImpl) processOrder(ctx context.Context, cancel context.CancelFunc, order model.Order) {
	defer cancel()

	var accessToken string
	ok, _ := database.RedisHelper.GetAsStruct("zerodha_token_"+strconv.FormatInt(order.UserID, 10), &accessToken)
	if !ok || accessToken == "" {
		log.Warn().Int64("userId", order.UserID).Msg("AccessToken not found in Redis for user")
		return
	}

	kc, err := s.zerodhaSvc.InitiateKiteConnect(context.Background(), accessToken, order.UserID)
	if err != nil {
		log.Error().Err(err).Int64("userId", order.UserID).Msg("Failed to initiate KiteConnect for user")
		return
	}

	buyPrice := order.BuyOrder.AveragePrice
	if buyPrice == 0 {
		log.Error().Str("symbol", order.Symbol).Msg("Buy price not found")
		return
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	val, ok := cache.MarginCache.Get(order.Symbol)
	if !ok {
		log.Error().Str("symbol", order.Symbol).Msg("Margin not found")
		return
	}

	margin := val.(model.Margin)

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("Order %s stopped externally\n", order.ID)
			return
		case <-ticker.C:
			currentPrice, err := s.angelSvc.GetLTP(margin.Symbol, margin.Token)
			if err != nil {
				log.Error().Err(err).Str("symbol", order.Symbol).Msg("Failed to get LTP")
				return
			}
			s.addStopLoss(order, currentPrice, buyPrice, kc, cancel)
		}
	}
}

func (s *OrderServiceImpl) addStopLoss(order model.Order, ltp, buyPrice float64, kc *kiteconnect.Client, cancel context.CancelFunc) {
	if order.StopLossOrder.OrderID == "" {
		if ltp >= buyPrice*1.01 {
			sl := util.FixToTick(buyPrice * 1.004)
			orderId, err := s.zerodhaSvc.PlaceMTFStopLossOrder(kc, order.Symbol, order.Quantity, sl, sl)
			if err != nil {
				log.Error().Err(err).Str("symbol", order.Symbol).Msg("Failed to place stop loss order")
				return
			}
			order.StopLossOrder.OrderID = orderId
			objID, err := primitive.ObjectIDFromHex(order.ID)
			if err != nil {
				log.Error().Err(err).Str("symbol", order.Symbol).Int64("userId", order.UserID).Msg("Invalid ObjectID during update")
				return
			}
			s.repo.PatchStruct(context.Background(), objID, order)
		}
	} else {
		orderResp, err := s.zerodhaSvc.GetOrderDetails(kc, order.StopLossOrder.OrderID)
		if err != nil {
			log.Error().Err(err).Str("symbol", order.Symbol).Msg("Failed to get order details")
			return
		}

		if orderResp.Status == "COMPLETE" {
			cancel()
			return
		}

		oldSl := orderResp.Price
		threshold := buyPrice * 0.01
		gap := buyPrice * 0.008

		if ltp >= oldSl+threshold {
			newSl := util.FixToTick(ltp - gap)
			err := s.zerodhaSvc.UpdateMTFStopLossOrder(kc, order.StopLossOrder.OrderID, newSl, newSl)
			if err != nil {
				log.Error().Err(err).Str("symbol", order.Symbol).Msg("Failed to update stop loss order")
				return
			}
		}

	}
}
