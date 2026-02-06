package service

import (
	"backend/cache"
	"backend/database"
	"backend/model"
	"backend/repository"
	"backend/util"
	"context"
	"strconv"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

type StrategyTradingService interface {
	ExecuteStrategy(strategyName string) error
}

type StrategyTradingServiceImpl struct {
	chartInkService   ChartInkService
	strategyService   StrategyService
	orderService      OrderService
	strategyOrderRepo *repository.StrategyOrderRepository
	angelWS           AngelOneWebSocket
	zerodhaService    ZerodhaService
}

func NewStrategyTradingService(
	ci ChartInkService,
	ss StrategyService,
	os OrderService,
	sor *repository.StrategyOrderRepository,
	angelWS AngelOneWebSocket,
	zs ZerodhaService,
) StrategyTradingService {
	return &StrategyTradingServiceImpl{
		chartInkService:   ci,
		strategyService:   ss,
		orderService:      os,
		strategyOrderRepo: sor,
		angelWS:           angelWS,
		zerodhaService:    zs,
	}
}

func (s *StrategyTradingServiceImpl) ExecuteStrategy(strategyName string) error {
	existingOrders, err := s.strategyOrderRepo.FindTodayOrdersByStrategy(context.Background(), strategyName)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch today's strategy orders")
	} else {
		log.Info().Int("count", len(existingOrders)).Msg("Fetched today's existing strategy orders")
	}

	if len(existingOrders) == 0 {
		log.Info().Msg("No orders found for the strategy today")
		return nil
	}

	for _, order := range existingOrders {
		var kc *kiteconnect.Client
		val, ok := cache.KiteClientCache.Get(strconv.FormatInt(order.UserID, 10))
		if !ok {
			var accessToken string
			ok, _ := database.RedisHelper.GetAsStruct("zerodha_token_"+strconv.FormatInt(order.UserID, 10), &accessToken)
			if !ok || accessToken == "" {
				log.Warn().Int64("userId", order.UserID).Msg("AccessToken not found in Redis for user")
				continue
			}

			kc, err = s.zerodhaService.InitiateKiteConnect(context.Background(), accessToken, order.UserID)
			if err != nil {
				log.Error().Err(err).Int64("userId", order.UserID).Msg("Failed to initiate KiteConnect for user")
				continue
			}
			cache.KiteClientCache.Set(strconv.FormatInt(order.UserID, 10), kc, util.ZerodhaTokenExpiry())
		} else {
			kc = val.(*kiteconnect.Client)
		}

		go s.tradeLoop(order, kc, strategyName)
	}
	return nil
}

func (s *StrategyTradingServiceImpl) tradeLoop(order model.StrategyOrder, kc *kiteconnect.Client, strategyName string) {
	for {
		now := time.Now().In(util.IstLocation)
		if now.Hour() >= 15 && now.Minute() >= 15 {
			log.Info().Msg("Market closing soon. Stopping trade loop.")
			return
		}

		signals, found := GetCachedSignals(strategyName)
		if !found || len(signals) == 0 {
			time.Sleep(1 * time.Minute)
			continue
		}

		var targetStock model.Margin
		var qty int
		for i := len(signals) - 1; i >= 0; i-- {
			signal := signals[i]
			marketTime, _ := time.ParseInLocation("2006-01-02 15:04:05", signal.MarketTime, util.IstLocation)
			if now.After(marketTime.Add(20*time.Minute)) && now.Before(marketTime.Add(23*time.Minute)) && len(signal.Stocks) > 0 {
				targetStock = signal.Stocks[0]
				s.angelWS.Subscribe(targetStock.Token, model.NSE)
				time.Sleep(1 * time.Second)
				ltp := s.angelWS.GetLTP(targetStock.Token)
				if ltp <= 0 {
					continue
				}

				qty = int((order.Amount / ltp) * float64(targetStock.Margin))
				if qty <= 0 {
					continue
				}

				break
			}
		}

		if targetStock.Symbol == "" {
			time.Sleep(1 * time.Minute)
			continue
		}

		success := s.punchSingleTrade(kc, targetStock, qty)

		if success {
			log.Info().Str("symbol", targetStock.Symbol).Msg("Target Achieved! Restarting loop for next trade.")
		} else {
			break
		}
	}
}

func (s *StrategyTradingServiceImpl) punchSingleTrade(kc *kiteconnect.Client, targetStock model.Margin, qty int) bool {
	tradingCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()

	orderResp, err := s.zerodhaService.PlaceMTFOrder(kc, targetStock.Symbol, qty, 0, kiteconnect.TransactionTypeBuy, kiteconnect.OrderTypeMarket)
	if err != nil {
		return false
	}

	time.Sleep(1 * time.Second)
	od, _ := s.zerodhaService.GetOrderDetails(kc, orderResp.OrderID)
	targetPrice := util.FixToTick(od.AveragePrice * 1.004)
	sOd, err := s.zerodhaService.PlaceMTFOrder(kc, targetStock.Symbol, qty, targetPrice, kiteconnect.TransactionTypeSell, kiteconnect.OrderTypeLimit)
	if err != nil {
		return false
	}

	var prevLtp float64

	for {
		if time.Now().In(util.IstLocation).Hour() >= 15 && time.Now().In(util.IstLocation).Minute() >= 30 {
			return false
		}
		ltp := s.angelWS.GetLTP(targetStock.Token)
		if ltp == -2 {
			return false
		}

		if ltp >= targetPrice && ltp != prevLtp {
			det, err := s.zerodhaService.GetOrderDetails(kc, sOd.OrderID)
			if err == nil && det.PendingQuantity == 0 {
				return true
			}
			prevLtp = ltp
		}

		if !util.PollWait(tradingCtx, timer) {
			return false
		}
	}

}

func (s *StrategyTradingServiceImpl) startPoller(strategy model.StrategyDto) {
	var c *cron.Cron
	task := func() {
		now := time.Now().In(util.IstLocation)
		if now.Hour() == 15 && now.Minute() > 30 {
			log.Info().Msg("Market closed. Skipping cron task.")
			c.Stop()
			return
		}

		log.Info().Str("strategy", strategy.Name).Msg("Cron Trigger: Fetching shared signals")

		signals, err := s.chartInkService.FetchBacktestTodayWithMargin(strategy)
		if err != nil {
			log.Error().Err(err).Msg("Cron fetch failed")
			return
		}

		cache.PollerCache.Set(strategy.Name, signals, 2*time.Minute)
	}

	c, _ = util.ScheduleTask(util.FifteenMinSpec, task)
}

func GetCachedSignals(strategyName string) ([]model.ChartinkBacktestSignalWithMargin, bool) {
	val, found := cache.PollerCache.Get(strategyName)
	if !found {
		return nil, false
	}
	return val.([]model.ChartinkBacktestSignalWithMargin), true
}
