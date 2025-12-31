package service

import (
	"context"
	"log"

	"backend/config"
	"backend/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ConfigService interface {
	GetConfigManager() *config.ConfigManager
	LoadMongoEnvConfig(ctx context.Context) error
	UpdateMongoEnvConfig(ctx context.Context, cfg model.MongoEnvConfig) error
	FindMongoEnvConfig(ctx context.Context) (*model.MongoEnvConfig, error)
	GetActiveMongoEnvConfig() model.MongoEnvConfig
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
func (s *ConfigServiceImpl) LoadMongoEnvConfig(ctx context.Context) error {
	val, err := s.FindMongoEnvConfig(ctx)
	if err != nil {
		log.Printf("Error Loading Mongo Configs: %v", err)
		return err
	}

	s.configManager.UpdateConfig(val)
	log.Printf("Mongo Configs Loaded Successfully")
	return nil
}

// UpdateMongoEnvConfig updates the DB and then reloads the ConfigManager
func (s *ConfigServiceImpl) UpdateMongoEnvConfig(ctx context.Context, cfg model.MongoEnvConfig) error {
	filter := bson.M{"_id": s.mongoId}
	update := bson.M{"$set": cfg}

	_, err := s.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("Error Updating Mongo Configs: %v", err)
		return err
	}

	// Trigger reload to sync in-memory ConfigManager
	return s.LoadMongoEnvConfig(ctx)
}

// FindMongoEnvConfig is a pure data fetcher
func (s *ConfigServiceImpl) FindMongoEnvConfig(ctx context.Context) (*model.MongoEnvConfig, error) {
	var cfg model.MongoEnvConfig
	err := s.collection.FindOne(ctx, bson.M{"_id": s.mongoId}).Decode(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// GetActiveMongoEnvConfig returns the current in-memory configuration
func (s *ConfigServiceImpl) GetActiveMongoEnvConfig() model.MongoEnvConfig {
	return *s.configManager.GetConfig()
}
