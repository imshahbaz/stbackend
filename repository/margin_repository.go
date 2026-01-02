package repository

import (
	"backend/database"
	"backend/model"
	"context"

	"go.mongodb.org/mongo-driver/bson"
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

func (r *MarginRepository) FindBySymbol(ctx context.Context, symbol string) (*model.Margin, error) {
	return r.FindByID(ctx, symbol)
}

func (r *MarginRepository) SaveAllMargins(ctx context.Context, margins []model.Margin) error {
	return r.SaveAll(ctx, "Symbol", margins)
}

func (r *MarginRepository) Save(ctx context.Context, margin model.Margin) error {
	filter := bson.M{"_id": margin.Symbol}
	_, err := r.UpdateSpecificFields(ctx, filter, margin)
	return err
}

func (r *MarginRepository) DeleteByIdNotIn(ctx context.Context, ids []string) (int64, error) {
	filter := bson.M{"_id": bson.M{"$nin": ids}}
	return r.DeleteByFilter(ctx, filter)
}
