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
	collection *mongo.Collection
}

func NewPriceActionRepo(db *mongo.Database) *PriceActionRepo {
	return &PriceActionRepo{
		collection: db.Collection(model.PACollectionName),
	}
}


func (r *PriceActionRepo) SaveOrderBlock(ctx context.Context, ob model.ObRequest) error {
	return r.saveNestedInfo(ctx, ob, "order_blocks")
}

func (r *PriceActionRepo) SaveFvg(ctx context.Context, ob model.ObRequest) error {
	return r.saveNestedInfo(ctx, ob, "fvg")
}

func (r *PriceActionRepo) UpdateOrderBlock(ctx context.Context, req model.ObRequest) error {
	return r.updateNestedInfo(ctx, req, "order_blocks")
}

func (r *PriceActionRepo) UpdateFvg(ctx context.Context, req model.ObRequest) error {
	return r.updateNestedInfo(ctx, req, "fvg")
}

func (r *PriceActionRepo) DeleteOrderBlockByDate(ctx context.Context, symbol, date string) error {
	return r.deleteNestedInfo(ctx, symbol, date, "order_blocks")
}

func (r *PriceActionRepo) DeleteFvgByDate(ctx context.Context, symbol, date string) error {
	return r.deleteNestedInfo(ctx, symbol, date, "fvg")
}


func (r *PriceActionRepo) GetAllPriceAction(ctx context.Context) ([]model.StockRecord, error) {
	return r.findMany(ctx, bson.M{})
}

func (r *PriceActionRepo) GetAllPAIn(ctx context.Context, ids []string) ([]model.StockRecord, error) {
	return r.findMany(ctx, bson.M{"_id": bson.M{"$in": ids}})
}

func (r *PriceActionRepo) GetPAByID(ctx context.Context, symbol string) (model.StockRecord, error) {
	var stock model.StockRecord
	err := r.collection.FindOne(ctx, bson.M{"_id": symbol}).Decode(&stock)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return model.StockRecord{}, fmt.Errorf("stock %s not found", symbol)
		}
		return model.StockRecord{}, err
	}
	return stock, nil
}


func (r *PriceActionRepo) saveNestedInfo(ctx context.Context, req model.ObRequest, fieldName string) error {
	var newInfo model.Info
	copier.Copy(&newInfo, &req)

	pull := mongo.NewUpdateOneModel().
		SetFilter(bson.M{"_id": req.Symbol}).
		SetUpdate(bson.M{"$pull": bson.M{fieldName: bson.M{"date": req.Date}}})

	push := mongo.NewUpdateOneModel().
		SetFilter(bson.M{"_id": req.Symbol}).
		SetUpdate(bson.M{
			"$push": bson.M{
				fieldName: bson.M{
					"$each": []model.Info{newInfo},
					"$sort": bson.M{"date": -1},
				},
			},
		}).SetUpsert(true)

	_, err := r.collection.BulkWrite(ctx, []mongo.WriteModel{pull, push}, options.BulkWrite().SetOrdered(true))
	return err
}

func (r *PriceActionRepo) updateNestedInfo(ctx context.Context, req model.ObRequest, fieldName string) error {
	filter := bson.M{"_id": req.Symbol, fieldName + ".date": req.Date}
	update := bson.M{
		"$set": bson.M{
			fieldName + ".$[elem].high": req.High,
			fieldName + ".$[elem].low":  req.Low,
		},
	}
	opts := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []any{bson.M{"elem.date": req.Date}},
	})

	res, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}
	if res.ModifiedCount == 0 {
		return fmt.Errorf("no %s record found for date %s", fieldName, req.Date)
	}
	return nil
}

func (r *PriceActionRepo) deleteNestedInfo(ctx context.Context, symbol, date, fieldName string) error {
	filter := bson.M{"_id": symbol}
	update := bson.M{"$pull": bson.M{fieldName: bson.M{"date": date}}}

	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.ModifiedCount == 0 {
		return fmt.Errorf("no %s record found to delete for date %s", fieldName, date)
	}
	return nil
}

func (r *PriceActionRepo) findMany(ctx context.Context, filter bson.M) ([]model.StockRecord, error) {
	var results []model.StockRecord
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	if results == nil {
		return []model.StockRecord{}, nil
	}
	return results, nil
}
