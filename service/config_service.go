package service

import (
	"context"
	"log"
	"net/http"

	"backend/config"
	"backend/model"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ConfigService interface {
	GetConfigManager() *config.ConfigManager
	LoadMongoEnvConfig(ctx *gin.Context)
	UpdateMongoEnvConfig(ctx *gin.Context, cfg model.MongoEnvConfig)
	FindMongoEnvConfig(ctx context.Context) (*model.MongoEnvConfig, error)
	GetActiveMongoEnvConfig(ctx *gin.Context)
}

type ConfigServiceImpl struct {
	collection    *mongo.Collection
	configManager *config.ConfigManager
	mongoId       string
}

func NewConfigService(db *mongo.Database, mongoId string) ConfigService {
	collection := db.Collection("configs")

	// Initial boot-up load
	var mongoConfig model.MongoEnvConfig
	err := collection.FindOne(context.Background(), bson.M{"_id": mongoId}).Decode(&mongoConfig)
	if err != nil {
		log.Panicf("Critical error: Could not load initial config from MongoDB: %v", err)
	}

	return &ConfigServiceImpl{
		collection:    collection,
		configManager: config.NewConfigManager(&mongoConfig),
		mongoId:       mongoId,
	}
}

func (s *ConfigServiceImpl) GetConfigManager() *config.ConfigManager {
	return s.configManager
}

// LoadMongoEnvConfig refreshes the in-memory ConfigManager from the Database
func (s *ConfigServiceImpl) LoadMongoEnvConfig(ctx *gin.Context) {
	val, err := s.FindMongoEnvConfig(ctx.Request.Context())
	if err != nil {
		log.Printf("Error Loading Mongo Configs: %v", err)
		ctx.JSON(http.StatusInternalServerError, model.Response{
			Success: false,
			Error:   "Error Loading Mongo Configs",
		})
		return
	}

	s.configManager.UpdateConfig(val)

	log.Printf("Mongo Configs Loaded Successfully")
	ctx.JSON(http.StatusOK, model.Response{
		Success: true,
		Message: "Mongo Configs Loaded Successfully",
	})
}

// UpdateMongoEnvConfig updates the DB and then reloads the ConfigManager
func (s *ConfigServiceImpl) UpdateMongoEnvConfig(ctx *gin.Context, cfg model.MongoEnvConfig) {
	filter := bson.M{"_id": s.mongoId}
	update := bson.M{"$set": cfg}

	_, err := s.collection.UpdateOne(ctx.Request.Context(), filter, update)
	if err != nil {
		log.Printf("Error Updating Mongo Configs: %v", err)
		ctx.JSON(http.StatusInternalServerError, model.Response{
			Success: false,
			Error:   "Error Updating Mongo Configs",
		})
		return
	}

	// Trigger reload to sync in-memory ConfigManager
	s.LoadMongoEnvConfig(ctx)
}

// FindMongoEnvConfig is a pure data fetcher (Decoupled from Gin)
func (s *ConfigServiceImpl) FindMongoEnvConfig(ctx context.Context) (*model.MongoEnvConfig, error) {
	var cfg model.MongoEnvConfig
	err := s.collection.FindOne(ctx, bson.M{"_id": s.mongoId}).Decode(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// GetActiveMongoEnvConfig returns the current in-memory configuration
func (s *ConfigServiceImpl) GetActiveMongoEnvConfig(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, s.configManager.GetConfig())
}
