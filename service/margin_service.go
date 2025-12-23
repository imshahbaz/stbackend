package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"

	"backend/model"
	"backend/repository"
	"backend/util"

	"github.com/patrickmn/go-cache"
)

// 1. Interface Definition (remains the same)
type MarginService interface {
	GetAllMargins() []model.Margin
	GetMargin(symbol string) (*model.Margin, bool)
	ReloadAllMargins(ctx context.Context) error
	LoadFromCsv(ctx context.Context, fileName string, file io.Reader) error
	GetStore() *cache.Cache
}

// 2. Implementation Struct
type MarginServiceImpl struct {
	repo     *repository.MarginRepository
	leverage float32
	store    *cache.Cache
}

// NewMarginService acts as the @RequiredArgsConstructor + @PostConstruct
func NewMarginService(repo *repository.MarginRepository, leverage float32) MarginService {
	s := &MarginServiceImpl{
		repo:     repo,
		leverage: leverage,
		store:    cache.New(cache.NoExpiration, 0),
	}

	// Trigger initial load (PostConstruct equivalent)
	ctx := context.Background()
	if err := s.ReloadAllMargins(ctx); err != nil {
		log.Printf("Warning: Failed initial margin load: %v", err)
	}

	return s
}

func (s *MarginServiceImpl) GetAllMargins() []model.Margin {
	// FIX: Use store.Items() to get all entries
	items := s.store.Items()
	margins := make([]model.Margin, 0, len(items))

	for _, item := range items {
		// Type assertion from interface{} to model.Margin
		margins = append(margins, item.Object.(model.Margin))
	}
	return margins
}

func (s *MarginServiceImpl) GetMargin(symbol string) (*model.Margin, bool) {
	// FIX: Use store.Get() - thread safety is handled internally
	val, exists := s.store.Get(symbol)
	if !exists {
		return nil, false
	}

	margin := val.(model.Margin)
	return &margin, true
}

func (s *MarginServiceImpl) ReloadAllMargins(ctx context.Context) error {
	margins, err := s.repo.FindAll(ctx)
	if err != nil {
		return err
	}

	// FIX: Flush old data and set new data
	s.store.Flush()
	for _, m := range margins {
		s.store.Set(m.Symbol, m, cache.NoExpiration)
	}
	return nil
}

func (s *MarginServiceImpl) LoadFromCsv(ctx context.Context, fileName string, file io.Reader) error {
	if file == nil {
		return fmt.Errorf("file is empty")
	}
	if filepath.Ext(fileName) != ".csv" {
		return fmt.Errorf("invalid file type: must be .csv")
	}

	margins, err := util.Read(file, s.leverage)
	if err != nil {
		return fmt.Errorf("csv parsing failed: %w", err)
	}

	if err := s.repo.SaveAll(ctx, margins); err != nil {
		return fmt.Errorf("failed to save margins: %w", err)
	}

	var ids []string
	for _, m := range margins {
		ids = append(ids, m.Symbol)
	}

	deletedCount, err := s.repo.DeleteByIdNotIn(ctx, ids)
	if err != nil {
		log.Printf("Error deleting old margins: %v", err)
	}

	// FIX: Update local cache using the same Flush/Set pattern
	s.store.Flush()
	for _, m := range margins {
		s.store.Set(m.Symbol, m, cache.NoExpiration)
	}

	log.Printf("CSV Loaded. Cache updated. Deleted %d old records.", deletedCount)
	return nil
}

func (s *MarginServiceImpl) GetStore() *cache.Cache {
	return s.store
}
