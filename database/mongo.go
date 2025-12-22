package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"backend/config"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InitMongoClient replaces your MongoConfig class
func InitMongoClient(sysConfigs *config.SystemConfigs) (*mongo.Client, *mongo.Database) {
	// 1. Format the connection string
	rawString := "mongodb+srv://%s:%s@jaguartrading.ptkr6fq.mongodb.net/ShahbazTrades"
	uri := fmt.Sprintf(rawString,
		sysConfigs.Config.MongoUser,
		sysConfigs.Config.MongoPassword,
	)

	// 2. Set client options
	clientOptions := options.Client().ApplyURI(uri)

	// 3. Connect to MongoDB (Context is used for timeouts in Go)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB: ", err)
	}

	// 4. Check the connection (Ping)
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("Could not ping MongoDB: ", err)
	}

	fmt.Println("Successfully connected to MongoDB (ShahbazTrades)")

	// Return both the client and the specific database instance
	return client, client.Database("ShahbazTrades")
}
