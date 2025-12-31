package service

import (
	"context"
	"time"

	"backend/cache"
	"backend/model"
	"backend/repository"
)

type StrategyService interface {
	ReloadAllStrategies(ctx context.Context) error
	GetAllStrategies() []model.StrategyDto
	CreateStrategy(ctx context.Context, request model.StrategyDto) (model.StrategyDto, error)
	UpdateStrategy(ctx context.Context, request model.StrategyDto) (model.StrategyDto, error)
	DeleteStrategy(ctx context.Context, id string) error
	GetAllStrategiesAdmin() []model.StrategyDto
}

type StrategyServiceImpl struct {
	repo *repository.StrategyRepository
}

func NewStrategyService(repo *repository.StrategyRepository) StrategyService {
	s := &StrategyServiceImpl{
		repo: repo,
	}

	_ = s.ReloadAllStrategies(context.Background())

	return s
}

func (s *StrategyServiceImpl) ReloadAllStrategies(ctx context.Context) error {
	strategies, err := s.repo.FindAll(ctx)
	if err != nil {
		return err
	}

	cache.StrategyCache.Flush()

	for _, strategy := range strategies {
		dto := model.StrategyDto{
			Name:       strategy.Name,
			ScanClause: strategy.ScanClause,
			Active:     strategy.Active,
		}

		cache.StrategyCache.Set(dto.Name, dto, -1)
	}

	return nil
}

func (s *StrategyServiceImpl) GetAllStrategies() []model.StrategyDto {
	return s.filterStrategies(false)
}

func (s *StrategyServiceImpl) GetAllStrategiesAdmin() []model.StrategyDto {
	return s.filterStrategies(true)
}

func (s *StrategyServiceImpl) CreateStrategy(ctx context.Context, request model.StrategyDto) (model.StrategyDto, error) {
	entity := request.ToEntity()
	if err := s.repo.Save(ctx, entity); err != nil {
		return model.StrategyDto{}, err
	}

	cache.StrategyCache.Set(request.Name, request, -1)

	go s.backgroundReload()

	return request, nil
}

func (s *StrategyServiceImpl) UpdateStrategy(ctx context.Context, request model.StrategyDto) (model.StrategyDto, error) {
	return s.CreateStrategy(ctx, request)
}

func (s *StrategyServiceImpl) DeleteStrategy(ctx context.Context, id string) error {
	if err := s.repo.DeleteById(ctx, id); err != nil {
		return err
	}

	cache.StrategyCache.Delete(id)

	return nil
}


func (s *StrategyServiceImpl) filterStrategies(includeInactive bool) []model.StrategyDto {
	items := cache.StrategyCache.Items()
	list := make([]model.StrategyDto, 0, len(items))

	for _, item := range items {
		if strategy, ok := item.Object.(model.StrategyDto); ok {
			if includeInactive || strategy.Active {
				list = append(list, strategy)
			}
		}
	}
	return list
}

func (s *StrategyServiceImpl) backgroundReload() {
	bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = s.ReloadAllStrategies(bgCtx)
}
