package service

import (
	"context"

	"backend/config"
	"backend/model"

	"github.com/rs/zerolog/log"
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
	m, c, m1, c1 := initMongoConfigs(isProduction, collection)

	return &ConfigServiceImpl{
		collection:     collection,
		configManager:  config.NewConfigManager(m1, c1),
		mongoId:        m,
		clientConfigId: c,
	}
}

func (s *ConfigServiceImpl) GetConfigManager() *config.ConfigManager {
	return s.configManager
}

func (s *ConfigServiceImpl) LoadMongoEnvConfig(ctx context.Context) error {
	val, err := s.FindMongoEnvConfig(ctx)
	if err != nil {
		log.Info().Msgf("Error Loading Mongo Configs: %v", err)
		return err
	}

	s.configManager.UpdateConfig(val)
	log.Info().Msg("Mongo Configs Loaded Successfully")
	return nil
}

func (s *ConfigServiceImpl) UpdateMongoEnvConfig(ctx context.Context, cfg model.MongoEnvConfig) error {
	filter := bson.M{"_id": s.mongoId}
	update := bson.M{"$set": cfg}

	_, err := s.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Info().Msgf("Error Updating Mongo Configs: %v", err)
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
		log.Info().Msgf("Error Loading Mongo Client Configs: %v", err)
		return err
	}

	s.configManager.UpdateClientConfig(val)
	log.Info().Msg("Mongo Client Configs Loaded Successfully")
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
		log.Info().Msgf("Error Updating Mongo Client Configs: %v", err)
		return err
	}

	return s.LoadClientConfig(ctx)
}

func initMongoConfigs(isProduction bool, collection *mongo.Collection) (string, string, *model.MongoEnvConfig, *model.ClientConfigs) {
	mongoId := "mongoConfigDev"
	if isProduction {
		mongoId = "mongoConfig"
	}

	clientConfigId := "clientConfigIdDev"
	if isProduction {
		clientConfigId = "clientConfigId"
	}

	idsToFetch := []string{mongoId, clientConfigId}

	// 2. Use $in operator to fetch multiple documents in one call
	cursor, err := collection.Find(context.Background(), bson.M{"_id": bson.M{"$in": idsToFetch}})
	if err != nil {
		log.Fatal().Msgf("Critical error: Could not query MongoDB: %v", err)
	}

	// 3. Decode results into a slice of maps
	var results []bson.M
	if err = cursor.All(context.Background(), &results); err != nil {
		log.Fatal().Msgf("Critical error: Could not decode results: %v", err)
	}

	// 4. Map the generic results back to your specific structs
	var mongoConfig model.MongoEnvConfig
	var clientConfig model.ClientConfigs

	for _, doc := range results {
		id := doc["_id"].(string)

		// Convert the map back to BSON bytes then into the specific struct
		bsonBytes, _ := bson.Marshal(doc)

		switch id {
		case mongoId:
			bson.Unmarshal(bsonBytes, &mongoConfig)
		case clientConfigId:
			bson.Unmarshal(bsonBytes, &clientConfig)
		}
	}

	// 5. Safety check to ensure both were found
	if mongoConfig.ID == "" || clientConfig.ID == "" {
		log.Fatal().Msg("Critical error: One or more config documents missing from MongoDB")
	}

	return mongoId, clientConfigId, &mongoConfig, &clientConfig
}
