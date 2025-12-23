package main

import (
	"backend/config"
	"backend/database"
	_ "backend/docs"
	"backend/routes"
	"log"

	"github.com/gin-gonic/gin"
)

// @title           Trades Management API
// @version         1.0
// @description     This is a specialized server for managing trading strategies and margins.
// @BasePath  /api

//go:generate swag init
func main() {
	// Initialize the config
	sysConfigs, err := config.LoadConfigs()
	if err != nil {
		log.Fatal("Error loading configuration: ", err)
	}

	if sysConfigs.Config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	_, db := database.InitMongoClient(sysConfigs)

	// 3. Setup Router & Initialize all Services (Clean delegation)
	router := routes.SetupRouter(db, sysConfigs)

	// 4. Start Server
	port := sysConfigs.Config.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run("0.0.0.0:" + port); err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}
