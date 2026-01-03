package routes

import (
	"backend/auth"
	"backend/client"
	"backend/config"
	"backend/controller"
	"backend/database"
	"backend/middleware"
	"backend/repository"
	"backend/service"
	"io"

	"github.com/bytedance/sonic"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/oauth2"
)

var configmanager *config.ConfigManager

var (
	brevoClient    *client.BrevoClient
	chartInkClient *client.ChartinkClient
	yahooClient    *client.YahooClient
	googleAuth     *oauth2.Config
)

var (
	userRepo        *repository.UserRepository
	marginRepo      *repository.MarginRepository
	strategyRepo    *repository.StrategyRepository
	priceActionRepo *repository.PriceActionRepo
)

var (
	emailSvc       service.EmailService
	otpSvc         service.OtpService
	userSvc        service.UserService
	marginSvc      service.MarginService
	strategySvc    service.StrategyService
	chartInkSvc    service.ChartInkService
	nseSvc         service.NseService
	priceActionSvc service.PriceActionService
)

func SetupRouter(db *mongo.Database, cfg *config.SystemConfigs) *gin.Engine {

	isProduction := cfg.Config.Environment == "production"

	configService := service.NewConfigService(db, isProduction)

	r := initApp(configService, db)

	auth.SecretKey = []byte(configmanager.GetConfig().JwtSecret)

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

	humaConfig.Formats["application/json"] = huma.Format{
		Marshal: func(w io.Writer, v any) error {
			return sonic.ConfigDefault.NewEncoder(w).Encode(v)
		},
		Unmarshal: sonic.Unmarshal,
	}

	humaApi := humagin.New(r, humaConfig)

	{
		controller.NewHealthController().RegisterRoutes(humaApi)

		controller.NewEmailController(emailSvc).RegisterRoutes(humaApi)

		controller.NewMarginController(marginSvc).RegisterRoutes(humaApi)

		controller.NewStrategyController(strategySvc, isProduction).RegisterRoutes(humaApi)

		controller.NewChartInkController(chartInkSvc, strategySvc).RegisterRoutes(humaApi)

		controller.NewAuthController(userSvc, configmanager, otpSvc, isProduction, googleAuth).RegisterRoutes(humaApi)

		controller.NewUserController(userSvc, isProduction, otpSvc).RegisterRoutes(humaApi)

		controller.NewNseController(nseSvc).RegisterRoutes(humaApi)

		controller.NewConfigController(configService, isProduction).RegisterRoutes(humaApi)

		controller.NewPriceActionController(priceActionSvc, isProduction).RegisterRoutes(humaApi)
	}

	return r
}

func initApp(configService service.ConfigService, db *mongo.Database) *gin.Engine {
	configmanager = configService.GetConfigManager()
	r := initGinEngine()
	initDB()
	initClients()
	initRepos(db)
	initsvcs()
	return r
}

func initGinEngine() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	if configmanager.GetConfig().DebugMode {
		r.Use(gin.Logger())
	}

	r.Use(middleware.CORS(configmanager))
	r.Use(middleware.RateLimiter(configmanager))
	return r
}

func initDB() {
	database.InitRedis(configmanager.GetConfig().RedisUrl)
}

func initClients() {
	brevoClient = client.NewBrevoClient()
	chartInkClient = client.NewChartinkClient()
	yahooClient = client.NewYahooClient()
	googleAuth = auth.GetGoogleOAuthConfig(configmanager.GetConfig().GoogleAuth)
}

func initRepos(db *mongo.Database) {
	userRepo = repository.NewUserRepository(db)
	marginRepo = repository.NewMarginRepository(db)
	strategyRepo = repository.NewStrategyRepository(db)
	priceActionRepo = repository.NewPriceActionRepo(db)
}

func initsvcs() {
	emailSvc = service.NewEmailService(brevoClient, configmanager)
	otpSvc = service.NewOtpService(emailSvc, configmanager)
	userSvc = service.NewUserService(userRepo)
	marginSvc = service.NewMarginService(marginRepo, configmanager)
	strategySvc = service.NewStrategyService(strategyRepo)
	chartInkSvc = service.NewChartInkService(chartInkClient, marginSvc)
	nseSvc = service.NewNseService(yahooClient)
	priceActionSvc = service.NewPriceActionService(chartInkSvc, nseSvc, priceActionRepo, marginSvc)
}
