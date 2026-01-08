package main

import (
	"backend/config"
	"backend/database"
	_ "backend/docs"
	"backend/routes"
	"runtime"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	sysConfigs, err := config.LoadConfigs()
	if err != nil {
		log.Fatal().Msgf("Error loading configuration: %v", err)
	}

	_, db := database.InitMongoClient(sysConfigs)

	router := routes.SetupRouter(db, sysConfigs)

	port := sysConfigs.Config.Port
	if port == "" {
		port = "8080"
	}

	log.Info().Msgf("Server starting on port %s", port)
	if err := router.Listen("0.0.0.0:" + port); err != nil {
		log.Fatal().Msgf("Server failed to start: %v", err)
	}
}

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.With().Logger()
}
