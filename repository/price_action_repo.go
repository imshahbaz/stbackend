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
	priceActionCollection *mongo.Collection
}

func NewPriceActionRepo(db *mongo.Database) *PriceActionRepo {
	return &PriceActionRepo{
		priceActionCollection: db.Collection(model.PACollectionName),
	}
}

func (r *PriceActionRepo) SaveOrderBlock(ctx context.Context, ob model.ObRequest) error {
	var newOB model.Info
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
					"$each": []model.Info{newOB},
					"$sort": bson.M{"date": -1},
				},
			},
		}).
		SetUpsert(true)

	opts := options.BulkWrite().SetOrdered(true)
	_, err := r.priceActionCollection.BulkWrite(ctx, []mongo.WriteModel{pullOp, pushOp}, opts)

	return err
}

func (r *PriceActionRepo) GetAllOrderBlock(ctx context.Context) ([]model.StockRecord, error) {
	var results []model.StockRecord

	cursor, err := r.priceActionCollection.Find(ctx, bson.M{})
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

	cursor, err := r.priceActionCollection.Find(ctx, filter)
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

	err := r.priceActionCollection.FindOne(ctx, filter).Decode(&stock)
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

	result, err := r.priceActionCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.ModifiedCount == 0 {
		return fmt.Errorf("no block found for date %s in %s", date, symbol)
	}

	return nil
}

func (r *PriceActionRepo) UpdateOrderBlock(ctx context.Context, updateData model.ObRequest) error {
	filter := bson.M{"_id": updateData.Symbol, "order_blocks.date": updateData.Date}

	// $[elem] is a placeholder for the element that matches our arrayFilter
	update := bson.M{
		"$set": bson.M{
			"order_blocks.$[elem].high": updateData.High,
			"order_blocks.$[elem].low":  updateData.Low,
		},
	}

	// This filter tells MongoDB which specific array element to update
	options := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []any{bson.M{"elem.date": updateData.Date}},
	})

	result, err := r.priceActionCollection.UpdateOne(ctx, filter, update, options)
	if err != nil {
		return err
	}

	if result.ModifiedCount == 0 {
		return fmt.Errorf("no order block found for date %s", updateData.Date)
	}

	return nil
}
