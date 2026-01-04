package repository

import (
	"backend/database"
	"backend/model"
	"context"
	"fmt"

	"github.com/jinzhu/copier"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PriceActionRepo struct {
	database.GenericRepo[model.StockRecord]
}

func NewPriceActionRepo(db *mongo.Database) *PriceActionRepo {
	return &PriceActionRepo{
		GenericRepo: database.GenericRepo[model.StockRecord]{
			Collection: db.Collection(model.PACollectionName),
		},
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
	return r.GenericRepo.GetAll(ctx, nil)
}

func (r *PriceActionRepo) GetAllPAIn(ctx context.Context, ids []string) ([]model.StockRecord, error) {
	filter := bson.M{
		"_id": bson.M{"$in": ids},
	}
	return r.GenericRepo.GetAll(ctx, filter)
}

func (r *PriceActionRepo) GetPAByID(ctx context.Context, symbol string) (model.StockRecord, error) {
	res, err := r.GenericRepo.Get(ctx, symbol)
	if err != nil {
		return model.StockRecord{}, err
	}
	if res == nil {
		return model.StockRecord{}, fmt.Errorf("stock %s not found", symbol)
	}
	return *res, nil
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

	_, err := r.Collection.BulkWrite(ctx, []mongo.WriteModel{pull, push}, options.BulkWrite().SetOrdered(true))
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

	res, err := r.Collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("no %s record found for date %s", fieldName, req.Date)
	}
	return nil
}

func (r *PriceActionRepo) deleteNestedInfo(ctx context.Context, symbol, date, fieldName string) error {
	filter := bson.M{"_id": symbol}
	update := bson.M{"$pull": bson.M{fieldName: bson.M{"date": date}}}

	res, err := r.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.ModifiedCount == 0 {
		return fmt.Errorf("no %s record found to delete for date %s", fieldName, date)
	}
	return nil
}
