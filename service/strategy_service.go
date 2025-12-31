package service

import (
	"context"
	"time"

	"backend/cache"
	"backend/model"
	"backend/repository"
)

// StrategyService defines the contract for managing trading strategies.
type StrategyService interface {
	ReloadAllStrategies(ctx context.Context) error
	GetAllStrategies() []model.StrategyDto
	CreateStrategy(ctx context.Context, request model.StrategyDto) (model.StrategyDto, error)
	UpdateStrategy(ctx context.Context, request model.StrategyDto) (model.StrategyDto, error)
	DeleteStrategy(ctx context.Context, id string) error
	GetAllStrategiesAdmin() []model.StrategyDto
}

// StrategyServiceImpl implements StrategyService using a repository and a global cache.
type StrategyServiceImpl struct {
	repo *repository.StrategyRepository
}

// NewStrategyService initializes the service and performs an initial data load into the cache.
func NewStrategyService(repo *repository.StrategyRepository) StrategyService {
	s := &StrategyServiceImpl{
		repo: repo,
	}

	// Initial load to populate cache on startup
	_ = s.ReloadAllStrategies(context.Background())

	return s
}

// ReloadAllStrategies synchronizes the database state with the StrategyCache.
func (s *StrategyServiceImpl) ReloadAllStrategies(ctx context.Context) error {
	strategies, err := s.repo.FindAll(ctx)
	if err != nil {
		return err
	}

	// Flush cache to remove stale data before repopulating
	cache.StrategyCache.Flush()

	for _, strategy := range strategies {
		dto := model.StrategyDto{
			Name:       strategy.Name,
			ScanClause: strategy.ScanClause,
			Active:     strategy.Active,
		}

		// Set with NoExpiration (-1)
		cache.StrategyCache.Set(dto.Name, dto, -1)
	}

	return nil
}

// GetAllStrategies returns only active strategies for public consumption.
func (s *StrategyServiceImpl) GetAllStrategies() []model.StrategyDto {
	return s.filterStrategies(false)
}

// GetAllStrategiesAdmin returns all strategies (including inactive) for administrative use.
func (s *StrategyServiceImpl) GetAllStrategiesAdmin() []model.StrategyDto {
	return s.filterStrategies(true)
}

// CreateStrategy persists a new strategy and updates the cache immediately.
func (s *StrategyServiceImpl) CreateStrategy(ctx context.Context, request model.StrategyDto) (model.StrategyDto, error) {
	entity := request.ToEntity()
	if err := s.repo.Save(ctx, entity); err != nil {
		return model.StrategyDto{}, err
	}

	// Optimistic Cache Update: Update cache immediately so the user sees changes without delay
	cache.StrategyCache.Set(request.Name, request, -1)

	// Trigger a background full sync to ensure consistency
	go s.backgroundReload()

	return request, nil
}

// UpdateStrategy reuses the creation logic to ensure identical persistence/caching behavior.
func (s *StrategyServiceImpl) UpdateStrategy(ctx context.Context, request model.StrategyDto) (model.StrategyDto, error) {
	return s.CreateStrategy(ctx, request)
}

// DeleteStrategy removes a strategy from the repository and the cache.
func (s *StrategyServiceImpl) DeleteStrategy(ctx context.Context, id string) error {
	if err := s.repo.DeleteById(ctx, id); err != nil {
		return err
	}

	// Remove from cache using the ID (which acts as the Key)
	cache.StrategyCache.Delete(id)

	return nil
}

// --- Internal Helper Methods ---

// filterStrategies handles the repetitive logic of iterating through the cache.
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

// backgroundReload provides a safe way to refresh the cache in a separate goroutine.
func (s *StrategyServiceImpl) backgroundReload() {
	bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = s.ReloadAllStrategies(bgCtx)
}
