package service

import (
	"context"
	"sync"
	"time"

	"backend/cache"
	"backend/model"
	"backend/repository"
)

// 1. Interface Definition
type StrategyService interface {
	ReloadAllStrategies(ctx context.Context) error
	GetAllStrategies() []model.StrategyDto
	CreateStrategy(ctx context.Context, request model.StrategyDto) (model.StrategyDto, error)
	UpdateStrategy(ctx context.Context, request model.StrategyDto) (model.StrategyDto, error)
	DeleteStrategy(ctx context.Context, id string) error
}

// 2. Implementation Struct
type StrategyServiceImpl struct {
	repo        *repository.StrategyRepository
	strategyMap map[string]model.StrategyDto
	mu          sync.RWMutex
}

// NewStrategyService acts as @RequiredArgsConstructor + @PostConstruct
func NewStrategyService(repo *repository.StrategyRepository) StrategyService {
	s := &StrategyServiceImpl{
		repo:        repo,
		strategyMap: make(map[string]model.StrategyDto),
	}

	// Initial load
	_ = s.ReloadAllStrategies(context.Background())

	return s
}

// ReloadAllStrategies replaces the stream().filter().map() logic
func (s *StrategyServiceImpl) ReloadAllStrategies(ctx context.Context) error {
	strategies, err := s.repo.FindAll(ctx)
	if err != nil {
		return err
	}

	cache.StrategyCache.Flush()

	for _, strategy := range strategies {
		if strategy.Active {
			dto := model.StrategyDto{
				Name:       strategy.Name,
				ScanClause: strategy.ScanClause,
				Active:     strategy.Active,
			}

			// Set with NoExpiration to keep it indefinitely
			cache.StrategyCache.Set(dto.Name, dto, -1)
		}
	}

	return nil
}

func (s *StrategyServiceImpl) GetAllStrategies() []model.StrategyDto {
	items := cache.StrategyCache.Items()

	list := make([]model.StrategyDto, 0, len(s.strategyMap))
	for _, item := range items {
		if strategy, ok := item.Object.(model.StrategyDto); ok {
			list = append(list, strategy)
		}
	}

	return list
}

func (s *StrategyServiceImpl) CreateStrategy(ctx context.Context, request model.StrategyDto) (model.StrategyDto, error) {
	entity := request.ToEntity()
	err := s.repo.Save(ctx, entity)
	if err != nil {
		return model.StrategyDto{}, err
	}

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.ReloadAllStrategies(bgCtx)
	}()

	return request, nil
}

func (s *StrategyServiceImpl) UpdateStrategy(ctx context.Context, request model.StrategyDto) (model.StrategyDto, error) {
	return s.CreateStrategy(ctx, request)
}

func (s *StrategyServiceImpl) DeleteStrategy(ctx context.Context, id string) error {
	err := s.repo.DeleteById(ctx, id)
	if err != nil {
		return err
	}
	cache.StrategyCache.Delete(id)
	return nil
}
