package repository

import (
	"backend/model"
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MarginRepository struct {
	collection *mongo.Collection
}

func NewMarginRepository(db *mongo.Database) *MarginRepository {
	return &MarginRepository{
		collection: db.Collection("margin"),
	}
}


func (r *MarginRepository) FindAll(ctx context.Context) ([]model.Margin, error) {
	var margins []model.Margin
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to execute find: %w", err)
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &margins); err != nil {
		return nil, fmt.Errorf("failed to decode margins: %w", err)
	}

	if margins == nil {
		return []model.Margin{}, nil
	}
	return margins, nil
}

func (r *MarginRepository) FindBySymbol(ctx context.Context, symbol string) (*model.Margin, error) {
	var margin model.Margin
	err := r.collection.FindOne(ctx, bson.M{"_id": symbol}).Decode(&margin)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &margin, nil
}


func (r *MarginRepository) SaveAll(ctx context.Context, margins []model.Margin) error {
	if len(margins) == 0 {
		return nil
	}

	models := make([]mongo.WriteModel, len(margins))
	for i, m := range margins {
		models[i] = mongo.NewUpdateOneModel().
			SetFilter(bson.M{"_id": m.Symbol}).
			SetUpdate(bson.M{"$set": m}).
			SetUpsert(true)
	}

	opts := options.BulkWrite().SetOrdered(false)
	_, err := r.collection.BulkWrite(ctx, models, opts)
	if err != nil {
		return fmt.Errorf("failed to perform bulk upsert: %w", err)
	}

	return nil
}

func (r *MarginRepository) Save(ctx context.Context, margin model.Margin) error {
	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": margin.Symbol},
		bson.M{"$set": margin},
		opts,
	)
	return err
}


func (r *MarginRepository) DeleteByIdNotIn(ctx context.Context, ids []string) (int64, error) {
	filter := bson.M{"_id": bson.M{"$nin": ids}}

	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}
