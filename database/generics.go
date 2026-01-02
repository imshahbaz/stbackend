package database

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type GenericRepo[T any] struct {
	Collection *mongo.Collection
}

func (r *GenericRepo[T]) UpdateGeneric(ctx context.Context, filter bson.M, data any) (*T, error) {
	update := bson.M{
		"$set": data,
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updatedDoc T
	err := r.Collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedDoc)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("document not found")
		}
		return nil, err
	}

	return &updatedDoc, nil
}

func (r *GenericRepo[T]) UpdateSpecificFields(ctx context.Context, filter bson.M, data any) (*mongo.UpdateResult, error) {
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

		cleanTag := strings.Split(tag, ",")[0]
		if cleanTag == "_id" {
			continue
		}

		if !field.IsZero() {
			updateData[cleanTag] = field.Interface()
		}
	}

	update := bson.M{"$set": updateData}
	return r.Collection.UpdateOne(ctx, filter, update)
}

func (r *GenericRepo[T]) FindByID(ctx context.Context, id any) (*T, error) {
	var result T
	err := r.Collection.FindOne(ctx, bson.M{"_id": id}).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

func (r *GenericRepo[T]) FindAll(ctx context.Context) ([]T, error) {
	var results []T
	cursor, err := r.Collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	err = cursor.All(ctx, &results)
	return results, err
}

func (r *GenericRepo[T]) SaveAll(ctx context.Context, idFieldName string, items []T) error {
	if len(items) == 0 {
		return nil
	}

	models := make([]mongo.WriteModel, len(items))
	for i, item := range items {
		val := reflect.ValueOf(item)
		if val.Kind() == reflect.Pointer {
			val = val.Elem()
		}

		idVal := val.FieldByName(idFieldName).Interface()

		models[i] = mongo.NewUpdateOneModel().
			SetFilter(bson.M{"_id": idVal}).
			SetUpdate(bson.M{"$set": item}).
			SetUpsert(true)
	}

	opts := options.BulkWrite().SetOrdered(false)
	_, err := r.Collection.BulkWrite(ctx, models, opts)
	return err
}

func (r *GenericRepo[T]) DeleteByFilter(ctx context.Context, filter bson.M) (int64, error) {
	res, err := r.Collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}

func (r *GenericRepo[T]) FindAllByIDs(ctx context.Context, ids any) ([]T, error) {
	var results []T
	cursor, err := r.Collection.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	if results == nil {
		return []T{}, nil
	}
	return results, nil
}
