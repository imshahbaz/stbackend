package repository

import (
	"backend/database"
	"backend/model"

	"go.mongodb.org/mongo-driver/mongo"
)

type OptionRepository struct {
	database.GenericRepo[model.OptionChain]
}

func NewOptionRepository(db *mongo.Database) *OptionRepository {
	return &OptionRepository{
		GenericRepo: database.GenericRepo[model.OptionChain]{
			Collection: db.Collection("options"),
		},
	}
}
