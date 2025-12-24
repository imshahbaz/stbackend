package repository

import (
	"backend/model"
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserRepository struct {
	collection *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{
		collection: db.Collection("users"),
	}
}

// FindByEmail replaces Optional<User> findByEmail(String email)
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.collection.FindOne(ctx, bson.M{"_id": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil // Return nil, nil to represent an empty Optional
		}
		return nil, err
	}
	return &user, nil
}

// FindByUsername replaces Optional<User> findByUsername(String username)
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	// Using "_id" because in your User entity, Username is tagged as bson:"_id"
	err := r.collection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// Save (Insert or Update)
func (r *UserRepository) Save(ctx context.Context, user *model.User) error {
	filter := bson.M{"_id": user.Email}
	update := bson.M{"$set": user}
	opts := options.Update().SetUpsert(true)

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// FindAll
func (r *UserRepository) FindAll(ctx context.Context) ([]model.User, error) {
	var users []model.User
	cursor, err := r.collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// DeleteByUsername
func (r *UserRepository) DeleteByUsername(ctx context.Context, username string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": username})
	return err
}

// ExistsByEmail
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"email": email})
	return count > 0, err
}

func (r *UserRepository) UpdateTheme(ctx context.Context, email string, theme model.UserTheme) (bool, error) {
	filter := bson.M{"_id": email}
	update := bson.M{
		"$set": bson.M{
			"theme": theme,
		},
	}

	// Enable Upsert
	opts := options.Update().SetUpsert(true)

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return false, err
	}

	return true, nil
}
