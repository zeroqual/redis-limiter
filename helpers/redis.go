package helpers

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

func InitRedis() *redis.Client {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "12345678",
		DB:       0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to ping redis: %v", err)
	}

	return redisClient
}
