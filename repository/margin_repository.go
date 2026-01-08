package repository

import (
	"backend/database"
	"backend/model"

	"go.mongodb.org/mongo-driver/mongo"
)

type MarginRepository struct {
	database.GenericRepo[model.Margin]
}

func NewMarginRepository(db *mongo.Database) *MarginRepository {
	return &MarginRepository{
		GenericRepo: database.GenericRepo[model.Margin]{
			Collection: db.Collection("margin"),
		},
	}
}
