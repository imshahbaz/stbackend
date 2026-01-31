package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"backend/cache"
	"backend/config"
	"backend/model"
	"backend/repository"
	"backend/util"

	"github.com/bytedance/sonic"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

var (
	optionOutput []model.OptionOutput
)

type MarginService interface {
	GetAllMargins() []model.Margin
	GetMargin(symbol string) (*model.Margin, bool)
	ReloadAllMargins(ctx context.Context) error
	LoadFromCsv(ctx context.Context, fileName string, file io.Reader) error
	SyncMTF(ctx context.Context, file io.Reader) error
	SyncMarginToken(ctx context.Context) error
	ReloadAllOptions(ctx context.Context) error
	GetAllOptions() []model.OptionOutput
}

type MarginServiceImpl struct {
	repo    *repository.MarginRepository
	cfg     *config.ConfigManager
	optRepo *repository.OptionRepository
}

func NewMarginService(repo *repository.MarginRepository, cfg *config.ConfigManager, optRepo *repository.OptionRepository) MarginService {
	return &MarginServiceImpl{
		repo:    repo,
		cfg:     cfg,
		optRepo: optRepo,
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

	client := resty.New().SetTimeout(5 * time.Minute)
	url := "https://margincalculator.angelbroking.com/OpenAPI_File/files/OpenAPIScripMaster.json"

	resp, err := client.R().
		SetContext(ctx).
		SetDoNotParseResponse(true).
		Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch instrument list: %w", err)
	}
	defer resp.RawBody().Close()

	body, err := io.ReadAll(resp.RawBody())
	if err != nil {
		return fmt.Errorf("failed to read full response body: %w", err)
	}

	log.Info().Int("bytes_received", len(body)).Msg("Download complete")

	var allInstruments []model.MinimalInstrument
	if err := sonic.Unmarshal(body, &allInstruments); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	log.Info().Int("count", len(allInstruments)).Msg("Total instruments decoded")

	updatedMargins := make([]model.Margin, 0, len(targetMargins))
	optionMap := make(map[string]*model.OptionChain, 100)
	names := []string{"NIFTY", "BANKNIFTY", "SENSEX"}
	now := util.ToIST(time.Now())
	cutoff := now.AddDate(0, 1, 0)
	for i := range allInstruments {
		inst := &allInstruments[i]

		if slices.Contains(names, inst.Name) && inst.Expiry != "" {

			expiry, err := util.ParseAllCapsDate(inst.Expiry)
			if err != nil {
				continue
			}

			if expiry.After(cutoff) || expiry.Before(now) {
				continue
			}

			strike := util.NormalizeStrike(inst.Strike)
			if strike == "" || strike == "-1" || strike == "0" {
				continue
			}

			option := model.OptionChain{
				Symbol:        inst.Name + inst.Expiry + strike + inst.Symbol[len(inst.Symbol)-2:],
				Expiry:        inst.Expiry,
				Strike:        strike,
				LotSize:       inst.LotSize,
				ExchangeType:  inst.ExchSeg,
				AngelOneToken: inst.Token,
			}

			finalOpt := option
			optionMap[option.Symbol] = &finalOpt
			continue
		}

		if inst.ExchSeg != "NSE" || !strings.HasSuffix(inst.Symbol, "-EQ") {
			continue
		}

		if margin, exists := targetMargins[inst.Name]; exists {
			margin.Token = inst.Token
			updatedMargins = append(updatedMargins, margin)
			delete(targetMargins, inst.Name)
		}
	}

	go func(optionMap map[string]*model.OptionChain, client *resty.Client, names []string) {
		if len(optionMap) == 0 {
			return
		}
		url := "https://api.mstock.trade/openapi/typeb/instruments/OpenAPIScripMaster"
		resp, err := client.R().
			SetContext(context.Background()).
			SetDoNotParseResponse(true).
			Get(url)
		if err != nil {
			log.Error().Err(err).Msg("failed to fetch instrument list")
			return
		}
		defer resp.RawBody().Close()

		var allInstruments []model.MinimalInstrument
		decoder := sonic.ConfigDefault.NewDecoder(resp.RawBody())
		if err := decoder.Decode(&allInstruments); err != nil {
			log.Error().Err(err).Msg("failed to decode instrument list")
			return
		}

		for i := range allInstruments {
			inst := &allInstruments[i]

			if slices.Contains(names, inst.Symbol) && inst.Strike != "" {
				key := inst.Symbol + strings.ToUpper(inst.Expiry) + inst.Strike + inst.Name[len(inst.Name)-2:]
				if option, exists := optionMap[key]; exists {
					option.MstockSymbol = inst.Name
				}

			}
		}

		ids := make([]any, len(optionMap))
		optionChain := make([]model.OptionChain, 0, len(optionMap))
		for id, option := range optionMap {
			ids = append(ids, id)
			optionChain = append(optionChain, *option)
		}

		if err := s.optRepo.GenericRepo.SaveAll(context.Background(), optionChain, "Symbol"); err != nil {
			log.Error().Err(err).Msg("failed to save option chain")
		}

		deletedCount, err := s.optRepo.GenericRepo.DeleteByIdNotIn(context.Background(), ids)
		if err != nil {
			log.Error().Err(err).Msg("failed to delete option chain")
		}

		s.updateOptionLocalCache(optionChain)
		log.Info().Msgf("Option chain synced. Symbols updated: %d. Deleted stale: %d", len(optionChain), deletedCount)

	}(optionMap, client, names)

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

func (s *MarginServiceImpl) updateOptionLocalCache(optionChain []model.OptionChain) {
	cache.OptionCache.Flush()
	for _, m := range optionChain {
		cache.OptionCache.Set(m.Symbol, m, -1)
	}
	optionOutput = nil
}

func (s *MarginServiceImpl) ReloadAllOptions(ctx context.Context) error {
	optionChain, err := s.optRepo.GenericRepo.GetAll(ctx, nil)
	if err != nil {
		return err
	}

	s.updateOptionLocalCache(optionChain)
	return nil
}

func (s *MarginServiceImpl) GetAllOptions() []model.OptionOutput {
	if optionOutput != nil {
		return optionOutput
	}

	items := cache.OptionCache.Items()

	niftyExpSet := make(map[string]struct{})
	bnExpSet := make(map[string]struct{})
	sensexExpSet := make(map[string]struct{})

	niftyStrkSet := make(map[string]struct{})
	bnStrkSet := make(map[string]struct{})
	sensexStrkSet := make(map[string]struct{})

	for _, item := range items {
		if option, ok := item.Object.(model.OptionChain); ok {
			symbol := option.Symbol

			if strings.HasPrefix(symbol, "NIFTY") {
				niftyExpSet[option.Expiry] = struct{}{}
				niftyStrkSet[option.Strike] = struct{}{}
			} else if strings.HasPrefix(symbol, "BANKNIFTY") {
				bnExpSet[option.Expiry] = struct{}{}
				bnStrkSet[option.Strike] = struct{}{}
			} else if strings.HasPrefix(symbol, "SENSEX") {
				sensexExpSet[option.Expiry] = struct{}{}
				sensexStrkSet[option.Strike] = struct{}{}
			}
		}
	}

	setToSortedSlice := func(m map[string]struct{}) []string {
		res := make([]string, 0, len(m))
		for k := range m {
			res = append(res, k)
		}
		slices.Sort(res)
		return res
	}

	result := []model.OptionOutput{
		{
			Name:   "NIFTY",
			Expiry: setToSortedSlice(niftyExpSet),
			Strike: setToSortedSlice(niftyStrkSet),
		},
		{
			Name:   "BANKNIFTY",
			Expiry: setToSortedSlice(bnExpSet),
			Strike: setToSortedSlice(bnStrkSet),
		},
		{
			Name:   "SENSEX",
			Expiry: setToSortedSlice(sensexExpSet),
			Strike: setToSortedSlice(sensexStrkSet),
		},
	}

	optionOutput = result
	return result
}
