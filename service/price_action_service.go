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
	SaveOrderBlock(ctx *gin.Context)
	DeleteOrderBlock(ctx *gin.Context)
	CheckOBMitigation(ctx *gin.Context)
	GetObBySymbol(ctx *gin.Context)
	AutomateOrderBlock(ctx *gin.Context)
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

func (s *PriceActionServiceImpl) SaveOrderBlock(ctx *gin.Context) {
	var request model.ObRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, model.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	err := s.priceActionRepo.SaveOrderBlock(ctx, request)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, model.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, model.Response{
		Success: true,
		Message: "Order block created",
	})
}

func (s *PriceActionServiceImpl) DeleteOrderBlock(ctx *gin.Context) {
	var request model.ObRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, model.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	err := s.priceActionRepo.DeleteOrderBlockByDate(ctx, request.Symbol, request.Date)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, model.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Success: true,
		Message: "Order block deleted",
	})
}

func (s *PriceActionServiceImpl) CheckOBMitigation(ctx *gin.Context) {
	rawStrategy, ok := cache.StrategyCache.Get("BULLISH CLOSE 200")

	strategy, ok := rawStrategy.(model.StrategyDto)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, model.Response{
			Success: false,
			Error:   "OB strategy error",
		})
		return
	}

	data, err := s.chartInkService.FetchWithMargin(strategy)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, model.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	idMap := make(map[string]model.StockMarginDto, len(data))
	for _, dto := range data {
		idMap[dto.Symbol] = dto
	}

	ids := make([]string, 0, len(data))
	for _, stock := range data {
		ids = append(ids, stock.Symbol)
	}

	obs, err := s.priceActionRepo.GetAllObIn(ctx, ids)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, model.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	response := make([]model.ObResponse, 0, len(obs))
	for _, ob := range obs {
		history, err := s.nseService.FetchStockData(ob.Symbol)
		if err != nil {
			continue
		}
		today := history[0]
		for _, block := range ob.OrderBlocks {
			formattedDate, err := util.ParseNseDate(history[1].Timestamp)
			if err != nil {
				continue
			}
			if (today.Low < block.High || today.Low < block.Low) && today.Close > block.High && formattedDate != block.Date {
				dto, _ := idMap[ob.Symbol]
				var obResp model.ObResponse
				copier.Copy(&obResp, &dto)
				obResp.Date = block.Date
				response = append(response, obResp)
				break
			}
		}
	}

	if len(response) > 0 {
		cache.PriceActionCache.Set("ObCache", response, -1)
	}

	ctx.JSON(http.StatusOK, model.Response{
		Success: true,
		Message: "Order block fetch success",
		Data:    response,
	})
}

func (s *PriceActionServiceImpl) GetObBySymbol(ctx *gin.Context) {
	symbol := ctx.Param("symbol")

	if symbol == "" {
		ctx.JSON(http.StatusBadRequest, model.Response{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	data, err := s.priceActionRepo.GetObByID(ctx, symbol)
	if err != nil {
		ctx.JSON(http.StatusNotFound, model.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, model.Response{
		Success: true,
		Message: "Order block found",
		Data:    data,
	})
}

func (s *PriceActionServiceImpl) UpdateOrderBlock(ctx *gin.Context) {
	var request model.ObRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, model.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	err := s.priceActionRepo.UpdateOrderBlock(ctx, request)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, model.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, model.Response{
		Success: true,
		Message: "Order block updated",
	})
}

func (s *PriceActionServiceImpl) AutomateOrderBlock(ctx *gin.Context) {
	rawStrategy, ok := cache.StrategyCache.Get("BULLISH OB 1D")

	strategy, ok := rawStrategy.(model.StrategyDto)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, model.Response{
			Success: false,
			Error:   "OB strategy error",
		})
		return
	}

	data, err := s.chartInkService.FetchWithMargin(strategy)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, model.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	for _, dto := range data {
		history, err := s.nseService.FetchStockData(dto.Symbol)
		if err != nil {
			continue
		}
		candle := history[1]
		formattedDate, err := util.ParseNseDate(candle.Timestamp)
		if err != nil {
			continue
		}
		request := model.ObRequest{
			Symbol: dto.Symbol,
			Date:   formattedDate,
			High:   candle.High,
			Low:    candle.Low,
		}
		s.priceActionRepo.SaveOrderBlock(ctx, request)
	}

	ctx.JSON(http.StatusOK, model.Response{
		Success: true,
		Message: "Order block automation completed",
	})

}
