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

	"github.com/bytedance/sonic"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
	"github.com/gofiber/fiber/v2"
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
	oauthSvc       service.OAuthService
	authSvc        service.AuthService
)

func SetupRouter(db *mongo.Database, cfg *config.SystemConfigs) *fiber.App {
	isProduction := cfg.Config.Environment == "production"

	configService := service.NewConfigService(db, isProduction)
	r := initApp(configService, db, isProduction)

	auth.SecretKey = []byte(configmanager.GetConfig().JwtSecret)

	humaConfig := *getHumaConfig(isProduction)
	humaApi := humafiber.New(r, humaConfig)

	// Register Routes
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

	return r
}

func initApp(configService service.ConfigService, db *mongo.Database, isProduction bool) *fiber.App {
	configmanager = configService.GetConfigManager()

	r := initFiberApp(isProduction)
	initDB()
	initClients()
	initRepos(db)
	initsvcs(isProduction)

	return r
}

func initFiberApp(isProd bool) *fiber.App {
	r := fiber.New(fiber.Config{
		DisableStartupMessage: isProd,
		Prefork:               isProd,
		JSONEncoder:           sonic.Marshal,
		JSONDecoder:           sonic.Unmarshal,
	})

	r.Use(middleware.RecoveryMiddleware)
	r.Use(middleware.CORS(configmanager))
	r.Use(middleware.ZerologMiddleware())
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
}

func initRepos(db *mongo.Database) {
	userRepo = repository.NewUserRepository(db)
	marginRepo = repository.NewMarginRepository(db)
	strategyRepo = repository.NewStrategyRepository(db)
	priceActionRepo = repository.NewPriceActionRepo(db)
}

func initsvcs(isProduction bool) {
	emailSvc = service.NewEmailService(brevoClient, configmanager)
	otpSvc = service.NewOtpService(emailSvc, configmanager)
	userSvc = service.NewUserService(userRepo)
	marginSvc = service.NewMarginService(marginRepo, configmanager)
	strategySvc = service.NewStrategyService(strategyRepo)
	chartInkSvc = service.NewChartInkService(chartInkClient, marginSvc)
	nseSvc = service.NewNseService(yahooClient)
	priceActionSvc = service.NewPriceActionService(chartInkSvc, nseSvc, priceActionRepo, marginSvc)
	oauthSvc = service.NewOAuthService(userSvc, configmanager, isProduction, googleAuth)
	authSvc = service.NewAuthService(userSvc, otpSvc, isProduction)

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
			log.Error().Err(err).Msg("Failed initial margin load")
		} else {
			log.Info().Msg("Margins loaded successfully")
		}
	}()

	go func() {
		log.Info().Msg("Loading strategies...")
		if err := strategySvc.ReloadAllStrategies(context.Background()); err != nil {
			log.Error().Err(err).Msg("Failed initial strategies load")
		} else {
			log.Info().Msg("Strategies loaded successfully")
		}
	}()
}
