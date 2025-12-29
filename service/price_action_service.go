package service

import (
	"backend/cache"
	"backend/model"
	"backend/repository"
	"backend/util"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
)

type PriceActionService interface {
	GetPABySymbol(ctx *gin.Context)

	// Order Block (OB) Operations
	SaveOrderBlock(ctx *gin.Context)
	DeleteOrderBlock(ctx *gin.Context)
	CheckOBMitigation(ctx *gin.Context)
	AutomateOrderBlock(ctx *gin.Context)
	UpdateOrderBlock(ctx *gin.Context)

	// Fair Value Gap (FVG) Operations
	SaveFvg(ctx *gin.Context)
	DeleteFvg(ctx *gin.Context)
	CheckFvgMitigation(ctx *gin.Context)
	AutomateFvg(ctx *gin.Context)
	UpdateFvg(ctx *gin.Context)
}

type PriceActionServiceImpl struct {
	chartInkService ChartInkService
	nseService      NseService
	priceActionRepo *repository.PriceActionRepo
}

func NewPriceActionService(c ChartInkService, n NseService, repo *repository.PriceActionRepo) PriceActionService {
	return &PriceActionServiceImpl{
		chartInkService: c,
		nseService:      n,
		priceActionRepo: repo,
	}
}

// --- Helper Methods ---

func (s *PriceActionServiceImpl) sendError(ctx *gin.Context, code int, msg string) {
	ctx.JSON(code, model.Response{Success: false, Error: msg})
}

func (s *PriceActionServiceImpl) sendSuccess(ctx *gin.Context, msg string, data interface{}) {
	ctx.JSON(http.StatusOK, model.Response{Success: true, Message: msg, Data: data})
}

func (s *PriceActionServiceImpl) bindObRequest(ctx *gin.Context) (model.ObRequest, bool) {
	var req model.ObRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		s.sendError(ctx, http.StatusBadRequest, err.Error())
		return req, false
	}
	return req, true
}

func (s *PriceActionServiceImpl) processMitigation(ctx *gin.Context, strategyName string, cacheKey string, isOB bool) {
	rawStrategy, ok := cache.StrategyCache.Get(strategyName)
	strategy, ok := rawStrategy.(model.StrategyDto)
	if !ok {
		s.sendError(ctx, http.StatusInternalServerError, strategyName+" error")
		return
	}

	data, err := s.chartInkService.FetchWithMargin(strategy)
	if err != nil {
		s.sendError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	idMap := make(map[string]model.StockMarginDto)
	ids := make([]string, 0, len(data))
	for _, dto := range data {
		idMap[dto.Symbol] = dto
		ids = append(ids, dto.Symbol)
	}

	pas, err := s.priceActionRepo.GetAllPAIn(ctx, ids)
	if err != nil {
		s.sendError(ctx, http.StatusInternalServerError, err.Error())
		return
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
		cache.PriceActionCache.Set(cacheKey, response, -1)
	}
	s.sendSuccess(ctx, strategyName+" fetch success", response)
}

// --- Interface Implementations ---

func (s *PriceActionServiceImpl) GetPABySymbol(ctx *gin.Context) {
	symbol := ctx.Param("symbol")
	if symbol == "" {
		s.sendError(ctx, http.StatusBadRequest, "Invalid request")
		return
	}
	data, err := s.priceActionRepo.GetPAByID(ctx, symbol)
	if err != nil {
		s.sendError(ctx, http.StatusNotFound, err.Error())
		return
	}
	s.sendSuccess(ctx, "Price action found", data)
}

// Order Block Methods
func (s *PriceActionServiceImpl) SaveOrderBlock(ctx *gin.Context) {
	if req, ok := s.bindObRequest(ctx); ok {
		if err := s.priceActionRepo.SaveOrderBlock(ctx, req); err != nil {
			s.sendError(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		s.sendSuccess(ctx, "Order block created", nil)
	}
}

func (s *PriceActionServiceImpl) UpdateOrderBlock(ctx *gin.Context) {
	if req, ok := s.bindObRequest(ctx); ok {
		if err := s.priceActionRepo.UpdateOrderBlock(ctx, req); err != nil {
			s.sendError(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		s.sendSuccess(ctx, "Order block updated", nil)
	}
}

func (s *PriceActionServiceImpl) DeleteOrderBlock(ctx *gin.Context) {
	if req, ok := s.bindObRequest(ctx); ok {
		if err := s.priceActionRepo.DeleteOrderBlockByDate(ctx, req.Symbol, req.Date); err != nil {
			s.sendError(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		s.sendSuccess(ctx, "Order block deleted", nil)
	}
}

func (s *PriceActionServiceImpl) CheckOBMitigation(ctx *gin.Context) {
	s.processMitigation(ctx, "BULLISH CLOSE 200", "ObCache", true)
}

func (s *PriceActionServiceImpl) AutomateOrderBlock(ctx *gin.Context) {
	rawStrategy, _ := cache.StrategyCache.Get("BULLISH OB 1D")
	strategy, ok := rawStrategy.(model.StrategyDto)
	if !ok {
		s.sendError(ctx, http.StatusInternalServerError, "OB strategy error")
		return
	}

	data, _ := s.chartInkService.FetchWithMargin(strategy)
	for _, dto := range data {
		if history, err := s.nseService.FetchStockData(dto.Symbol); err == nil && len(history) >= 3 {
			candle := history[2]
			if date, err := util.ParseNseDate(candle.Timestamp); err == nil {
				s.priceActionRepo.SaveOrderBlock(ctx, model.ObRequest{
					Symbol: dto.Symbol, Date: date, High: candle.High, Low: candle.Low,
				})
			}
		}
	}
	s.sendSuccess(ctx, "Order block automation completed", nil)
}

// FVG Methods
func (s *PriceActionServiceImpl) SaveFvg(ctx *gin.Context) {
	if req, ok := s.bindObRequest(ctx); ok {
		if err := s.priceActionRepo.SaveFvg(ctx, req); err != nil {
			s.sendError(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		s.sendSuccess(ctx, "Fvg created", nil)
	}
}

func (s *PriceActionServiceImpl) UpdateFvg(ctx *gin.Context) {
	if req, ok := s.bindObRequest(ctx); ok {
		if err := s.priceActionRepo.UpdateFvg(ctx, req); err != nil {
			s.sendError(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		s.sendSuccess(ctx, "Fvg updated", nil)
	}
}

func (s *PriceActionServiceImpl) DeleteFvg(ctx *gin.Context) {
	if req, ok := s.bindObRequest(ctx); ok {
		if err := s.priceActionRepo.DeleteFvgByDate(ctx, req.Symbol, req.Date); err != nil {
			s.sendError(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		s.sendSuccess(ctx, "Fvg deleted", nil)
	}
}

func (s *PriceActionServiceImpl) CheckFvgMitigation(ctx *gin.Context) {
	s.processMitigation(ctx, "BULLISH CLOSE 200", "FvgCache", false)
}

func (s *PriceActionServiceImpl) AutomateFvg(ctx *gin.Context) {
	rawStrategy, _ := cache.StrategyCache.Get("FAIR VALUE GAP")
	strategy, ok := rawStrategy.(model.StrategyDto)
	if !ok {
		s.sendError(ctx, http.StatusInternalServerError, "Fvg strategy error")
		return
	}

	data, _ := s.chartInkService.FetchWithMargin(strategy)
	for _, dto := range data {
		if history, err := s.nseService.FetchStockData(dto.Symbol); err == nil && len(history) >= 3 {
			if date, err := util.ParseNseDate(history[1].Timestamp); err == nil {
				s.priceActionRepo.SaveFvg(ctx, model.ObRequest{
					Symbol: dto.Symbol, Date: date, High: history[2].High, Low: history[0].Low,
				})
			}
		}
	}
	s.sendSuccess(ctx, "Fvg automation completed", nil)
}
