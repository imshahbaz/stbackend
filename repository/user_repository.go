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
	collection        *mongo.Collection
	counterCollection *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{
		collection:        db.Collection("users"),
		counterCollection: db.Collection("counters"),
	}
}

func (r *UserRepository) Save(ctx context.Context, user *model.User) error {
	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": user.UserID},
		bson.M{"$set": user},
		opts,
	)
	return err
}

func (r *UserRepository) FindOne(ctx context.Context, filter bson.M) (*model.User, error) {
	var user model.User
	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserRepository) GetNextSequence(ctx context.Context, sequenceName string) (int, error) {
	filter := bson.M{"_id": sequenceName}
	update := bson.M{"$inc": bson.M{"seq": 1}}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(true)

	var result struct {
		Seq int `bson:"seq"`
	}

	err := s.counterCollection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&result)
	if err != nil {
		return 0, err
	}

	return result.Seq, nil
}

func (s *UserRepository) UpdateUser(ctx context.Context, filter bson.M, data interface{}) (*model.User, error) {
	return database.UpdateGeneric[model.User](ctx, s.collection, filter, data)
}
