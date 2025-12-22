package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"sync"

	"backend/model"      // Replace with your actual model package
	"backend/repository" // Replace with your actual repo package
	"backend/util"       // Replace with your actual util package
)

// 1. Interface Definition
type MarginService interface {
	GetAllMargins() []model.Margin
	GetMargin(symbol string) (*model.Margin, bool)
	ReloadAllMargins(ctx context.Context) error
	LoadFromCsv(ctx context.Context, fileName string, file io.Reader) error
}

// 2. Implementation Struct
type MarginServiceImpl struct {
	repo     *repository.MarginRepository
	leverage float32
	// marginMap is made thread-safe using a Read-Write Mutex
	marginMap map[string]model.Margin
	mu        sync.RWMutex
}

// NewMarginService acts as the @RequiredArgsConstructor + @PostConstruct
func NewMarginService(repo *repository.MarginRepository, leverage float32) MarginService {
	s := &MarginServiceImpl{
		repo:      repo,
		leverage:  leverage,
		marginMap: make(map[string]model.Margin),
	}

	// Trigger initial load (PostConstruct equivalent)
	ctx := context.Background()
	if err := s.ReloadAllMargins(ctx); err != nil {
		log.Printf("Warning: Failed initial margin load: %v", err)
	}

	return s
}

func (s *MarginServiceImpl) GetAllMargins() []model.Margin {
	s.mu.RLock()
	defer s.mu.RUnlock()

	margins := make([]model.Margin, 0, len(s.marginMap))
	for _, m := range s.marginMap {
		margins = append(margins, m)
	}
	return margins
}

func (s *MarginServiceImpl) GetMargin(symbol string) (*model.Margin, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	margin, exists := s.marginMap[symbol]
	if !exists {
		return nil, false
	}
	return &margin, true
}

func (s *MarginServiceImpl) ReloadAllMargins(ctx context.Context) error {
	margins, err := s.repo.FindAll(ctx)
	if err != nil {
		return err
	}

	newMap := make(map[string]model.Margin)
	for _, m := range margins {
		newMap[m.Symbol] = m
	}

	s.mu.Lock()
	s.marginMap = newMap
	s.mu.Unlock()
	return nil
}

func (s *MarginServiceImpl) LoadFromCsv(ctx context.Context, fileName string, file io.Reader) error {
	// 1. Validation
	if file == nil {
		return fmt.Errorf("file is empty")
	}
	if filepath.Ext(fileName) != ".csv" {
		return fmt.Errorf("invalid file type: must be .csv")
	}

	// 2. Read using our CsvReader utility
	margins, err := util.Read(file, s.leverage)
	if err != nil {
		return fmt.Errorf("csv parsing failed: %w", err)
	}

	// 3. Save All to DB
	if err := s.repo.SaveAll(ctx, margins); err != nil {
		return fmt.Errorf("failed to save margins: %w", err)
	}

	// 4. Delete old records not in the new list (Sync logic)
	var ids []string
	for _, m := range margins {
		ids = append(ids, m.Symbol)
	}
	deletedCount, err := s.repo.DeleteByIdNotIn(ctx, ids)
	if err != nil {
		log.Printf("Error deleting old margins: %v", err)
	} else {
		log.Printf("Deleted %d old margin(s)", deletedCount)
	}

	// 5. Update local cache
	newMap := make(map[string]model.Margin)
	for _, m := range margins {
		newMap[m.Symbol] = m
	}

	s.mu.Lock()
	s.marginMap = newMap
	s.mu.Unlock()

	return nil
}
