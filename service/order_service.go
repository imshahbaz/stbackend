package service

import (
	"context"
	"strconv"
	"time"

	"backend/database"
	"backend/model"
	"backend/repository"
	"backend/util"

	"github.com/rs/zerolog/log"
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
}

type OrderServiceImpl struct {
	repo       *repository.OrderRepo
	zerodhaSvc ZerodhaService
}

func NewOrderService(repo *repository.OrderRepo, zerodhaSvc ZerodhaService) OrderService {
	return &OrderServiceImpl{repo: repo, zerodhaSvc: zerodhaSvc}
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

func (s *OrderServiceImpl) InitiateMtfOrders(ctx context.Context) {
	orders, err := s.GetTodaysOrders(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch today's orders for MTF initiation")
		return
	}

	if len(orders) == 0 {
		log.Info().Msg("No orders found for today to initiate MTF")
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

			kc, err := s.zerodhaSvc.InitiateKiteConnect(context.Background(), accessToken)
			if err != nil {
				log.Error().Err(err).Int64("userId", uid).Msg("Failed to initiate KiteConnect for user")
				return
			}

			for _, ord := range list {
				orderResponse, err := s.zerodhaSvc.PlaceMTFOrder(kc, ord.Symbol, ord.Quantity, 0)
				if err != nil {
					log.Error().Err(err).Str("symbol", ord.Symbol).Int64("userId", uid).Msg("Failed to place MTF order")
				} else {
					log.Info().Str("orderId", orderResponse.OrderID).Str("symbol", ord.Symbol).Int64("userId", uid).Msg("MTF order placed successfully")
					ord.BuyOrder = model.OrderInfo{
						OrderID: orderResponse.OrderID,
					}
					go s.repo.PatchStruct(ctx, ord.ID, ord)
				}
			}
		}(userID, oList)
	}
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
