package repository

import (
	"context"
	"errors"

	"backend/model"

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

// --- Finder Methods ---

// FindByEmail retrieves a user by their primary key (email)
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	return r.findOne(ctx, bson.M{"_id": email})
}

// FindByUsername retrieves a user by their username field
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	return r.findOne(ctx, bson.M{"username": username})
}

// FindAll retrieves all users in the collection
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

// --- Persistence Methods ---

// Save performs an Upsert based on the User's Email (_id)
func (r *UserRepository) Save(ctx context.Context, user *model.User) error {
	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": user.Email},
		bson.M{"$set": user},
		opts,
	)
	return err
}

// UpdateTheme performs a partial update on the theme field only
func (r *UserRepository) UpdateTheme(ctx context.Context, email string, theme model.UserTheme) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": email},
		bson.M{"$set": bson.M{"theme": theme}},
	)
	return err
}

// --- Existence & Deletion ---

// ExistsByEmail checks if a record exists using the primary key
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"_id": email})
	return count > 0, err
}

// DeleteByUsername removes a record by the username field
func (r *UserRepository) DeleteByUsername(ctx context.Context, username string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"username": username})
	return err
}

// --- Private Helpers ---

// findOne handles the repetitive logic of FindOne and ErrNoDocuments checking
func (r *UserRepository) findOne(ctx context.Context, filter bson.M) (*model.User, error) {
	var user model.User
	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByMobile(ctx context.Context, mobile string) (*model.User, error) {
	return r.findOne(ctx, bson.M{"mobile": mobile})
}
