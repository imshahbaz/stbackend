package routes

import (
	"backend/auth"
	"backend/client"
	"backend/config"
	"backend/controller"
	"backend/middleware"
	"backend/repository"
	"backend/service"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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

	// --- 1. Clients ---
	brevoClient := client.NewBrevoClient()
	chartInkClient := client.NewChartinkClient()

	// --- 2. Repositories ---
	userRepo := repository.NewUserRepository(db)
	marginRepo := repository.NewMarginRepository(db)
	strategyRepo := repository.NewStrategyRepository(db)

	// --- 3. Services (Dependency Injection) ---
	emailSvc := service.NewEmailService(brevoClient, configmanager)
	otpSvc := service.NewOtpService(emailSvc, configmanager)
	userSvc := service.NewUserService(userRepo)

	marginSvc := service.NewMarginService(marginRepo, configmanager)
	strategySvc := service.NewStrategyService(strategyRepo)
	chartInkSvc := service.NewChartInkService(chartInkClient, marginSvc)
	yahooClient := client.NewYahooClient()
	nseSvc := service.NewNseService(yahooClient)
	auth.SecretKey = []byte(configmanager.GetConfig().JwtSecret)

	if !isProduction {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	priceActionRepo := repository.NewPriceActionRepo(db)
	priceActionSvc := service.NewPriceActionService(chartInkSvc, nseSvc, priceActionRepo, marginSvc)

	// --- 4. Routes & Controllers ---
	api := r.Group("/api")
	{

		// Health Check
		controller.NewHealthController().RegisterRoutes(api)

		// Email Endpoints
		controller.NewEmailController(emailSvc).RegisterRoutes(api)

		// Margin Endpoints
		controller.NewMarginController(marginSvc).RegisterRoutes(api)

		// Strategy Endpoints
		controller.NewStrategyController(strategySvc, isProduction).RegisterRoutes(api)

		// ChartInk Endpoints
		controller.NewChartInkController(chartInkSvc, strategySvc).RegisterRoutes(api)

		//User/Auth Endpoints (Once implemented)
		controller.NewAuthController(userSvc, configmanager, otpSvc, isProduction).RegisterRoutes(api)

		controller.NewUserController(userSvc, isProduction).RegisterRoutes(api)

		controller.NewNseController(nseSvc).RegisterRoutes(api)

		controller.NewConfigController(configService, isProduction).RegisterRoutes(api)

		controller.NewPriceActionController(priceActionSvc, isProduction).RegisterRoutes(api)
	}

	return r
}
