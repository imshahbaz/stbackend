package routes

import (
	"backend/auth"
	"backend/client"
	"backend/config"
	"backend/controller"
	"backend/middleware"
	"backend/repository"
	"backend/service"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func SetupRouter(db *mongo.Database, cfg *config.SystemConfigs) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	isProduction := cfg.Config.Environment == "production"
	mongoId := "mongoConfigDev"
	if isProduction {
		mongoId = "mongoConfig"
	}
	configService := service.NewConfigService(db, mongoId)
	configmanager := configService.GetConfigManager()

	if configmanager.GetConfig().DebugMode {
		r.Use(gin.Logger())
	}

	r.Use(middleware.CORS(configmanager))
	r.Use(middleware.RateLimiter(configmanager))

	brevoClient := client.NewBrevoClient()
	chartInkClient := client.NewChartinkClient()

	userRepo := repository.NewUserRepository(db)
	marginRepo := repository.NewMarginRepository(db)
	strategyRepo := repository.NewStrategyRepository(db)

	emailSvc := service.NewEmailService(brevoClient, configmanager)
	otpSvc := service.NewOtpService(emailSvc, configmanager)
	userSvc := service.NewUserService(userRepo)

	marginSvc := service.NewMarginService(marginRepo, configmanager)
	strategySvc := service.NewStrategyService(strategyRepo)
	chartInkSvc := service.NewChartInkService(chartInkClient, marginSvc)
	yahooClient := client.NewYahooClient()
	nseSvc := service.NewNseService(yahooClient)
	auth.SecretKey = []byte(configmanager.GetConfig().JwtSecret)

	priceActionRepo := repository.NewPriceActionRepo(db)
	priceActionSvc := service.NewPriceActionService(chartInkSvc, nseSvc, priceActionRepo, marginSvc)


	humaConfig := huma.DefaultConfig("Shahbaz Trades Management API", "1.0.0")
	if isProduction {
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
	humaApi := humagin.New(r, humaConfig)

	{
		controller.NewHealthController().RegisterRoutes(humaApi)

		controller.NewEmailController(emailSvc).RegisterRoutes(humaApi)

		controller.NewMarginController(marginSvc).RegisterRoutes(humaApi)

		controller.NewStrategyController(strategySvc, isProduction).RegisterRoutes(humaApi)

		controller.NewChartInkController(chartInkSvc, strategySvc).RegisterRoutes(humaApi)

		controller.NewAuthController(userSvc, configmanager, otpSvc, isProduction).RegisterRoutes(humaApi)

		controller.NewUserController(userSvc, isProduction).RegisterRoutes(humaApi)

		controller.NewNseController(nseSvc).RegisterRoutes(humaApi)

		controller.NewConfigController(configService, isProduction).RegisterRoutes(humaApi)

		controller.NewPriceActionController(priceActionSvc, isProduction).RegisterRoutes(humaApi)
	}

	return r
}
