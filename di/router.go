package di

import (
	"backend/config"
	"backend/controller"
	"backend/middleware"
	"backend/service"
	"context"
	"io"

	"github.com/bytedance/sonic"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type App struct {
	Router         *gin.Engine
	MarginSvc      service.MarginService
	StrategySvc    service.StrategyService
	AngelOneSvc    service.AngelOneService
	AngelOneWebSvc service.AngelOneWebSocket
	FcmSvc         service.FcmService
}

func (a *App) Start() {
	go func() {
		log.Info().Msg("Loading margins...")
		if err := a.MarginSvc.ReloadAllMargins(context.Background()); err != nil {
			log.Info().Msgf("Warning: Failed initial margin load: %v", err)
		} else {
			log.Info().Msg("Margins loaded on startup...")
		}
	}()

	go func() {
		log.Info().Msg("Loading options...")
		if err := a.MarginSvc.ReloadAllOptions(context.Background()); err != nil {
			log.Info().Msgf("Warning: Failed initial option load: %v", err)
		} else {
			log.Info().Msg("Options loaded on startup...")
		}
	}()

	go func() {
		log.Info().Msg("Loading strategies...")
		if err := a.StrategySvc.ReloadAllStrategies(context.Background()); err != nil {
			log.Info().Msgf("Warning: Failed initial strategies load: %v", err)
		} else {
			log.Info().Msg("Strategies loaded on startup...")
		}
	}()

	go func() {
		log.Info().Msg("Refreshing Angel One broker session...")
		if jwt, feedToken, err := a.AngelOneSvc.RefreshBrokerSession(); err != nil {
			log.Error().Err(err).Msg("Warning: Failed initial Angel One session refresh")
		} else {
			log.Info().Msg("Angel One broker session refreshed on startup...")
			a.AngelOneWebSvc.UpdateConfig(jwt, feedToken)
		}
	}()

	go func() {
		if a.FcmSvc != nil {
			log.Info().Msg("FCM client initialized on app startup")
		} else {
			log.Warn().Msg("FCM client was not initialized - please check configuration")
		}
	}()
}

func SetupRouterWrapper(
	isProd service.IsProduction,
	cm *config.ConfigManager,
	healthCtrl *controller.HealthController,
	emailCtrl *controller.EmailController,
	marginCtrl *controller.MarginController,
	strategyCtrl *controller.StrategyController,
	chartInkCtrl *controller.ChartInkController,
	authCtrl *controller.AuthController,
	userCtrl *controller.UserController,
	nseCtrl *controller.NseController,
	configCtrl *controller.ConfigController,
	paCtrl *controller.PriceActionController,
	newsCtrl *controller.NewsController,
	zerodhaCtrl *controller.ZerodhaController,
	orderCtrl *controller.OrderController,
	angelOneCtrl *controller.AngelOneController,
	mstockCtrl *controller.MstockController,
	strategyTradingCtrl *controller.StrategyTradingController,
	strategyOrderCtrl *controller.StrategyOrderController,
	marginSvc service.MarginService,
	strategySvc service.StrategyService,
	angelOneSvc service.AngelOneService,
	angelOneWebSvc service.AngelOneWebSocket,
	fcmSvc service.FcmService,
) *App {
	r := gin.New()
	r.Use(middleware.RecoveryMiddleware)
	r.Use(middleware.ZerologMiddleware())
	r.Use(middleware.CORS(cm))
	r.Use(middleware.RateLimiter(cm))

	humaConfig := huma.DefaultConfig("Shahbaz Trades Management API", "1.0.0")
	if bool(isProd) {
		humaConfig.DocsPath = ""
		humaConfig.OpenAPIPath = ""
	}
	humaConfig.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearer": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
		},
	}

	humaConfig.Formats["application/json"] = huma.Format{
		Marshal: func(w io.Writer, v any) error {
			return sonic.ConfigDefault.NewEncoder(w).Encode(v)
		},
		Unmarshal: sonic.Unmarshal,
	}

	humaApi := humagin.New(r, humaConfig)

	healthCtrl.RegisterRoutes(humaApi)
	emailCtrl.RegisterRoutes(humaApi)
	marginCtrl.RegisterRoutes(humaApi)
	strategyCtrl.RegisterRoutes(humaApi)
	chartInkCtrl.RegisterRoutes(humaApi)
	authCtrl.RegisterRoutes(humaApi)
	userCtrl.RegisterRoutes(humaApi)
	nseCtrl.RegisterRoutes(humaApi)
	configCtrl.RegisterRoutes(humaApi)
	paCtrl.RegisterRoutes(humaApi)
	newsCtrl.RegisterRoutes(humaApi)
	zerodhaCtrl.RegisterRoutes(humaApi)
	orderCtrl.RegisterRoutes(humaApi)
	angelOneCtrl.RegisterRoutes(humaApi)
	mstockCtrl.RegisterRoutes(humaApi)
	strategyTradingCtrl.RegisterRoutes(humaApi)
	strategyOrderCtrl.RegisterRoutes(humaApi)

	return &App{
		Router:         r,
		MarginSvc:      marginSvc,
		StrategySvc:    strategySvc,
		AngelOneSvc:    angelOneSvc,
		AngelOneWebSvc: angelOneWebSvc,
		FcmSvc:         fcmSvc,
	}
}
