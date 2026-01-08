package main

import (
	"backend/config"
	"backend/database"
	_ "backend/docs"
	"backend/routes"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bytedance/sonic"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	sysConfigs, err := config.LoadConfigs()
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading configuration")
	}

	mongoClient, db := database.InitMongoClient(sysConfigs)

	router := routes.SetupRouter(db, sysConfigs)

	port := sysConfigs.Config.Port
	if port == "" {
		port = "8080"
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		log.Info().Msgf("Server starting on port %s", port)
		if err := router.Listen("0.0.0.0:" + port); err != nil {
			log.Info().Msgf("Server listener stopped: %v", err)
		}
	}()

	<-quit
	log.Info().Msg("Shutdown signal received, initiating graceful shutdown...")

	shutdownTimeout := 10 * time.Second
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := router.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Error during server shutdown")
	} else {
		log.Info().Msg("Server shut down successfully")
	}

	if mongoClient != nil {
		if err := mongoClient.Disconnect(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("Error disconnecting from MongoDB")
		} else {
			log.Info().Msg("MongoDB connection closed safely")
		}
	}

	log.Info().Msg("Server exited successfully")
}

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.With().Logger()
	zerolog.InterfaceMarshalFunc = sonic.Marshal
}
