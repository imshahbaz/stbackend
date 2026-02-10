package repository

import (
	"backend/database"
	"backend/model"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type StrategyRepository struct {
	Generic database.GenericRepo[model.Strategy]
}

func NewStrategyRepository(db *mongo.Database) *StrategyRepository {
	return &StrategyRepository{
		Generic: database.GenericRepo[model.Strategy]{
			Collection: db.Collection("chartink_strategy"),
		},
	}
}
