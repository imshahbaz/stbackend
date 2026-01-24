package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"backend/cache"
	"backend/config"
	"backend/model"
	"backend/repository"
	"backend/util"

	"github.com/bytedance/sonic"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

type MarginService interface {
	GetAllMargins() []model.Margin
	GetMargin(symbol string) (*model.Margin, bool)
	ReloadAllMargins(ctx context.Context) error
	LoadFromCsv(ctx context.Context, fileName string, file io.Reader) error
	SyncMTF(ctx context.Context, file io.Reader) error
	SyncMarginToken(ctx context.Context) error
}

type MarginServiceImpl struct {
	repo *repository.MarginRepository
	cfg  *config.ConfigManager
}

func NewMarginService(repo *repository.MarginRepository, cfg *config.ConfigManager) MarginService {
	return &MarginServiceImpl{
		repo: repo,
		cfg:  cfg,
	}
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
	margins, err := s.repo.GenericRepo.GetAll(ctx, nil)
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

	if err := s.syncMargins(ctx, margins, "CSV"); err != nil {
		return err
	}

	return nil
}

func (s *MarginServiceImpl) updateLocalCache(margins []model.Margin) {
	cache.MarginCache.Flush()
	for _, m := range margins {
		cache.MarginCache.Set(m.Symbol, m, -1)
	}
}

func (s *MarginServiceImpl) SyncMTF(ctx context.Context, file io.Reader) error {
	leverage := s.cfg.GetConfig().Leverage

	var rawMargins []struct {
		Symbol   string  `json:"tradingsymbol"`
		Leverage float32 `json:"leverage"`
	}

	decoder := sonic.ConfigDefault.NewDecoder(file)
	if err := decoder.Decode(&rawMargins); err != nil {
		return fmt.Errorf("failed to decode MTF JSON: %w", err)
	}

	var margins []model.Margin
	for _, m := range rawMargins {
		if m.Leverage >= leverage {
			margins = append(margins, model.Margin{
				Symbol: m.Symbol,
				Name:   m.Symbol,
				Margin: float32(m.Leverage),
			})
		}
	}

	if err := s.syncMargins(ctx, margins, "MTF JSON"); err != nil {
		return err
	}

	return nil
}

func (s *MarginServiceImpl) syncMargins(ctx context.Context, margins []model.Margin, source string) error {
	if len(margins) == 0 {
		return nil
	}

	if err := s.repo.GenericRepo.SaveAll(ctx, margins, "Symbol"); err != nil {
		return fmt.Errorf("failed to save margins: %w", err)
	}

	ids := make([]any, len(margins))
	for i, m := range margins {
		ids[i] = m.Symbol
	}

	deletedCount, err := s.repo.GenericRepo.DeleteByIdNotIn(ctx, ids)
	if err != nil {
		log.Error().Err(err).Msgf("Error deleting old margins from %s", source)
	}

	s.updateLocalCache(margins)

	log.Info().Msgf("%s Loaded. Cache updated. Symbols synced: %d. Deleted stale: %d", source, len(margins), deletedCount)
	return nil
}

func (s *MarginServiceImpl) SyncMarginToken(ctx context.Context) error {
	type MinimalInstrument struct {
		Name    string `json:"name"`
		Symbol  string `json:"symbol"`
		Token   string `json:"token"`
		ExchSeg string `json:"exch_seg"`
	}

	marginItems := cache.MarginCache.Items()
	if len(marginItems) == 0 {
		return nil
	}

	targetMargins := make(map[string]model.Margin, len(marginItems))
	for symbol, item := range marginItems {
		if m, ok := item.Object.(model.Margin); ok {
			targetMargins[symbol] = m
		}
	}

	client := resty.New()
	url := "https://margincalculator.angelbroking.com/OpenAPI_File/files/OpenAPIScripMaster.json"

	resp, err := client.R().
		SetContext(ctx).
		SetDoNotParseResponse(true).
		Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch instrument list: %w", err)
	}
	defer resp.RawBody().Close()

	var allInstruments []MinimalInstrument
	decoder := sonic.ConfigDefault.NewDecoder(resp.RawBody())
	if err := decoder.Decode(&allInstruments); err != nil {
		return fmt.Errorf("failed to decode instrument list: %w", err)
	}

	updatedMargins := make([]model.Margin, 0, len(targetMargins))
	for i := range allInstruments {
		inst := &allInstruments[i]

		if inst.ExchSeg != "NSE" || !strings.HasSuffix(inst.Symbol, "-EQ") {
			continue
		}

		if margin, exists := targetMargins[inst.Name]; exists {
			margin.Token = inst.Token
			updatedMargins = append(updatedMargins, margin)
			delete(targetMargins, inst.Name)
		}

		if len(targetMargins) == 0 {
			break
		}
	}

	if len(updatedMargins) == 0 {
		log.Info().Msg("No matching margin tokens found to update")
		return nil
	}

	if err := s.repo.GenericRepo.SaveAll(ctx, updatedMargins, "Symbol"); err != nil {
		return fmt.Errorf("failed to save updated margins: %w", err)
	}

	s.updateLocalCache(updatedMargins)

	log.Info().Msgf("Margin tokens synced. Symbols updated: %d", len(updatedMargins))
	return nil
}
