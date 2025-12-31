package database

import (
	"backend/customerrors"
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func UpdateGeneric[T any](ctx context.Context, collection *mongo.Collection, filter bson.M, data interface{}) (*T, error) {
	update := bson.M{
		"$set": data,
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updatedDoc T
	err := collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedDoc)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, customerrors.ErrUserNotFound
		}
		return nil, err
	}

	return &updatedDoc, nil
}
