package service

import (
	"backend/model"
	"backend/repository"
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type StrategyOrderService interface {
	Create(ctx context.Context, dto model.StrategyOrderDto) (model.StrategyOrderDto, error)
	Get(ctx context.Context, id string) (model.StrategyOrderDto, error)
	GetAll(ctx context.Context, strategyName string) ([]model.StrategyOrderDto, error)
	Update(ctx context.Context, dto model.StrategyOrderDto) (model.StrategyOrderDto, error)
	Delete(ctx context.Context, id string) error
	GetByUserID(ctx context.Context, userId int64) ([]model.StrategyOrderDto, error)
}

type StrategyOrderServiceImpl struct {
	repo *repository.StrategyOrderRepository
}

func NewStrategyOrderService(repo *repository.StrategyOrderRepository) StrategyOrderService {
	return &StrategyOrderServiceImpl{repo: repo}
}

func (s *StrategyOrderServiceImpl) Create(ctx context.Context, dto model.StrategyOrderDto) (model.StrategyOrderDto, error) {
	entity, err := dto.ToEntity()
	if err != nil {
		return model.StrategyOrderDto{}, err
	}
	entity.ID = "" // Ensure ID is empty for new document
	err = s.repo.Generic.Insert(ctx, entity)
	if err != nil {
		return model.StrategyOrderDto{}, err
	}
	return dto, nil
}

func (s *StrategyOrderServiceImpl) Get(ctx context.Context, id string) (model.StrategyOrderDto, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return model.StrategyOrderDto{}, err
	}

	res, err := s.repo.Generic.Get(ctx, objID)
	if err != nil {
		return model.StrategyOrderDto{}, err
	}
	if res == nil {
		return model.StrategyOrderDto{}, nil
	}
	return res.ToDto(), nil
}

func (s *StrategyOrderServiceImpl) GetAll(ctx context.Context, strategyName string) ([]model.StrategyOrderDto, error) {
	filter := bson.M{}
	if strategyName != "" {
		filter["strategyName"] = strategyName
	}
	sort := bson.M{"date": -1}
	orders, err := s.repo.Generic.GetAllSorted(ctx, filter, sort)
	if err != nil {
		return nil, err
	}

	dtos := make([]model.StrategyOrderDto, 0, len(orders))
	for _, o := range orders {
		dtos = append(dtos, o.ToDto())
	}
	return dtos, nil
}

func (s *StrategyOrderServiceImpl) Update(ctx context.Context, dto model.StrategyOrderDto) (model.StrategyOrderDto, error) {
	objID, err := bson.ObjectIDFromHex(dto.ID)
	if err != nil {
		return model.StrategyOrderDto{}, err
	}

	entity, err := dto.ToEntity()
	if err != nil {
		return model.StrategyOrderDto{}, err
	}

	updateData := bson.M{
		"userId":       entity.UserID,
		"date":         entity.Date,
		"strategyName": entity.StrategyName,
		"amount":       entity.Amount,
	}

	updated, err := s.repo.Generic.UpdateFields(ctx, objID, updateData)
	if err != nil {
		return model.StrategyOrderDto{}, err
	}
	return updated.ToDto(), nil
}

func (s *StrategyOrderServiceImpl) Delete(ctx context.Context, id string) error {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	return s.repo.Generic.Delete(ctx, objID)
}

func (s *StrategyOrderServiceImpl) GetByUserID(ctx context.Context, userId int64) ([]model.StrategyOrderDto, error) {
	filter := bson.M{"userId": userId}
	sort := bson.M{"date": -1}
	orders, err := s.repo.Generic.GetAllSorted(ctx, filter, sort)
	if err != nil {
		return nil, err
	}

	dtos := make([]model.StrategyOrderDto, 0, len(orders))
	for _, o := range orders {
		dtos = append(dtos, o.ToDto())
	}
	return dtos, nil
}
