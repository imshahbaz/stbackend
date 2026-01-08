package database

import (
	"backend/config"
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func InitMongoClient(sysConfigs *config.SystemConfigs) (*mongo.Client, *mongo.Database) {
	rawString := "mongodb+srv://%s:%s@jaguartrading.ptkr6fq.mongodb.net/ShahbazTrades"
	uri := fmt.Sprintf(rawString,
		sysConfigs.Config.MongoUser,
		sysConfigs.Config.MongoPassword,
	)

	clientOptions := options.Client().ApplyURI(uri)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal().Msgf("Failed to connect to MongoDB: %v", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal().Msgf("Could not ping MongoDB: %v", err)
	}

	fmt.Println("Successfully connected to MongoDB (ShahbazTrades)")

	return client, client.Database("ShahbazTrades")
}
