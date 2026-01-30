package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"backend/cache"
	"backend/database"
	"backend/model"
	"backend/repository"
	"backend/util"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
	kiteconnect "github.com/zerodha/gokiteconnect/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	mu                sync.Mutex
	oneMinuteCron     *cron.Cron
	fifteenMinuteCron *cron.Cron
	updateChan        chan model.Order
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
	ResetTradingSystem(ctx context.Context) error
	StartTradingWs(ctx context.Context)
}

type OrderServiceImpl struct {
	repo       *repository.OrderRepo
	zerodhaSvc ZerodhaService
	angelSvc   AngelOneService
	angelOneWs AngelOneWebSocket
}

func NewOrderService(repo *repository.OrderRepo, zerodhaSvc ZerodhaService, angelSvc AngelOneService, angelOneWs AngelOneWebSocket) OrderService {
	return &OrderServiceImpl{repo: repo, zerodhaSvc: zerodhaSvc, angelSvc: angelSvc, angelOneWs: angelOneWs}
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
			var kc *kiteconnect.Client
			var err error
			val, ok := cache.KiteClientCache.Get(strconv.FormatInt(uid, 10))
			if !ok {
				var accessToken string
				ok, _ := database.RedisHelper.GetAsStruct("zerodha_token_"+strconv.FormatInt(uid, 10), &accessToken)
				if !ok || accessToken == "" {
					log.Warn().Int64("userId", uid).Msg("AccessToken not found in Redis for user")
					return
				}

				kc, err = s.zerodhaSvc.InitiateKiteConnect(context.Background(), accessToken, uid)
				if err != nil {
					log.Error().Err(err).Int64("userId", uid).Msg("Failed to initiate KiteConnect for user")
					return
				}
				cache.KiteClientCache.Set(strconv.FormatInt(uid, 10), kc, util.ZerodhaTokenExpiry())
			} else {
				kc = val.(*kiteconnect.Client)
			}

			for i := range list {
				if err := processFn(kc, &list[i]); err != nil {
					log.Error().Err(err).Str("symbol", list[i].Symbol).Int64("userId", uid).Msgf("Error processing order in %s", taskName)
					continue
				}

				objID, err := primitive.ObjectIDFromHex(list[i].ID)
				if err != nil {
					log.Error().Err(err).Str("symbol", list[i].Symbol).Int64("userId", uid).Msg("Invalid ObjectID during update")
					continue
				}

				_, err = s.repo.PatchStruct(context.Background(), objID, list[i])
				if err != nil {
					log.Error().Err(err).Str("symbol", list[i].Symbol).Int64("userId", uid).Msg("Failed to update order in database")
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

		orderResponse, err := s.zerodhaSvc.PlaceMTFOrder(kc, ord.Symbol, ord.Quantity, 0, kiteconnect.TransactionTypeBuy)
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

	val, ok := cache.MarginCache.Get(dto.Symbol)
	if !ok {
		return errors.New("Margin not found")
	}

	margin := val.(model.Margin)
	order := model.Order{
		UserID:   dto.UserID,
		Symbol:   dto.Symbol,
		Quantity: dto.Quantity,
		Date:     orderDate,
		Margin:   margin,
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

	val, ok := cache.MarginCache.Get(dto.Symbol)
	if !ok {
		return errors.New("Margin not found")
	}

	margin := val.(model.Margin)

	updateData := model.Order{
		UserID:   dto.UserID,
		Symbol:   dto.Symbol,
		Quantity: dto.Quantity,
		Date:     orderDate,
		Margin:   margin,
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

	orders, err := s.GetTodaysOrders(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch today's orders")
		return
	}

	if len(orders) == 0 {
		log.Info().Msg("No orders found for today")
		return
	}

	mu.Lock()
	if oneMinuteCron != nil {
		oneMinuteCron.Stop()
		oneMinuteCron = nil
	}

	if fifteenMinuteCron != nil {
		fifteenMinuteCron.Stop()
		fifteenMinuteCron = nil
	}
	mu.Unlock()

	firstSlTradeManager := util.NewTradeManager()
	trailingSlTradeManager := util.NewTradeManager()

	newMap := make(map[string]*model.Order, len(orders))

	for i := range orders {
		newMap[orders[i].ID] = &orders[i]
	}

	firstSlTradeManager.ReplaceAll(newMap)

	oneMinuteCron, err = util.ScheduleTask(util.OneMinSpec, func() {
		orders := firstSlTradeManager.GetActiveList()
		if len(orders) == 0 {
			return
		}

		tokens := s.extractTokens(orders)

		ltpMap, err := s.angelSvc.GetMultipleLTP(tokens)
		if err != nil {
			log.Error().Err(err).Msg("Failed to fetch LTP")
			return
		}

		for _, order := range orders {
			if ltp, ok := ltpMap[order.Margin.Token]; ok {
				go func(o *model.Order, ltp float64) {
					res := s.processOrder(o, ltp)
					if res < 0 {
						firstSlTradeManager.RemoveTrade(o.ID)
					} else if res == 1 {
						firstSlTradeManager.RemoveTrade(o.ID)
						trailingSlTradeManager.AddTrade(o)
					}
				}(order, ltp)
			}
		}
	})

	fifteenMinuteCron, err = util.ScheduleTask(util.FifteenMinSpec, func() {
		orders := trailingSlTradeManager.GetActiveList()
		if len(orders) == 0 {
			return
		}

		tokens := s.extractTokens(orders)

		ltpMap, err := s.angelSvc.GetMultipleLTP(tokens)
		if err != nil {
			log.Error().Err(err).Msg("Failed to fetch LTP")
			return
		}

		for _, order := range orders {
			if ltp, ok := ltpMap[order.Margin.Token]; ok {
				go func(o *model.Order, ltp float64) {
					res := s.processOrder(o, ltp)
					if res < 0 {
						trailingSlTradeManager.RemoveTrade(o.ID)
					}
				}(order, ltp)
			}
		}
	})
}

func (s *OrderServiceImpl) extractTokens(orders []*model.Order) []string {
	tokens := make([]string, len(orders))
	for i, o := range orders {
		tokens[i] = o.Margin.Token
	}
	return tokens
}

func (s *OrderServiceImpl) processOrder(order *model.Order, ltp float64) int8 {
	var kc *kiteconnect.Client
	val, ok := cache.KiteClientCache.Get(strconv.FormatInt(order.UserID, 10))
	if !ok {
		var accessToken string
		ok, _ := database.RedisHelper.GetAsStruct("zerodha_token_"+strconv.FormatInt(order.UserID, 10), &accessToken)
		if !ok || accessToken == "" {
			log.Warn().Int64("userId", order.UserID).Msg("AccessToken not found in Redis for user")
			return -1
		}

		var err error
		kc, err = s.zerodhaSvc.InitiateKiteConnect(context.Background(), accessToken, order.UserID)
		if err != nil {
			log.Error().Err(err).Int64("userId", order.UserID).Msg("Failed to initiate KiteConnect for user")
			return -1
		}
		cache.KiteClientCache.Set(strconv.FormatInt(order.UserID, 10), kc, util.ZerodhaTokenExpiry())
	} else {
		kc = val.(*kiteconnect.Client)
	}

	buyPrice := order.BuyOrder.AveragePrice
	if buyPrice == 0 {
		log.Error().Str("symbol", order.Symbol).Msg("Buy price not found")
		return -1
	}

	return s.addStopLoss(order, ltp, buyPrice, kc)
}

func (s *OrderServiceImpl) addStopLoss(order *model.Order, ltp, buyPrice float64, kc *kiteconnect.Client) int8 {
	if util.IsTimePastClosingGrace() && ltp > order.BuyOrder.AveragePrice*1.004 {
		log.Info().Str("symbol", order.Symbol).Msg("Market closing (3:25 PM). Squaring off...")

		if order.StopLossOrder.OrderID == "" {
			_, err := s.zerodhaSvc.PlaceMTFOrder(kc, order.Symbol, order.Quantity, 0, kiteconnect.TransactionTypeSell)
			if err != nil {
				log.Error().Err(err).Str("symbol", order.Symbol).Msg("Failed square-off")
				return 0
			}
		} else {
			orderResp, err := s.zerodhaSvc.GetOrderDetails(kc, order.StopLossOrder.OrderID)
			if err != nil {
				return 0
			}

			if orderResp.Status == "COMPLETE" || orderResp.Status == "REJECTED" {
				return -1
			}

			qtyLeft := int(orderResp.PendingQuantity)

			if qtyLeft > 0 {
				_, err = s.zerodhaSvc.ConvertSLToMarket(kc, order.StopLossOrder.OrderID, qtyLeft, 0)
				if err != nil {
					log.Error().Err(err).Msg("Conversion to Market failed")
					return 0
				}
			}
		}
		return -1
	}

	if order.StopLossOrder.OrderID == "" {
		if ltp >= buyPrice*1.008 {
			sl := util.FixToTick(buyPrice * 1.004)
			orderId, err := s.zerodhaSvc.PlaceMTFStopLossOrder(kc, order.Symbol, order.Quantity, sl, sl)
			if err != nil {
				log.Error().Err(err).Str("symbol", order.Symbol).Msg("Failed to place stop loss order")
				return 0
			}
			order.StopLossOrder.OrderID = orderId
			order.StopLossOrder.AveragePrice = sl
			pushToDb(order)
			return 1
		}
		return 0
	} else {
		if order.StopLossOrder.AveragePrice == 0 {
			orderResp, err := s.zerodhaSvc.GetOrderDetails(kc, order.StopLossOrder.OrderID)
			if err != nil {
				log.Error().Err(err).Str("symbol", order.Symbol).Msg("Failed to get order details")
				return 0
			}
			order.StopLossOrder.AveragePrice = orderResp.Price
			order.StopLossOrder.OrderStatus = orderResp.Status
			pushToDb(order)
			if orderResp.Status == "COMPLETE" {
				return -1
			}
		}

		oldSl := order.StopLossOrder.AveragePrice
		threshold := buyPrice * 0.01
		gap := buyPrice * 0.008

		if ltp >= oldSl+threshold {
			newSl := util.FixToTick(ltp - gap)
			err := s.zerodhaSvc.UpdateMTFStopLossOrder(kc, order.StopLossOrder.OrderID, newSl, newSl)
			if err != nil {
				if strings.Contains(err.Error(), "processed") {
					return -1
				}
				log.Error().Err(err).Str("symbol", order.Symbol).Msg("Failed to update stop loss order")
				return 0
			}

			order.StopLossOrder.AveragePrice = newSl
			pushToDb(order)
			return 1
		}
		return 0
	}
}

func (s *OrderServiceImpl) ResetTradingSystem(ctx context.Context) error {
	mu.Lock()
	defer mu.Unlock()

	log.Info().Msg("Manual System Reset Triggered via Controller")

	if oneMinuteCron != nil {
		oneMinuteCron.Stop()
		oneMinuteCron = nil
	}
	if fifteenMinuteCron != nil {
		fifteenMinuteCron.Stop()
		fifteenMinuteCron = nil
	}

	log.Info().Msg("System state cleared. Ready for re-initialization.")
	return nil
}

func (s *OrderServiceImpl) StartTradingWs(ctx context.Context) {
	orders, err := s.GetTodaysOrders(ctx)
	if err != nil || len(orders) == 0 {
		return
	}

	s.startWorkers()

	for i := range orders {
		priceChan, err := s.angelOneWs.Subscribe(orders[i].Margin.Token, model.NSE)
		if err != nil {
			continue
		}

		go func(order *model.Order, priceChan chan float64) {
			var prevLtp float64
			for ltp := range priceChan {
				if prevLtp == ltp {
					continue
				}
				res := s.processOrder(order, ltp)
				if res < 0 {
					log.Info().Str("orderId", order.ID).Str("symbol", order.Symbol).Msg("Order Squared Off")
					break
				}
				prevLtp = ltp
			}
		}(&orders[i], priceChan)
	}
}

func (s *OrderServiceImpl) startWorkers() {
	updateChan = make(chan model.Order, 500)

	for i := range 5 {
		go func(workerID int) {
			log.Debug().Int("workerID", workerID).Msg("DB Worker started")

			for order := range updateChan {
				objID, err := primitive.ObjectIDFromHex(order.ID)
				if err != nil {
					log.Error().Err(err).Str("symbol", order.Symbol).Msg("Invalid ObjectID")
					continue
				}

				_, err = s.repo.PatchStruct(context.Background(), objID, order)
				if err != nil {
					log.Error().Err(err).Str("orderId", order.ID).Msg("DB Patch failed")
				}
			}
		}(i)
	}
}

func pushToDb(order *model.Order) {
	select {
	case updateChan <- *order:
	default:
	}
}
