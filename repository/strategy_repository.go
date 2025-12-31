package repository

import (
	"backend/model"
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type StrategyRepository struct {
	collection *mongo.Collection
}

func NewStrategyRepository(db *mongo.Database) *StrategyRepository {
	return &StrategyRepository{
		collection: db.Collection("chartink_strategy"),
	}
}

func (r *StrategyRepository) Save(ctx context.Context, strategy model.Strategy) error {
	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": strategy.Name},
		bson.M{"$set": strategy},
		opts,
	)
	return err
}

func (r *StrategyRepository) FindById(ctx context.Context, name string) (*model.Strategy, error) {
	var strategy model.Strategy
	err := r.collection.FindOne(ctx, bson.M{"_id": name}).Decode(&strategy)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil // Consistent with our Optional pattern
		}
		return nil, err
	}
	return &strategy, nil
}

func (r *StrategyRepository) FindAll(ctx context.Context) ([]model.Strategy, error) {
	var strategies []model.Strategy
	cursor, err := r.collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &strategies); err != nil {
		return nil, err
	}

	if strategies == nil {
		return []model.Strategy{}, nil
	}
	return strategies, nil
}

func (r *StrategyRepository) DeleteById(ctx context.Context, name string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": name})
	return err
}
