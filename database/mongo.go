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

	// Set options with a shorter connection timeout for faster failure/startup
	clientOptions := options.Client().
		ApplyURI(uri).
		SetConnectTimeout(5 * time.Second).
		SetServerSelectionTimeout(5 * time.Second)

	// mongo.Connect doesn't block for connection by default, it just initializes the client.
	// We'll skip the manual Ping here to speed up startup. The first actual query
	// (fetching configs) will establish the connection.
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal().Msgf("Failed to initialize MongoDB client: %v", err)
	}

	fmt.Println("MongoDB client initialized (ShahbazTrades)")

	return client, client.Database("ShahbazTrades")
}
