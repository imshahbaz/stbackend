package service

import (
	"context"
	"errors"
	"io"
	"log"
	"sort"
	"sync"
	"sync/atomic"
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
	AutomateOrderBlock(ctx context.Context, attempt int) error

	SaveFvg(ctx context.Context, req model.ObRequest) error
	UpdateFvg(ctx context.Context, req model.ObRequest) error
	DeleteFvg(ctx context.Context, symbol string, date string) error
	CheckFvgMitigation(ctx context.Context) ([]model.ObResponse, error)
	AutomateFvg(ctx context.Context, attempt int) error
	PACleanUp(ctx context.Context) error

	addOlderOb(ctx context.Context, i int, history []model.NSEHistoricalData, count *int)
	AddOlderFvgAndOb(ctx context.Context, fileName string, file io.Reader, stopDate string)
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
		history, err := s.nseService.FetchStockData(ctx, pa.Symbol)
		if err != nil || len(history) == 0 {
			continue
		}
		today := history[0]

		blocks := pa.OrderBlocks
		if !isOB {
			blocks = pa.Fvg
		}

		for _, block := range blocks {
			checkDate, _ := util.ParseNseDate(history[1].Timestamp)
			if !isOB && block.Date == checkDate {
				continue
			}
			if s.checkValidMitigation(today, block) {
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
		cache.SetPriceActionResponseCache(cacheKey, response)
	}
	return response, nil
}

func (s *PriceActionServiceImpl) AutomateOrderBlock(ctx context.Context, attempt int) error {
	if attempt >= 3 {
		return nil
	}

	raw, _ := cache.StrategyCache.Get("FAIR VALUE GAP")
	strategy, ok := raw.(model.StrategyDto)

	if !ok {
		return errors.New("OB strategy not in cache")
	}

	data, _ := s.chartInkService.FetchWithMargin(strategy)
	count := 0
	for _, dto := range data {
		s.nseService.ClearStockDataCache(dto.Symbol)
		if history, err := s.nseService.FetchStockData(ctx, dto.Symbol); err == nil && len(history) >= 3 {
			if s.automationReschedule(history[0]) {
				log.Printf("Rescheduling Ob automation for %d time", attempt+1)
				time.AfterFunc(25*time.Minute, func() {
					s.AutomateOrderBlock(context.Background(), attempt+1)
				})
				return nil
			}

			for i := 2; i < len(history); i++ {
				candle := history[i]
				if candle.Close < candle.Open {
					if date, err := util.ParseNseDate(candle.Timestamp); err == nil {
						_ = s.priceActionRepo.SaveOrderBlock(ctx, model.ObRequest{
							Symbol: dto.Symbol, Date: date, High: candle.High, Low: candle.Low,
						})
						count++
					}
					break
				}
			}

		}
	}

	if count > 0 {
		cache.DeletePriceActionResponseCache("ObCache")
	}
	log.Printf("%d Order block's inserted", count)
	return nil
}

func (s *PriceActionServiceImpl) AutomateFvg(ctx context.Context, attempt int) error {
	if attempt >= 3 {
		return nil
	}
	raw, _ := cache.StrategyCache.Get("FAIR VALUE GAP")
	strategy, ok := raw.(model.StrategyDto)
	if !ok {
		return errors.New("FVG strategy not in cache")
	}

	data, _ := s.chartInkService.FetchWithMargin(strategy)
	count := 0
	for _, dto := range data {
		s.nseService.ClearStockDataCache(dto.Symbol)
		if history, err := s.nseService.FetchStockData(ctx, dto.Symbol); err == nil && len(history) >= 3 {
			if s.automationReschedule(history[0]) {
				log.Printf("Rescheduling Fvg automation for %d time", attempt+1)
				time.AfterFunc(30*time.Minute, func() {
					s.AutomateFvg(context.Background(), attempt+1)
				})
				return nil
			}
			if date, err := util.ParseNseDate(history[1].Timestamp); err == nil {
				_ = s.priceActionRepo.SaveFvg(ctx, model.ObRequest{
					Symbol: dto.Symbol, Date: date, High: history[0].Low, Low: history[2].High,
				})
				count++
			}
		}
	}
	if count > 0 {
		cache.DeletePriceActionResponseCache("FvgCache")
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

func (s *PriceActionServiceImpl) SaveOrderBlock(ctx context.Context, req model.ObRequest) error {
	time.AfterFunc(3*time.Second, func() {
		cache.DeletePriceActionResponseCache("ObCache")
	})
	return s.priceActionRepo.SaveOrderBlock(ctx, req)
}

func (s *PriceActionServiceImpl) UpdateOrderBlock(ctx context.Context, req model.ObRequest) error {
	time.AfterFunc(3*time.Second, func() {
		cache.DeletePriceActionResponseCache("ObCache")
	})
	return s.priceActionRepo.UpdateOrderBlock(ctx, req)
}

func (s *PriceActionServiceImpl) DeleteOrderBlock(ctx context.Context, sym string, d string) error {
	time.AfterFunc(3*time.Second, func() {
		cache.DeletePriceActionResponseCache("ObCache")
	})
	return s.priceActionRepo.DeleteOrderBlockByDate(ctx, sym, d)
}

func (s *PriceActionServiceImpl) SaveFvg(ctx context.Context, req model.ObRequest) error {
	time.AfterFunc(3*time.Second, func() {
		cache.DeletePriceActionResponseCache("FvgCache")
	})
	return s.priceActionRepo.SaveFvg(ctx, req)
}

func (s *PriceActionServiceImpl) UpdateFvg(ctx context.Context, req model.ObRequest) error {
	time.AfterFunc(3*time.Second, func() {
		cache.DeletePriceActionResponseCache("FvgCache")
	})
	return s.priceActionRepo.UpdateFvg(ctx, req)
}

func (s *PriceActionServiceImpl) DeleteFvg(ctx context.Context, sym string, d string) error {
	time.AfterFunc(3*time.Second, func() {
		cache.DeletePriceActionResponseCache("FvgCache")
	})
	return s.priceActionRepo.DeleteFvgByDate(ctx, sym, d)
}

func (s *PriceActionServiceImpl) processHistory(ctx context.Context, stock string, date string) (string, []model.NSEHistoricalData, int, bool) {
	m, exists := s.marginSvc.GetMargin(stock)
	if !exists {
		return "", nil, 0, false
	}

	history, err := s.nseService.FetchStockData(ctx, m.Symbol)
	if err != nil || len(history) < 3 {
		return "", nil, 0, false
	}

	for i := 0; i <= len(history)-2; i++ {
		candleDate, err := util.ParseNseDate(history[i].Timestamp)
		if err == nil && candleDate == date {
			return m.Symbol, history, i, true
		}
	}
	return "", nil, 0, false
}

func (s *PriceActionServiceImpl) addOlderOb(ctx context.Context, i int, history []model.NSEHistoricalData, count *int) {
	for k := i; k < len(history); k++ {
		candle := history[k]
		if candle.Close < candle.Open {
			actualDate, _ := util.ParseNseDate(candle.Timestamp)
			_ = s.priceActionRepo.SaveOrderBlock(ctx, model.ObRequest{
				Symbol: candle.Symbol,
				Date:   actualDate,
				High:   candle.High,
				Low:    candle.Low,
			})
			*count++
			break
		}
	}
}

func (s *PriceActionServiceImpl) AddOlderFvgAndOb(ctx context.Context, fileName string, file io.Reader, stopDate string) {
	req, err := util.ReadCSVReversed(file, stopDate)
	if err != nil {
		return
	}

	count := 0
	obCount := 0
	for _, stock := range req {
		if symbol, history, i, found := s.processHistory(ctx, stock.Symbol, stock.Date); found {
			if actualDate, err := util.ParseNseDate(history[i+1].Timestamp); err == nil {
				_ = s.priceActionRepo.SaveFvg(ctx, model.ObRequest{
					Symbol: symbol,
					Date:   actualDate,
					High:   history[i].Low,
					Low:    history[i+2].High,
				})
				count++
			}
			s.addOlderOb(ctx, i+1, history, &obCount)
		}
	}
	if count > 0 {
		cache.DeletePriceActionResponseCache("FvgCache")
	}
	log.Printf("%d Fvg's inserted", count)
	log.Printf("%d Order block's inserted", obCount)
}

func (s *PriceActionServiceImpl) checkValidMitigation(candle model.NSEHistoricalData, info model.Info) bool {
	if candle.Close < info.Low || candle.Low < info.Low || candle.Low > info.High {
		return false
	}
	return true
}

func (s *PriceActionServiceImpl) automationReschedule(candle model.NSEHistoricalData) bool {
	now := time.Now().In(util.IstLocation)
	day := now.Weekday()
	if day == 0 || day == 6 {
		return false
	}

	today := now.Format(util.OutputLayout)
	candleDate, _ := util.ParseNseDate(candle.Timestamp)
	return candleDate < today
}

func (s *PriceActionServiceImpl) PACleanUp(ctx context.Context) error {
	data, err := s.priceActionRepo.GetAllPriceAction(ctx)
	if err != nil {
		return err
	}

	var (
		fvgCleaned atomic.Int32
		obCleaned  atomic.Int32
		wg         sync.WaitGroup
		sem        = make(chan struct{}, 1)
	)

	for _, record := range data {
		if len(record.Fvg) == 0 && len(record.OrderBlocks) == 0 {
			continue
		}

		wg.Add(1)
		go func(record model.StockRecord) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			history, err := s.nseService.FetchStockData(ctx, record.Symbol)
			if err != nil {
				return
			}

			// Process FVGs
			for _, info := range record.Fvg {
				if shouldDeleteZone(info.Date, info.Low, info.High, history, false) {
					s.priceActionRepo.DeleteFvgByDate(ctx, record.Symbol, info.Date)
					fvgCleaned.Add(1)
				}
			}

			// Process Order Blocks
			for _, info := range record.OrderBlocks {
				if shouldDeleteZone(info.Date, info.Low, info.High, history, true) {
					s.priceActionRepo.DeleteOrderBlockByDate(ctx, record.Symbol, info.Date)
					obCleaned.Add(1)
				}
			}
		}(record)
	}

	wg.Wait()

	// Cache invalidation based on atomic counts
	if fvgCleaned.Load() > 0 {
		cache.DeletePriceActionResponseCache("FvgCache")
	}
	if obCleaned.Load() > 0 {
		cache.DeletePriceActionResponseCache("ObCache")
	}

	log.Printf("Cleanup complete. FVG: %d, OB: %d", fvgCleaned.Load(), obCleaned.Load())
	return nil
}

// Helper to encapsulate the price action violation logic
func shouldDeleteZone(zoneDate string, low, high float64, history []model.NSEHistoricalData, isOb bool) bool {
	count := 0
	lastIndex := -1
	for i, candle := range history {
		candleDate, _ := util.ParseNseDate(candle.Timestamp)

		// Only look at candles appearing AFTER the zone was created
		if zoneDate >= candleDate {
			if isOb && lastIndex >= 0 && (lastIndex+1 == i) {
				count--
			}
			break
		}

		// Rule 1: Price closed below or dipped below the zone floor
		if candle.Low < low {
			return true
		}

		// Rule 2: Mitigation check (Price entered the zone and bounced)
		if candle.Close > candle.Open && candle.Low < high {
			count++
			lastIndex = i
		}
	}

	return count > 1
}
