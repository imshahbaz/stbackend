package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"

	"backend/cache"
	"backend/config"
	"backend/model"
	"backend/repository"
	"backend/util"
)

type MarginService interface {
	GetAllMargins() []model.Margin
	GetMargin(symbol string) (*model.Margin, bool)
	ReloadAllMargins(ctx context.Context) error
	LoadFromCsv(ctx context.Context, fileName string, file io.Reader) error
}

type MarginServiceImpl struct {
	repo *repository.MarginRepository
	cfg  *config.ConfigManager
}

func NewMarginService(repo *repository.MarginRepository, cfg *config.ConfigManager) MarginService {
	s := &MarginServiceImpl{
		repo: repo,
		cfg:  cfg,
	}

	if err := s.ReloadAllMargins(context.Background()); err != nil {
		log.Printf("Warning: Failed initial margin load: %v", err)
	}

	return s
}

func (s *MarginServiceImpl) GetAllMargins() []model.Margin {
	items := cache.MarginCache.Items()
	margins := make([]model.Margin, 0, len(items))

	for _, item := range items {
		if m, ok := item.Object.(model.Margin); ok {
			margins = append(margins, m)
		}
	}
	return margins
}

func (s *MarginServiceImpl) GetMargin(symbol string) (*model.Margin, bool) {
	val, exists := cache.MarginCache.Get(symbol)
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

	s.updateLocalCache(margins)
	return nil
}

func (s *MarginServiceImpl) LoadFromCsv(ctx context.Context, fileName string, file io.Reader) error {
	if file == nil {
		return fmt.Errorf("file is empty")
	}
	if filepath.Ext(fileName) != ".csv" {
		return fmt.Errorf("invalid file type: must be .csv")
	}

	margins, err := util.Read(file, s.cfg.GetConfig().Leverage)
	if err != nil {
		return fmt.Errorf("csv parsing failed: %w", err)
	}

	if err := s.repo.SaveAll(ctx, margins); err != nil {
		return fmt.Errorf("failed to save margins: %w", err)
	}

	ids := make([]string, len(margins))
	for i, m := range margins {
		ids[i] = m.Symbol
	}

	deletedCount, err := s.repo.DeleteByIdNotIn(ctx, ids)
	if err != nil {
		log.Printf("Error deleting old margins: %v", err)
	}

	s.updateLocalCache(margins)

	log.Printf("CSV Loaded. Cache updated. Symbols synced: %d. Deleted stale: %d", len(margins), deletedCount)
	return nil
}


func (s *MarginServiceImpl) updateLocalCache(margins []model.Margin) {
	cache.MarginCache.Flush()
	for _, m := range margins {
		cache.MarginCache.Set(m.Symbol, m, -1)
	}
}
