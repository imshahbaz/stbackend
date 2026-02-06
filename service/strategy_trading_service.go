package service

import (
	"backend/cache"
	"backend/database"
	"backend/model"
	"backend/repository"
	"backend/util"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

const (
	defaultStrategyName = "RSI15MIN"
	marketCloseHour     = 15
	marketCloseMinute   = 15
	marketSquareOffMin  = 30
	signalGracePeriod   = 20 * time.Minute
	signalExpiryPeriod  = 23 * time.Minute
)

type StrategyTradingService interface {
	ContinuousTrade(ctx context.Context, strategyName string) error
}

type StrategyTradingServiceImpl struct {
	chartInkService   ChartInkService
	strategyService   StrategyService
	strategyOrderRepo *repository.StrategyOrderRepository
	angelWS           AngelOneWebSocket
	zerodhaService    ZerodhaService
}

func NewStrategyTradingService(
	ci ChartInkService,
	ss StrategyService,
	sor *repository.StrategyOrderRepository,
	angelWS AngelOneWebSocket,
	zs ZerodhaService,
) StrategyTradingService {
	return &StrategyTradingServiceImpl{
		chartInkService:   ci,
		strategyService:   ss,
		strategyOrderRepo: sor,
		angelWS:           angelWS,
		zerodhaService:    zs,
	}
}

func (s *StrategyTradingServiceImpl) ContinuousTrade(ctx context.Context, strategyName string) error {
	if strategyName == "" {
		strategyName = defaultStrategyName
	}

	existingOrders, err := s.strategyOrderRepo.FindTodayOrdersByStrategy(ctx, strategyName)
	if err != nil {
		log.Error().Err(err).Str("strategy", strategyName).Msg("Failed to fetch today's strategy orders")
		return err
	}

	if len(existingOrders) == 0 {
		log.Info().Str("strategy", strategyName).Msg("No orders found for the strategy today")
		return nil
	}

	strategy, ok := s.strategyService.GetStrategyByName(strategyName)
	if !ok {
		log.Error().Str("strategy", strategyName).Msg("Strategy configuration not found")
		return fmt.Errorf("strategy %s not found", strategyName)
	}

	log.Info().Int("orderCount", len(existingOrders)).Str("strategy", strategyName).Msg("Starting strategy trading")

	s.startPoller(strategy)

	for _, order := range existingOrders {
		kc, err := s.getKiteClientForUser(ctx, order.UserID)
		if err != nil {
			log.Error().Err(err).Int64("userId", order.UserID).Msg("Skipping user due to session error")
			continue
		}

		go s.tradeLoop(order, kc, strategyName)
	}

	return nil
}

func (s *StrategyTradingServiceImpl) getKiteClientForUser(ctx context.Context, userID int64) (*kiteconnect.Client, error) {
	userIdStr := strconv.FormatInt(userID, 10)

	// Check local memory cache
	if val, ok := cache.KiteClientCache.Get(userIdStr); ok {
		if kc, ok := val.(*kiteconnect.Client); ok {
			return kc, nil
		}
	}

	// Check Redis for access token
	var accessToken string
	ok, err := database.RedisHelper.GetAsStruct("zerodha_token_"+userIdStr, &accessToken)
	if err != nil || !ok || accessToken == "" {
		return nil, fmt.Errorf("access token not found in redis for user %d", userID)
	}

	// Initiate KiteConnect
	kc, err := s.zerodhaService.InitiateKiteConnect(ctx, accessToken, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate kite connect: %w", err)
	}

	// Cache the client
	cache.KiteClientCache.Set(userIdStr, kc, util.ZerodhaTokenExpiry())

	return kc, nil
}

func (s *StrategyTradingServiceImpl) tradeLoop(order model.StrategyOrder, kc *kiteconnect.Client, strategyName string) {
	loopCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Info().Int64("userId", order.UserID).Str("strategy", strategyName).Msg("Started trade loop for user")

	for {
		select {
		case <-loopCtx.Done():
			log.Info().Int64("userId", order.UserID).Msg("Stopping trade loop: context canceled")
			return
		default:
			now := time.Now().In(util.IstLocation)
			if now.Hour() >= marketCloseHour && now.Minute() >= marketCloseMinute {
				log.Info().Int64("userId", order.UserID).Msg("Market closing. Stopping trade loop.")
				return
			}

			signals, found := s.getCachedSignals(strategyName)
			if !found || len(signals) == 0 {
				time.Sleep(1 * time.Minute)
				continue
			}

			targetStock, qty := s.findTargetStock(now, signals, order.Amount)
			if targetStock.Symbol == "" {
				time.Sleep(1 * time.Minute)
				continue
			}

			success := s.punchSingleTrade(kc, targetStock, qty)
			if !success {
				log.Warn().Int64("userId", order.UserID).Str("symbol", targetStock.Symbol).Msg("Punch trade failed or market exited. Ending loop for user.")
				return
			}

			log.Info().Int64("userId", order.UserID).Str("symbol", targetStock.Symbol).Msg("Target Achieved! Looking for next trade opportunity.")
		}
	}
}

func (s *StrategyTradingServiceImpl) findTargetStock(now time.Time, signals []model.ChartinkBacktestSignalWithMargin, orderAmount float64) (model.Margin, int) {
	for i := len(signals) - 1; i >= 0; i-- {
		signal := signals[i]
		marketTime, err := time.ParseInLocation("2006-01-02 15:04:05", signal.MarketTime, util.IstLocation)
		if err != nil {
			continue
		}

		// Check if signal is within the valid time window
		if now.After(marketTime.Add(signalGracePeriod)) && now.Before(marketTime.Add(signalExpiryPeriod)) && len(signal.Stocks) > 0 {
			target := signal.Stocks[0]
			s.angelWS.Subscribe(target.Token, model.NSE)

			// Wait for LTP update
			time.Sleep(1 * time.Second)
			ltp := s.angelWS.GetLTP(target.Token)
			if ltp <= 0 {
				continue
			}

			qty := int((orderAmount / ltp) * float64(target.Margin))
			if qty > 0 {
				return target, qty
			}
		}
	}
	return model.Margin{}, 0
}

func (s *StrategyTradingServiceImpl) punchSingleTrade(kc *kiteconnect.Client, targetStock model.Margin, qty int) bool {
	tradingCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timer := time.NewTimer(time.Hour)
	defer timer.Stop()

	log.Info().Str("symbol", targetStock.Symbol).Int("qty", qty).Msg("Placing entry order")
	orderResp, err := s.zerodhaService.PlaceMTFOrder(kc, targetStock.Symbol, qty, 0, kiteconnect.TransactionTypeBuy, kiteconnect.OrderTypeMarket)
	if err != nil {
		log.Error().Err(err).Str("symbol", targetStock.Symbol).Msg("Failed to place buy order")
		return false
	}

	// Wait for order to be filled and get average price
	time.Sleep(1 * time.Second)
	od, err := s.zerodhaService.GetOrderDetails(kc, orderResp.OrderID)
	if err != nil {
		log.Error().Err(err).Str("orderId", orderResp.OrderID).Msg("Failed to get entry order details")
		return false
	}

	targetPrice := util.FixToTick(od.AveragePrice * 1.004)
	log.Info().Str("symbol", targetStock.Symbol).Float64("entry", od.AveragePrice).Float64("target", targetPrice).Msg("Placing exit order")

	sOd, err := s.zerodhaService.PlaceMTFOrder(kc, targetStock.Symbol, qty, targetPrice, kiteconnect.TransactionTypeSell, kiteconnect.OrderTypeLimit)
	if err != nil {
		log.Error().Err(err).Str("symbol", targetStock.Symbol).Msg("Failed to place sell order")
		return false
	}

	var prevLtp float64
	for {
		now := time.Now().In(util.IstLocation)
		if now.Hour() >= marketCloseHour && now.Minute() >= marketSquareOffMin {
			log.Info().Str("symbol", targetStock.Symbol).Msg("Market square-off time reached. Exiting trade monitor.")
			return false
		}

		ltp := s.angelWS.GetLTP(targetStock.Token)
		if ltp == -2 { // Sentinel value for error/disconnection
			log.Error().Str("symbol", targetStock.Symbol).Msg("Lost LTP feed for stock")
			return false
		}

		if ltp >= targetPrice && ltp != prevLtp {
			det, err := s.zerodhaService.GetOrderDetails(kc, sOd.OrderID)
			if err == nil && det.PendingQuantity == 0 {
				log.Info().Str("symbol", targetStock.Symbol).Msg("Exit order filled")
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
		if now.Hour() >= marketCloseHour && now.Minute() > marketSquareOffMin {
			log.Info().Str("strategy", strategy.Name).Msg("Market closed. Stopping strategy poller.")
			if c != nil {
				c.Stop()
			}
			return
		}

		log.Info().Str("strategy", strategy.Name).Msg("Fetching strategy signals")
		signals, err := s.chartInkService.FetchBacktestTodayWithMargin(strategy)
		if err != nil {
			log.Error().Err(err).Str("strategy", strategy.Name).Msg("Signal fetch failed")
			return
		}

		cache.PollerCache.Set(strategy.Name, signals, 2*time.Minute)
	}

	var err error
	c, err = util.ScheduleTask(util.FifteenMinSpec, task)
	if err != nil {
		log.Error().Err(err).Str("strategy", strategy.Name).Msg("Failed to schedule strategy poller")
	}
}

func (s *StrategyTradingServiceImpl) getCachedSignals(strategyName string) ([]model.ChartinkBacktestSignalWithMargin, bool) {
	val, found := cache.PollerCache.Get(strategyName)
	if !found {
		return nil, false
	}
	signals, ok := val.([]model.ChartinkBacktestSignalWithMargin)
	return signals, ok
}
