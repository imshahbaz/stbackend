package repository

import (
	"backend/model"
	"context"
	"fmt"

	"github.com/jinzhu/copier"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PriceActionRepo struct {
	obCollection *mongo.Collection
}

func NewPriceActionRepo(db *mongo.Database) *PriceActionRepo {
	return &PriceActionRepo{
		obCollection: db.Collection(model.ObCollectionName),
	}
}

func (r *PriceActionRepo) SaveOrderBlock(ctx context.Context, ob model.ObRequest) error {
	var newOB model.OBInfo
	copier.Copy(&newOB, &ob)

	pullOp := mongo.NewUpdateOneModel().
		SetFilter(bson.M{"_id": ob.Symbol}).
		SetUpdate(bson.M{
			"$pull": bson.M{
				"order_blocks": bson.M{"date": ob.Date},
			},
		})

	pushOp := mongo.NewUpdateOneModel().
		SetFilter(bson.M{"_id": ob.Symbol}).
		SetUpdate(bson.M{
			"$push": bson.M{
				"order_blocks": bson.M{
					"$each": []model.OBInfo{newOB},
					"$sort": bson.M{"date": -1},
				},
			},
		}).
		SetUpsert(true)

	opts := options.BulkWrite().SetOrdered(true)
	_, err := r.obCollection.BulkWrite(ctx, []mongo.WriteModel{pullOp, pushOp}, opts)

	return err
}

func (r *PriceActionRepo) GetAllOrderBlock(ctx context.Context) ([]model.StockRecord, error) {
	var results []model.StockRecord

	cursor, err := r.obCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}

func (r *PriceActionRepo) GetAllObIn(ctx context.Context, ids []string) ([]model.StockRecord, error) {
	var stocks []model.StockRecord

	filter := bson.M{
		"_id": bson.M{"$in": ids},
	}

	cursor, err := r.obCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &stocks); err != nil {
		return nil, err
	}

	if stocks == nil {
		stocks = []model.StockRecord{}
	}

	return stocks, nil
}

func (r *PriceActionRepo) GetObByID(ctx context.Context, symbol string) (model.StockRecord, error) {
	var stock model.StockRecord

	filter := bson.M{"_id": symbol}

	err := r.obCollection.FindOne(ctx, filter).Decode(&stock)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return model.StockRecord{}, fmt.Errorf("stock %s not found in cache", symbol)
		}
		return model.StockRecord{}, err
	}

	return stock, nil
}

func (r *PriceActionRepo) DeleteOrderBlockByDate(ctx context.Context, symbol string, date string) error {
	filter := bson.M{"_id": symbol}
	update := bson.M{
		"$pull": bson.M{
			"order_blocks": bson.M{"date": date},
		},
	}

	result, err := r.obCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.ModifiedCount == 0 {
		return fmt.Errorf("no block found for date %s in %s", date, symbol)
	}

	return nil
}
