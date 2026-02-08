package main

import (
	"backend/config"
	"backend/di"
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	sysConfigs, err := config.LoadConfigs()
	if err != nil {
		log.Fatal().Msgf("Error loading configuration: %v", err)
	}

	if sysConfigs.Config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	app, cleanup, err := di.InitializeApp(sysConfigs)
	if err != nil {
		log.Fatal().Msgf("Error initializing application: %v", err)
	}

	app.Start()

	port := sysConfigs.Config.Port
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: app.Router,
	}

	go func() {
		log.Info().Msgf("Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Msgf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Msgf("Server forced to shutdown: %v", err)
	}

	cleanup()

	log.Info().Msg("Server exiting")
}

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.With().Logger()
}
