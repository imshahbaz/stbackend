package service

import (
	"backend/config"
	"backend/model"
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ConfigService interface {
	GetConfigManager() *config.ConfigManager
	LoadMongoEnvConfig(ctx *gin.Context)
	UpdateMongoEnvConfig(ctx *gin.Context, cfg model.MongoEnvConfig)
	FindMongoEnvConfig(ctx *gin.Context) (*model.MongoEnvConfig, error)
	GetActiveMongoEnvConfig(ctx *gin.Context)
}

type ConfigServiceImpl struct {
	collection    *mongo.Collection
	configManager *config.ConfigManager
	mongoId       string
}

func NewConfigService(db *mongo.Database, mongoId string) ConfigService {
	collection := db.Collection("configs")
	var mongoConfig model.MongoEnvConfig
	err := collection.FindOne(context.TODO(), bson.M{"_id": mongoId}).Decode(&mongoConfig)
	if err != nil {
		log.Panicf("Critical error: Could not load initial config from MongoDB")
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

func (s *ConfigServiceImpl) LoadMongoEnvConfig(ctx *gin.Context) {
	val, err := s.FindMongoEnvConfig(ctx)
	if err != nil {
		log.Printf("Error Loading Mongo Configs")
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

func (s *ConfigServiceImpl) UpdateMongoEnvConfig(ctx *gin.Context, cfg model.MongoEnvConfig) {
	filter := bson.M{"_id": s.mongoId}
	update := bson.M{"$set": cfg}

	_, err := s.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("Error Updating Mongo Configs")
		ctx.JSON(http.StatusInternalServerError, model.Response{
			Success: false,
			Error:   "Error Updating Mongo Configs",
		})
		return
	}

	s.LoadMongoEnvConfig(ctx)
}

func (s *ConfigServiceImpl) FindMongoEnvConfig(ctx *gin.Context) (*model.MongoEnvConfig, error) {
	var config model.MongoEnvConfig
	err := s.collection.FindOne(ctx, bson.M{"_id": s.mongoId}).Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (s *ConfigServiceImpl) GetActiveMongoEnvConfig(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, s.configManager.GetConfig())
}
