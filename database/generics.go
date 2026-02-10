package database

import (
	"context"
	"fmt"
	"reflect"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// BulkUpdateItem represents a single update in a bulk operation
type BulkUpdateItem struct {
	ID     any
	Fields bson.M
}

// GenericRepo is a generic MongoDB repository for any type T
type GenericRepo[T any] struct {
	Collection *mongo.Collection
}

//
// -------------------- INSERT --------------------
//

// Insert adds a single document to the collection
func (r *GenericRepo[T]) Insert(ctx context.Context, doc T) error {
	_, err := r.Collection.InsertOne(ctx, doc)
	return err
}

//
// -------------------- GET --------------------
//

// Get retrieves a single document by ID
func (r *GenericRepo[T]) Get(ctx context.Context, id any) (*T, error) {
	var result T
	err := r.Collection.FindOne(ctx, bson.M{"_id": id}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

// GetAll retrieves all documents matching a filter
func (r *GenericRepo[T]) GetAll(ctx context.Context, filter bson.M) ([]T, error) {
	return r.GetAllSorted(ctx, filter, nil)
}

// GetAllSorted retrieves all documents matching a filter with sort options
func (r *GenericRepo[T]) GetAllSorted(ctx context.Context, filter bson.M, sort bson.M) ([]T, error) {
	if filter == nil {
		filter = bson.M{}
	}

	findOptions := options.Find()
	if sort != nil {
		findOptions.SetSort(sort)
	}

	cur, err := r.Collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var results []T
	if err := cur.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}

// FindByFilter retrieves all documents matching the given filter
func (r *GenericRepo[T]) FindByFilter(ctx context.Context, filter bson.M) ([]T, error) {
	if filter == nil {
		filter = bson.M{} // empty filter = fetch all
	}

	cur, err := r.Collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var results []T
	if err := cur.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}

// FindOneByFilter retrieves a single document matching the given filter.
// Returns nil if no document is found.
func (r *GenericRepo[T]) FindOneByFilter(ctx context.Context, filter bson.M) (*T, error) {
	if filter == nil {
		return nil, fmt.Errorf("filter cannot be nil")
	}

	var result T
	err := r.Collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // no document found
		}
		return nil, err
	}

	return &result, nil
}

//
// -------------------- UPDATE --------------------
//

// Update replaces the entire document by ID
func (r *GenericRepo[T]) Update(ctx context.Context, id any, doc T) error {
	_, err := r.Collection.ReplaceOne(ctx, bson.M{"_id": id}, doc)
	return err
}

// UpdateFields updates specific fields of the document with the given `id`.
// Returns the updated document.
func (r *GenericRepo[T]) UpdateFields(ctx context.Context, id any, fields bson.M) (*T, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updated T

	err := r.Collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": fields},
		opts,
	).Decode(&updated)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("document not found")
		}
		return nil, err
	}

	return &updated, nil
}

//
// -------------------- DELETE --------------------
//

// Delete removes a document by ID
func (r *GenericRepo[T]) Delete(ctx context.Context, id any) error {
	_, err := r.Collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// DeleteByIdNotIn deletes all documents whose _id is NOT in the provided ids slice
// Returns the number of deleted documents
func (r *GenericRepo[T]) DeleteByIdNotIn(ctx context.Context, ids []any) (int64, error) {
	if len(ids) == 0 {
		// Nothing to compare, optionally delete everything or return 0
		return 0, nil
	}

	filter := bson.M{
		"_id": bson.M{
			"$nin": ids,
		},
	}

	res, err := r.Collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}

	return res.DeletedCount, nil
}

//
// -------------------- SAVEALL (Bulk Upsert) --------------------
//

// SaveAll inserts or updates multiple documents in bulk
// idField is the struct field name that maps to _id in Mongo
func (r *GenericRepo[T]) SaveAll(ctx context.Context, items []T, idField string) error {
	if len(items) == 0 {
		return nil
	}

	models := make([]mongo.WriteModel, 0, len(items))

	for _, item := range items {
		val := reflect.ValueOf(item)
		if val.Kind() == reflect.Pointer {
			val = val.Elem()
		}

		id := val.FieldByName(idField)
		if !id.IsValid() {
			return fmt.Errorf("field %s not found in struct", idField)
		}

		model := mongo.NewUpdateOneModel().
			SetFilter(bson.M{"_id": id.Interface()}).
			SetUpdate(bson.M{"$set": item}).
			SetUpsert(true)

		models = append(models, model)
	}

	opts := options.BulkWrite().SetOrdered(false)
	_, err := r.Collection.BulkWrite(ctx, models, opts)
	return err
}
