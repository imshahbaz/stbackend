//go:build wireinject
// +build wireinject

//go:generate go run github.com/google/wire/cmd/wire

package di

import (
	"backend/auth"
	"backend/client"
	"backend/config"
	"backend/controller"
	"backend/database"
	"backend/model"
	"backend/repository"
	"backend/service"
	"context"

	"github.com/google/wire"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/oauth2"
)

func provideIsProduction(sysCfg *config.SystemConfigs) service.IsProduction {
	return service.IsProduction(sysCfg.Config.Environment == "production")
}

func provideMongoDatabase(sysCfg *config.SystemConfigs) (*mongo.Database, func()) {
	client, db := database.InitMongoClient(sysCfg)
	cleanup := func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Error().Err(err).Msg("Failed to disconnect MongoDB")
		} else {
			log.Info().Msg("MongoDB disconnected successfully")
		}
	}
	return db, cleanup
}

func provideConfigManager(cfgSvc service.ConfigService) (*config.ConfigManager, func()) {
	cm := cfgSvc.GetConfigManager()
	auth.SecretKey = []byte(cm.GetConfig().JwtSecret)
	database.InitRedis(cm.GetConfig().RedisUrl)
	cleanup := func() {
		if database.RedisHelper != nil {
			database.RedisHelper.Close()
			log.Info().Msg("Valkey/Redis connection closed")
		}
	}
	return cm, cleanup
}

func provideAngelOneConfig(cm *config.ConfigManager) *model.AngelOneConfig {
	return &cm.GetConfig().AngelOneConfig
}

func provideGoogleOAuthConfig(cm *config.ConfigManager) *oauth2.Config {
	return auth.GetGoogleOAuthConfig(cm.GetConfig().GoogleAuth)
}

func provideBrevoClient() *client.BrevoClient       { return client.NewBrevoClient() }
func provideChartinkClient() *client.ChartinkClient { return client.NewChartinkClient() }
func provideYahooClient() *client.YahooClient       { return client.NewYahooClient() }
func provideGenAiClient(cm *config.ConfigManager) *client.GenAiClient {
	return client.NewGenAiClient(cm.GetConfig().GoogleAuth)
}

func provideAngelOneWebSocket(conf *model.AngelOneConfig) (service.AngelOneWebSocket, func()) {
	ws := service.NewAngelOneWebSocket("", "", conf)
	cleanup := func() {
		ws.Disconnect()
		log.Info().Msg("Angel One WebSocket disconnected")
	}
	return ws, cleanup
}

var RepositorySet = wire.NewSet(
	repository.NewUserRepository,
	repository.NewMarginRepository,
	repository.NewStrategyRepository,
	repository.NewPriceActionRepo,
	repository.NewOrderRepo,
	repository.NewOptionRepository,
	repository.NewStrategyOrderRepository,
)

var ServiceSet = wire.NewSet(
	service.NewConfigService,
	service.NewEmailService,
	service.NewOtpService,
	service.NewUserService,
	service.NewMarginService,
	service.NewStrategyService,
	service.NewNseService,
	service.NewChartInkService,
	service.NewPriceActionService,
	service.NewOAuthService,
	service.NewAuthService,
	service.NewNewsService,
	service.NewZerodhaService,
	service.NewAngelOneService,
	service.NewOrderService,
	service.NewMstockService,
	service.NewStrategyTradingService,
	service.NewStrategyOrderService,
	service.NewFcmService,
)

var ControllerSet = wire.NewSet(
	controller.NewHealthController,
	controller.NewEmailController,
	controller.NewMarginController,
	controller.NewStrategyController,
	controller.NewChartInkController,
	controller.NewAuthController,
	controller.NewUserController,
	controller.NewNseController,
	controller.NewConfigController,
	controller.NewPriceActionController,
	controller.NewNewsController,
	controller.NewZerodhaController,
	controller.NewOrderController,
	controller.NewAngelOneController,
	controller.NewMstockController,
	controller.NewStrategyTradingController,
	controller.NewStrategyOrderController,
)

func InitializeApp(sysCfg *config.SystemConfigs) (*App, func(), error) {
	wire.Build(
		provideIsProduction,
		provideMongoDatabase,
		provideConfigManager,
		provideAngelOneConfig,
		provideGoogleOAuthConfig,
		provideBrevoClient,
		provideChartinkClient,
		provideYahooClient,
		provideGenAiClient,
		provideAngelOneWebSocket,
		RepositorySet,
		ServiceSet,
		ControllerSet,
		SetupRouterWrapper,
	)
	return nil, nil, nil
}
