package repository

import (
	"backend/database"
	"backend/model"

	"go.mongodb.org/mongo-driver/mongo"
)

var (
	orderCollection = "zerodha_orders"
)

type OrderRepo struct {
	database.GenericRepo[model.Order]
}

func NewOrderRepo(db *mongo.Database) *OrderRepo {
	return &OrderRepo{
		GenericRepo: database.GenericRepo[model.Order]{
			Collection: db.Collection(orderCollection),
		},
	}
}
