package database

import (
	"context"
	"crypto/tls"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	RedisHelper *redisUtil
)

type redisUtil struct {
	client *redis.Client
	ctx    context.Context
}

func InitRedis(url string) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		log.Fatalf("Invalid Redis URL: %v", err)
	}

	if opts.TLSConfig == nil {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	redisClient := redis.NewClient(opts)
	ctx := context.Background()

	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Could not connect to Redis/Valkey: %v", err)
	}

	log.Println("✅ Connected to Aiven Valkey successfully")

	RedisHelper = &redisUtil{
		client: redisClient,
		ctx:    ctx,
	}
}

func (r *redisUtil) Set(key string, value interface{}, expiration time.Duration) error {
	err := r.client.Set(r.ctx, key, value, expiration).Err()
	if err != nil {
		log.Printf("Redis SET Error [%s]: %v", key, err)
	}
	return err
}

func (r *redisUtil) Get(key string) (string, error) {
	val, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		log.Printf("Redis GET Error [%s]: %v", key, err)
		return "", err
	}
	return val, nil
}

func (r *redisUtil) Delete(key string) error {
	err := r.client.Del(r.ctx, key).Err()
	if err != nil {
		log.Printf("Redis DEL Error [%s]: %v", key, err)
	}
	return err
}

func (r *redisUtil) Exists(key string) bool {
	count, err := r.client.Exists(r.ctx, key).Result()
	return err == nil && count > 0
}
