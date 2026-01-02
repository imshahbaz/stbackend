package repository

import (
	"context"

	"backend/database"
	"backend/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserRepository struct {
	database.GenericRepo[model.User]
	counterCollection *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{
		GenericRepo: database.GenericRepo[model.User]{
			Collection: db.Collection("users"),
		},
		counterCollection: db.Collection("counters"),
	}
}

func (r *UserRepository) Save(ctx context.Context, user *model.User) error {
	filter := bson.M{"_id": user.UserID}
	_, err := r.UpdateSpecificFields(ctx, filter, user)
	return err
}

func (r *UserRepository) FindOne(ctx context.Context, filter bson.M) (*model.User, error) {
	var user model.User
	err := r.Collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetNextSequence(ctx context.Context, sequenceName string) (int, error) {
	filter := bson.M{"_id": sequenceName}
	update := bson.M{"$inc": bson.M{"seq": 1}}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(true)

	var result struct {
		Seq int `bson:"seq"`
	}

	err := r.counterCollection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&result)
	if err != nil {
		return 0, err
	}

	return result.Seq, nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, filter bson.M, data any) (*model.User, error) {
	return r.UpdateGeneric(ctx, filter, data)
}

func (r *UserRepository) PatchUser(ctx context.Context, filter bson.M, data any) error {
	_, err := r.UpdateSpecificFields(ctx, filter, data)
	return err
}
