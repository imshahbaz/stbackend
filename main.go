package main

import (
	"backend/config"
	"backend/database"
	_ "backend/docs"
	"backend/routes"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	sysConfigs, err := config.LoadConfigs()
	if err != nil {
		log.Fatal("Error loading configuration: ", err)
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
		log.Fatal("Server failed to start: ", err)
	}
}
