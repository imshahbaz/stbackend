package routes

import (
	"backend/client"
	"backend/config"
	"backend/controller"
	"backend/repository"
	"backend/service"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.mongodb.org/mongo-driver/mongo"
)

func SetupRouter(db *mongo.Database, cfg *config.SystemConfigs) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"}, // Your Frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// --- 1. Clients ---
	brevoClient := client.NewBrevoClient()
	chartInkClient := client.NewChartinkClient()

	// --- 2. Repositories ---
	//userRepo := repository.NewUserRepository(db)
	marginRepo := repository.NewMarginRepository(db)
	strategyRepo := repository.NewStrategyRepository(db)

	// --- 3. Services (Dependency Injection) ---
	emailSvc := service.NewEmailService(brevoClient, cfg.Config.BrevoApiKey)
	// otpSvc := service.NewOtpService(emailSvc, cfg.Config.BrevoEmail)
	// userSvc := service.NewUserService(userRepo)

	// Note: Margin leverage comes from config
	marginSvc := service.NewMarginService(marginRepo, 4.0)
	strategySvc := service.NewStrategyService(strategyRepo)
	chartInkSvc := service.NewChartInkService(chartInkClient)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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
		controller.NewStrategyController(strategySvc).RegisterRoutes(api)

		// ChartInk Endpoints
		controller.NewChartInkController(chartInkSvc, strategySvc).RegisterRoutes(api)

		// User/Auth Endpoints (Once implemented)
		// controller.NewUserController(userSvc, otpSvc).RegisterRoutes(api)
	}

	return r
}
