package database

import (
	"backend/customerrors"
	"context"
	"errors"
	"reflect"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func UpdateGeneric[T any](ctx context.Context, collection *mongo.Collection, filter bson.M, data any) (*T, error) {
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

func UpdateSpecificFields(collection *mongo.Collection, filter bson.M, data any) (*mongo.UpdateResult, error) {
	updateData := bson.M{}

	val := reflect.ValueOf(data)
	typ := reflect.TypeOf(data)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = typ.Elem()
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		tag := fieldType.Tag.Get("bson")
		if tag == "" || tag == "-" {
			continue
		}

		if !field.IsZero() {
			updateData[tag] = field.Interface()
		}
	}

	update := bson.M{"$set": updateData}
	return collection.UpdateOne(context.TODO(), filter, update)
}
