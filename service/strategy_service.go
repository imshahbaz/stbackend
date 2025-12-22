package service

import (
	"context"
	"sync"

	"backend/model"      // Replace with your actual path
	"backend/repository" // Replace with your actual path
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

	newMap := make(map[string]model.StrategyDto)
	for _, strategy := range strategies {
		if strategy.Active {
			// Converting Entity to DTO (Assuming you have a ToDto method)
			dto := model.StrategyDto{
				Name:       strategy.Name,
				ScanClause: strategy.ScanClause,
				Active:     strategy.Active,
			}
			newMap[dto.Name] = dto
		}
	}

	s.mu.Lock()
	s.strategyMap = newMap
	s.mu.Unlock()
	return nil
}

func (s *StrategyServiceImpl) GetAllStrategies() []model.StrategyDto {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]model.StrategyDto, 0, len(s.strategyMap))
	for _, v := range s.strategyMap {
		list = append(list, v)
	}
	return list
}

func (s *StrategyServiceImpl) CreateStrategy(ctx context.Context, request model.StrategyDto) (model.StrategyDto, error) {
	entity := request.ToEntity()
	err := s.repo.Save(ctx, entity)
	if err != nil {
		return model.StrategyDto{}, err
	}

	// Update local cache if active
	if entity.Active {
		s.mu.Lock()
		s.strategyMap[entity.Name] = request
		s.mu.Unlock()
	}

	return request, nil
}

func (s *StrategyServiceImpl) UpdateStrategy(ctx context.Context, request model.StrategyDto) (model.StrategyDto, error) {
	// In MongoDB, Save (Upsert) handles both Create and Update
	return s.CreateStrategy(ctx, request)
}

func (s *StrategyServiceImpl) DeleteStrategy(ctx context.Context, id string) error {
	err := s.repo.DeleteById(ctx, id)
	if err != nil {
		return err
	}

	s.mu.Lock()
	delete(s.strategyMap, id)
	s.mu.Unlock()
	return nil
}
