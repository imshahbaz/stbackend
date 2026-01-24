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
	"context"
	"io"

	"sync"

	"github.com/bytedance/sonic"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/oauth2"
)

var configmanager *config.ConfigManager

var (
	brevoClient    *client.BrevoClient
	chartInkClient *client.ChartinkClient
	yahooClient    *client.YahooClient
	googleAuth     *oauth2.Config
	genAiClient    *client.GenAiClient
)

var (
	userRepo        *repository.UserRepository
	marginRepo      *repository.MarginRepository
	strategyRepo    *repository.StrategyRepository
	priceActionRepo *repository.PriceActionRepo
	orderRepo       *repository.OrderRepo
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
	oauthSvc       service.OAuthService
	authSvc        service.AuthService
	newsSvc        service.NewsService
	zerodhaSvc     service.ZerodhaService
	orderSvc       service.OrderService
	angelOneSvc    service.AngelOneService
)

func SetupRouter(db *mongo.Database, cfg *config.SystemConfigs) *gin.Engine {

	isProduction := cfg.Config.Environment == "production"

	configService := service.NewConfigService(db, isProduction)

	r := initApp(configService, db, isProduction)

	auth.SecretKey = []byte(configmanager.GetConfig().JwtSecret)

	humaConfig := *getHumaConfig(isProduction)

	humaApi := humagin.New(r, humaConfig)

	{
		controller.NewHealthController().RegisterRoutes(humaApi)

		controller.NewEmailController(emailSvc).RegisterRoutes(humaApi)

		controller.NewMarginController(marginSvc).RegisterRoutes(humaApi)

		controller.NewStrategyController(strategySvc, isProduction).RegisterRoutes(humaApi)

		controller.NewChartInkController(chartInkSvc, strategySvc).RegisterRoutes(humaApi)

		controller.NewAuthController(userSvc, configmanager, otpSvc, isProduction, oauthSvc, authSvc).RegisterRoutes(humaApi)

		controller.NewUserController(userSvc, isProduction, otpSvc).RegisterRoutes(humaApi)

		controller.NewNseController(nseSvc).RegisterRoutes(humaApi)

		controller.NewConfigController(configService, isProduction).RegisterRoutes(humaApi)

		controller.NewPriceActionController(priceActionSvc, isProduction).RegisterRoutes(humaApi)

		controller.NewNewsController(newsSvc).RegisterRoutes(humaApi)

		controller.NewZerodhaController(zerodhaSvc, isProduction, userSvc).RegisterRoutes(humaApi)

		controller.NewOrderController(orderSvc, isProduction).RegisterRoutes(humaApi)

		controller.NewAngelOneController(angelOneSvc, isProduction).RegisterRoutes(humaApi)
	}

	return r
}

func initApp(configService service.ConfigService, db *mongo.Database, isProduction bool) *gin.Engine {
	configmanager = configService.GetConfigManager()
	r := initGinEngine()

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		initClients()
	}()

	go func() {
		defer wg.Done()
		initRepos(db)
	}()

	go func() {
		defer wg.Done()
		initDB()
	}()

	wg.Wait()
	initsvcs(isProduction)
	return r
}

func initGinEngine() *gin.Engine {
	r := gin.New()
	r.Use(middleware.RecoveryMiddleware)
	r.Use(middleware.ZerologMiddleware())
	r.Use(middleware.CORS(configmanager))
	r.Use(middleware.RateLimiter(configmanager))
	return r
}

func initDB() {
	log.Info().Msg("Initialising redis...")
	database.InitRedis(configmanager.GetConfig().RedisUrl)
}

func initClients() {
	brevoClient = client.NewBrevoClient()
	chartInkClient = client.NewChartinkClient()
	yahooClient = client.NewYahooClient()
	googleAuth = auth.GetGoogleOAuthConfig(configmanager.GetConfig().GoogleAuth)
	genAiClient = client.NewGenAiClient(configmanager.GetConfig().GoogleAuth)
}

func initRepos(db *mongo.Database) {
	userRepo = repository.NewUserRepository(db)
	marginRepo = repository.NewMarginRepository(db)
	strategyRepo = repository.NewStrategyRepository(db)
	priceActionRepo = repository.NewPriceActionRepo(db)
	orderRepo = repository.NewOrderRepo(db)
}

func initsvcs(isProduction bool) {
	emailSvc = service.NewEmailService(brevoClient, configmanager)
	otpSvc = service.NewOtpService(emailSvc, configmanager)
	userSvc = service.NewUserService(userRepo)
	marginSvc = service.NewMarginService(marginRepo, configmanager)
	strategySvc = service.NewStrategyService(strategyRepo)
	nseSvc = service.NewNseService(yahooClient)
	chartInkSvc = service.NewChartInkService(chartInkClient, marginSvc, nseSvc)
	priceActionSvc = service.NewPriceActionService(chartInkSvc, nseSvc, priceActionRepo, marginSvc)
	oauthSvc = service.NewOAuthService(userSvc, configmanager, isProduction, googleAuth)
	authSvc = service.NewAuthService(userSvc, otpSvc, isProduction)
	newsSvc = service.NewNewsService(genAiClient, nseSvc)
	zerodhaSvc = service.NewZerodhaService(userSvc)
	angelOneSvc = service.NewAngelOneService(&configmanager.GetConfig().AngelOneConfig)
	orderSvc = service.NewOrderService(orderRepo, zerodhaSvc, angelOneSvc)

	go loadInitialData()
}

func getHumaConfig(isProduction bool) *huma.Config {
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

	return &humaConfig
}

func loadInitialData() {
	go func() {
		log.Info().Msg("Loading margins...")
		if err := marginSvc.ReloadAllMargins(context.Background()); err != nil {
			log.Info().Msgf("Warning: Failed initial margin load: %v", err)
		} else {
			log.Info().Msg("Margins loaded on startup...")
		}
	}()

	go func() {
		log.Info().Msg("Loading strategies...")
		if err := strategySvc.ReloadAllStrategies(context.Background()); err != nil {
			log.Info().Msgf("Warning: Failed initial strategies load: %v", err)
		} else {
			log.Info().Msg("Strategies loaded on startup...")
		}
	}()

	go func() {
		log.Info().Msg("Refreshing Angel One broker session...")
		if err := angelOneSvc.RefreshBrokerSession(); err != nil {
			log.Error().Err(err).Msg("Warning: Failed initial Angel One session refresh")
		} else {
			log.Info().Msg("Angel One broker session refreshed on startup...")
		}
	}()
}
