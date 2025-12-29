package service

import (
	"context"
	"errors"
	"io"
	"log"
	"sort"
	"time"

	"backend/cache"
	"backend/model"
	"backend/repository"
	"backend/util"

	"github.com/jinzhu/copier"
)

type PriceActionService interface {
	GetPABySymbol(ctx context.Context, symbol string) (model.StockRecord, error)
	SaveOrderBlock(ctx context.Context, req model.ObRequest) error
	UpdateOrderBlock(ctx context.Context, req model.ObRequest) error
	DeleteOrderBlock(ctx context.Context, symbol string, date string) error
	CheckOBMitigation(ctx context.Context) ([]model.ObResponse, error)
	AutomateOrderBlock(ctx context.Context) error

	SaveFvg(ctx context.Context, req model.ObRequest) error
	UpdateFvg(ctx context.Context, req model.ObRequest) error
	DeleteFvg(ctx context.Context, symbol string, date string) error
	CheckFvgMitigation(ctx context.Context) ([]model.ObResponse, error)
	AutomateFvg(ctx context.Context) error

	AddOlderOb(ctx context.Context, fileName string, file io.Reader, stopDate string)
	AddOlderFvg(ctx context.Context, fileName string, file io.Reader, stopDate string)
}

type PriceActionServiceImpl struct {
	chartInkService ChartInkService
	nseService      NseService
	priceActionRepo *repository.PriceActionRepo
	marginSvc       MarginService
}

func NewPriceActionService(c ChartInkService, n NseService,
	repo *repository.PriceActionRepo, marginSvc MarginService) PriceActionService {
	return &PriceActionServiceImpl{
		chartInkService: c,
		nseService:      n,
		priceActionRepo: repo,
		marginSvc:       marginSvc,
	}
}

// --- Internal Engine ---

func (s *PriceActionServiceImpl) processMitigation(ctx context.Context, strategyName string, cacheKey string, isOB bool) ([]model.ObResponse, error) {
	rawStrategy, found := cache.StrategyCache.Get(strategyName)
	if !found {
		return nil, errors.New("strategy not found in cache: " + strategyName)
	}
	strategy := rawStrategy.(model.StrategyDto)

	data, err := s.chartInkService.FetchWithMargin(strategy)
	if err != nil {
		return nil, err
	}

	idMap := make(map[string]model.StockMarginDto)
	ids := make([]string, 0, len(data))
	for _, dto := range data {
		idMap[dto.Symbol] = dto
		ids = append(ids, dto.Symbol)
	}

	pas, err := s.priceActionRepo.GetAllPAIn(ctx, ids)
	if err != nil {
		return nil, err
	}

	var response []model.ObResponse
	for _, pa := range pas {
		history, err := s.nseService.FetchStockData(pa.Symbol)
		if err != nil || len(history) == 0 {
			continue
		}
		today := history[0]

		blocks := pa.OrderBlocks
		if !isOB {
			blocks = pa.Fvg
		}

		for _, block := range blocks {
			// Mitigation Logic: Price pierces the zone but stays within bounds
			if (today.Low < block.High || today.Low < block.Low) && today.Close > block.High {
				var obResp model.ObResponse
				copier.Copy(&obResp, idMap[pa.Symbol])
				obResp.Date = block.Date
				response = append(response, obResp)
				break
			}
		}
	}

	if len(response) > 0 {
		sort.Slice(response, func(i, j int) bool {
			return response[i].Margin > response[j].Margin
		})
		cache.PriceActionCache.Set(cacheKey, response, 1*time.Hour)
	}
	return response, nil
}

// --- Interface Methods ---

func (s *PriceActionServiceImpl) AutomateOrderBlock(ctx context.Context) error {
	raw, _ := cache.StrategyCache.Get("BULLISH OB 1D")
	strategy, ok := raw.(model.StrategyDto)
	if !ok {
		return errors.New("OB strategy not in cache")
	}

	data, _ := s.chartInkService.FetchWithMargin(strategy)
	count := 0
	for _, dto := range data {
		if history, err := s.nseService.FetchStockData(dto.Symbol); err == nil && len(history) >= 3 {
			candle := history[2]
			if date, err := util.ParseNseDate(candle.Timestamp); err == nil {
				_ = s.priceActionRepo.SaveOrderBlock(ctx, model.ObRequest{
					Symbol: dto.Symbol, Date: date, High: candle.High, Low: candle.Low,
				})
				count++
			}
		}
	}
	log.Printf("%d Order block's inserted", count)
	return nil
}

func (s *PriceActionServiceImpl) AutomateFvg(ctx context.Context) error {
	raw, _ := cache.StrategyCache.Get("FAIR VALUE GAP")
	strategy, ok := raw.(model.StrategyDto)
	if !ok {
		return errors.New("FVG strategy not in cache")
	}

	data, _ := s.chartInkService.FetchWithMargin(strategy)
	count := 0
	for _, dto := range data {
		if history, err := s.nseService.FetchStockData(dto.Symbol); err == nil && len(history) >= 3 {
			if date, err := util.ParseNseDate(history[1].Timestamp); err == nil {
				_ = s.priceActionRepo.SaveFvg(ctx, model.ObRequest{
					Symbol: dto.Symbol, Date: date, High: history[0].Low, Low: history[2].High,
				})
				count++
			}
		}
	}
	log.Printf("%d Fvg's inserted", count)
	return nil
}

func (s *PriceActionServiceImpl) GetPABySymbol(ctx context.Context, symbol string) (model.StockRecord, error) {
	return s.priceActionRepo.GetPAByID(ctx, symbol)
}

func (s *PriceActionServiceImpl) CheckOBMitigation(ctx context.Context) ([]model.ObResponse, error) {
	return s.processMitigation(ctx, "BULLISH CLOSE 200", "ObCache", true)
}

func (s *PriceActionServiceImpl) CheckFvgMitigation(ctx context.Context) ([]model.ObResponse, error) {
	return s.processMitigation(ctx, "BULLISH CLOSE 200", "FvgCache", false)
}

// Pass-through CRUD methods
func (s *PriceActionServiceImpl) SaveOrderBlock(ctx context.Context, req model.ObRequest) error {
	return s.priceActionRepo.SaveOrderBlock(ctx, req)
}

func (s *PriceActionServiceImpl) UpdateOrderBlock(ctx context.Context, req model.ObRequest) error {
	return s.priceActionRepo.UpdateOrderBlock(ctx, req)
}

func (s *PriceActionServiceImpl) DeleteOrderBlock(ctx context.Context, sym string, d string) error {
	return s.priceActionRepo.DeleteOrderBlockByDate(ctx, sym, d)
}

func (s *PriceActionServiceImpl) SaveFvg(ctx context.Context, req model.ObRequest) error {
	return s.priceActionRepo.SaveFvg(ctx, req)
}

func (s *PriceActionServiceImpl) UpdateFvg(ctx context.Context, req model.ObRequest) error {
	return s.priceActionRepo.UpdateFvg(ctx, req)
}

func (s *PriceActionServiceImpl) DeleteFvg(ctx context.Context, sym string, d string) error {
	return s.priceActionRepo.DeleteFvgByDate(ctx, sym, d)
}

// Shared logic to find the index and data for a specific date
func (s *PriceActionServiceImpl) processHistory(stock string, date string) (string, []model.NSEHistoricalData, int, bool) {
	m, exists := s.marginSvc.GetMargin(stock)
	if !exists {
		return "", nil, 0, false
	}

	// Uses your time cache strategy internally
	history, err := s.nseService.FetchStockData(m.Symbol)
	if err != nil || len(history) < 3 {
		return "", nil, 0, false
	}

	for i := 0; i <= len(history)-3; i++ {
		candleDate, err := util.ParseNseDate(history[i].Timestamp)
		if err == nil && candleDate == date {
			return m.Symbol, history, i, true
		}
	}
	return "", nil, 0, false
}

func (s *PriceActionServiceImpl) AddOlderOb(ctx context.Context, fileName string, file io.Reader, stopDate string) {
	req, err := util.ReadCSVReversed(file, stopDate)
	if err != nil {
		return
	}

	count := 0
	for _, stock := range req {
		if symbol, history, i, found := s.processHistory(stock.Symbol, stock.Date); found {
			target := history[i+2]
			if actualDate, err := util.ParseNseDate(target.Timestamp); err == nil {
				_ = s.priceActionRepo.SaveOrderBlock(ctx, model.ObRequest{
					Symbol: symbol,
					Date:   actualDate,
					High:   target.High,
					Low:    target.Low,
				})
				count++
			}
		}
	}
	log.Printf("%d Order block's inserted", count)
}

func (s *PriceActionServiceImpl) AddOlderFvg(ctx context.Context, fileName string, file io.Reader, stopDate string) {
	req, err := util.ReadCSVReversed(file, stopDate)
	if err != nil {
		return
	}

	count := 0
	for _, stock := range req {
		if symbol, history, i, found := s.processHistory(stock.Symbol, stock.Date); found {
			if actualDate, err := util.ParseNseDate(history[i+1].Timestamp); err == nil {
				_ = s.priceActionRepo.SaveFvg(ctx, model.ObRequest{
					Symbol: symbol,
					Date:   actualDate,
					High:   history[i].Low,
					Low:    history[i+2].High,
				})
				count++
			}
		}
	}
	log.Printf("%d Fvg's inserted", count)
}
