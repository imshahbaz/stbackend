package routes

import (
	"backend/auth"
	"backend/client"
	"backend/config"
	"backend/controller"
	"backend/middleware"
	"backend/repository"
	"backend/service"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.mongodb.org/mongo-driver/mongo"
)

func SetupRouter(db *mongo.Database, cfg *config.SystemConfigs) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	if cfg.Config.DebugMode == "true" {
		r.Use(gin.Logger())
	}

	r.Use(middleware.CORS(cfg))
	isProduction := cfg.Config.Environment == "production"

	// --- 1. Clients ---
	brevoClient := client.NewBrevoClient()
	chartInkClient := client.NewChartinkClient()

	// --- 2. Repositories ---
	userRepo := repository.NewUserRepository(db)
	marginRepo := repository.NewMarginRepository(db)
	strategyRepo := repository.NewStrategyRepository(db)

	// --- 3. Services (Dependency Injection) ---
	emailSvc := service.NewEmailService(brevoClient, cfg.Config.BrevoApiKey)
	otpSvc := service.NewOtpService(emailSvc, cfg.Config.BrevoEmail)
	userSvc := service.NewUserService(userRepo)

	// Note: Margin leverage comes from config
	leverage, err := strconv.ParseFloat(cfg.Config.Leverage, 64)
	if err != nil {
		leverage = 4.0
	}

	marginSvc := service.NewMarginService(marginRepo, float32(leverage))
	strategySvc := service.NewStrategyService(strategyRepo)
	chartInkSvc := service.NewChartInkService(chartInkClient, marginSvc)
	nseSvc := service.NewNseService()
	auth.SecretKey = []byte((os.Getenv("jwtSecret")))

	if !isProduction {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

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
		controller.NewAuthController(userSvc, cfg, otpSvc).RegisterRoutes(api)

		controller.NewUserController(userSvc, isProduction).RegisterRoutes(api)

		controller.NewNseController(nseSvc).RegisterRoutes(api)
	}

	return r
}
