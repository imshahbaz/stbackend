package main

import (
	"backend/config"
	"backend/database"
	_ "backend/docs"
	"backend/routes"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	sysConfigs, err := config.LoadConfigs()
	if err != nil {
		log.Fatal().AnErr("Error loading configuration: ", err)
	}

	if sysConfigs.Config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	_, db := database.InitMongoClient(sysConfigs)

	router := routes.SetupRouter(db, sysConfigs)

	port := sysConfigs.Config.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run("0.0.0.0:" + port); err != nil {
		log.Fatal().AnErr("Server failed to start: ", err)
	}
}

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.With().Logger()
}
