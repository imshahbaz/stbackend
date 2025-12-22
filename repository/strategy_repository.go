package repository

import (
	"backend/model"
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type StrategyRepository struct {
	collection *mongo.Collection
}

// NewStrategyRepository initializes the repository with the correct collection name
func NewStrategyRepository(db *mongo.Database) *StrategyRepository {
	return &StrategyRepository{
		collection: db.Collection("chartink_strategy"),
	}
}

// Save handles both Insert and Update (Equivalent to repo.save())
func (r *StrategyRepository) Save(ctx context.Context, strategy model.Strategy) error {
	filter := bson.M{"_id": strategy.Name}
	update := bson.M{"$set": strategy}

	// opts.SetUpsert(true) makes it behave like Spring's .save()
	// (Update if exists, Insert if not)
	opts := options.Update().SetUpsert(true)

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// FindById (Equivalent to repo.findById())
func (r *StrategyRepository) FindById(ctx context.Context, name string) (*model.Strategy, error) {
	var strategy model.Strategy
	err := r.collection.FindOne(ctx, bson.M{"_id": name}).Decode(&strategy)
	if err != nil {
		return nil, err
	}
	return &strategy, nil
}

// FindAll (Equivalent to repo.findAll())
func (r *StrategyRepository) FindAll(ctx context.Context) ([]model.Strategy, error) {
	var strategies []model.Strategy
	cursor, err := r.collection.Find(ctx, bson.D{}) // Empty filter gets all
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &strategies); err != nil {
		return nil, err
	}
	return strategies, nil
}

// DeleteById (Equivalent to repo.deleteById())
func (r *StrategyRepository) DeleteById(ctx context.Context, name string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": name})
	return err
}
