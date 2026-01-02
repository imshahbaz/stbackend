package repository

import (
	"backend/database"
	"backend/model"
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type StrategyRepository struct {
	database.GenericRepo[model.Strategy]
}

func NewStrategyRepository(db *mongo.Database) *StrategyRepository {
	return &StrategyRepository{
		GenericRepo: database.GenericRepo[model.Strategy]{
			Collection: db.Collection("chartink_strategy"),
		},
	}
}

func (r *StrategyRepository) Save(ctx context.Context, strategy model.Strategy) error {
	filter := bson.M{"_id": strategy.Name}
	_, err := r.UpdateSpecificFields(ctx, filter, strategy)
	return err
}

func (r *StrategyRepository) FindById(ctx context.Context, name string) (*model.Strategy, error) {
	return r.FindByID(ctx, name)
}

func (r *StrategyRepository) DeleteById(ctx context.Context, name string) error {
	filter := bson.M{"_id": name}
	_, err := r.DeleteByFilter(ctx, filter)
	return err
}
