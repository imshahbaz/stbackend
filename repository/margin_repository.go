package repository

import (
	"backend/model" // Adjust to your actual module path
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MarginRepository struct {
	collection *mongo.Collection
}

// NewMarginRepository acts like the @Repository bean initialization
func NewMarginRepository(db *mongo.Database) *MarginRepository {
	return &MarginRepository{
		collection: db.Collection("margin"),
	}
}

// DeleteByIdNotIn mirrors your @Query delete logic
func (r *MarginRepository) DeleteByIdNotIn(ctx context.Context, ids []string) (int64, error) {
	// Java: { '_id' : { '$nin' : ?0 } }
	// Go: bson.M is a map, "$nin" is the operator
	filter := bson.M{
		"_id": bson.M{
			"$nin": ids,
		},
	}

	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}

// Standard Save method (Replacement for repo.save())
func (r *MarginRepository) Save(ctx context.Context, margin model.Margin) error {
	_, err := r.collection.InsertOne(ctx, margin)
	return err
}

func (r *MarginRepository) FindAll(ctx context.Context) ([]model.Margin, error) {
	// Passing an empty filter bson.M{} fetches everything
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to execute find: %w", err)
	}
	defer cursor.Close(ctx)

	var margins []model.Margin
	// All() automatically iterates the cursor and decodes results into the slice
	if err := cursor.All(ctx, &margins); err != nil {
		return nil, fmt.Errorf("failed to decode margins: %w", err)
	}

	return margins, nil
}

// SaveAll mirrors repo.saveAll() using a bulk InsertMany operation
func (r *MarginRepository) SaveAll(ctx context.Context, margins []model.Margin) error {
	if len(margins) == 0 {
		return nil
	}

	// MongoDB's InsertMany requires a slice of []interface{} (or []any)
	docs := make([]any, len(margins))
	for i, m := range margins {
		docs[i] = m
	}

	_, err := r.collection.InsertMany(ctx, docs)
	if err != nil {
		return fmt.Errorf("failed to perform bulk insert: %w", err)
	}

	return nil
}
