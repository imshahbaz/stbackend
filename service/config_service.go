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

	//backend
	LoadMongoEnvConfig(ctx context.Context) error
	UpdateMongoEnvConfig(ctx context.Context, cfg model.MongoEnvConfig) error
	FindMongoEnvConfig(ctx context.Context) (*model.MongoEnvConfig, error)
	GetActiveMongoEnvConfig() model.MongoEnvConfig

	//client
	LoadClientConfig(ctx context.Context) error
	FindMongoClientConfig(ctx context.Context) (*model.ClientConfigs, error)
	GetActiveMongoClientConfig() model.ClientConfigs
	UpdateMongoClientConfig(ctx context.Context, cfg model.ClientConfigs) error
}

type ConfigServiceImpl struct {
	collection     *mongo.Collection
	configManager  *config.ConfigManager
	mongoId        string
	clientConfigId string
}

func NewConfigService(db *mongo.Database, isProduction bool) ConfigService {
	collection := db.Collection("configs")
	mongoId := "mongoConfigDev"
	if isProduction {
		mongoId = "mongoConfig"
	}
	var mongoConfig model.MongoEnvConfig
	err := collection.FindOne(context.Background(), bson.M{"_id": mongoId}).Decode(&mongoConfig)
	if err != nil {
		log.Panicf("Critical error: Could not load initial config from MongoDB: %v", err)
	}

	clientConfigId := "clientConfigIdDev"
	if isProduction {
		clientConfigId = "clientConfigId"
	}

	var clientConfig model.ClientConfigs
	err = collection.FindOne(context.Background(), bson.M{"_id": clientConfigId}).Decode(&clientConfig)
	if err != nil {
		log.Panicf("Critical error: Could not load initial client config from MongoDB: %v", err)
	}

	return &ConfigServiceImpl{
		collection:     collection,
		configManager:  config.NewConfigManager(&mongoConfig, &clientConfig),
		mongoId:        mongoId,
		clientConfigId: clientConfigId,
	}
}

func (s *ConfigServiceImpl) GetConfigManager() *config.ConfigManager {
	return s.configManager
}

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

func (s *ConfigServiceImpl) UpdateMongoEnvConfig(ctx context.Context, cfg model.MongoEnvConfig) error {
	filter := bson.M{"_id": s.mongoId}
	update := bson.M{"$set": cfg}

	_, err := s.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("Error Updating Mongo Configs: %v", err)
		return err
	}

	return s.LoadMongoEnvConfig(ctx)
}

func (s *ConfigServiceImpl) FindMongoEnvConfig(ctx context.Context) (*model.MongoEnvConfig, error) {
	var cfg model.MongoEnvConfig
	err := s.collection.FindOne(ctx, bson.M{"_id": s.mongoId}).Decode(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *ConfigServiceImpl) GetActiveMongoEnvConfig() model.MongoEnvConfig {
	return *s.configManager.GetConfig()
}

func (s *ConfigServiceImpl) LoadClientConfig(ctx context.Context) error {
	val, err := s.FindMongoClientConfig(ctx)
	if err != nil {
		log.Printf("Error Loading Mongo Client Configs: %v", err)
		return err
	}

	s.configManager.UpdateClientConfig(val)
	log.Printf("Mongo Client Configs Loaded Successfully")
	return nil
}

func (s *ConfigServiceImpl) FindMongoClientConfig(ctx context.Context) (*model.ClientConfigs, error) {
	var cfg model.ClientConfigs
	err := s.collection.FindOne(ctx, bson.M{"_id": s.clientConfigId}).Decode(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *ConfigServiceImpl) GetActiveMongoClientConfig() model.ClientConfigs {
	return *s.configManager.GetClientConfig()
}

func (s *ConfigServiceImpl) UpdateMongoClientConfig(ctx context.Context, cfg model.ClientConfigs) error {
	filter := bson.M{"_id": s.clientConfigId}
	update := bson.M{"$set": cfg}

	_, err := s.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("Error Updating Mongo Client Configs: %v", err)
		return err
	}

	return s.LoadClientConfig(ctx)
}
