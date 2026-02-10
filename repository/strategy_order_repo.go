package repository

import (
	"backend/database"
	"backend/model"
	"backend/util"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type StrategyOrderRepository struct {
	Generic database.GenericRepo[model.StrategyOrder]
}

func NewStrategyOrderRepository(db *mongo.Database) *StrategyOrderRepository {
	return &StrategyOrderRepository{
		Generic: database.GenericRepo[model.StrategyOrder]{
			Collection: db.Collection("strategy_orders"),
		},
	}
}

func (r *StrategyOrderRepository) FindByStrategyAndTime(ctx context.Context, strategyName string, startTime, endTime time.Time) ([]model.StrategyOrder, error) {
	filter := bson.M{
		"strategyName": strategyName,
		"date": bson.M{
			"$gte": startTime,
			"$lte": endTime,
		},
	}
	return r.Generic.GetAll(ctx, filter)
}

func (r *StrategyOrderRepository) FindTodayOrdersByStrategy(ctx context.Context, strategyName string) ([]model.StrategyOrder, error) {
	startTime := util.GetISTMidnight()
	endTime := startTime.Add(24 * time.Hour)

	return r.FindByStrategyAndTime(ctx, strategyName, startTime, endTime)
}
