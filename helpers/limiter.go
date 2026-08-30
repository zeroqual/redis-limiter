package helpers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenBucket struct {
	Capacity       int
	RefillRate     int           //2
	RefillInterval time.Duration //60
	client         *redis.Client
}

func NewTokenBucket(cap, rate int, interval time.Duration, client *redis.Client) *TokenBucket {
	return &TokenBucket{
		Capacity:       cap,
		RefillRate:     rate,
		RefillInterval: interval,
		client:         client,
	}
}

// init rate limit in redis if not exists
func (b *TokenBucket) InitRateLimit(userID string, ctx context.Context) {
	key := fmt.Sprintf("rate_limit:%s", userID)
	cmd := b.client.HGet(ctx, key, "capacity")
	_, err := cmd.Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			_, err := b.client.HSet(ctx, key, "capacity", b.Capacity, "last_refill_time", time.Now().Format(time.RFC3339)).Result()
			if err != nil {
				log.Printf("failed to create rate limit in redis: %v", err)
			}
		}
		log.Printf("unknown err: %v", err)
	}
}

func (b *TokenBucket) GetLastValues(userID string, ctx context.Context) (tokens int, lastRefill time.Time, err error) {
	//trying to get all info
	key := fmt.Sprintf("rate_limit:%s", userID)
	all, err := b.client.HGetAll(ctx, key).Result()
	if len(all) == 0 {
		now := time.Now()
		_, err := b.client.HSet(ctx, key, "capacity", b.Capacity, "last_refill_time", now).Result()
		if err != nil {
			return 0, time.Time{}, fmt.Errorf("failed to create rate limit in redis: %w", err)
		}
		return b.Capacity, now, nil
	}

	//parse values from map
	tokens1, err := strconv.Atoi(all["capacity"])
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("invalid tokens value in redis: %w", err)
	}
	lastRefill, err = time.Parse(time.RFC3339, all["last_refill_time"])
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("invalid last_refill_time value in redis: %w", err)
	}

	return tokens1, lastRefill, nil
}

func (b *TokenBucket) RefillTokens(userID string, ctx context.Context, tokens int, lastRefill time.Time) int {
	elapsed := time.Since(lastRefill) // 60 sec
	ref := int(elapsed / b.RefillInterval)

	//1

	newTokens := tokens + ref*b.RefillRate
	fmt.Printf("elapsed:%s\nref:%d\nnewToken:%d\n", elapsed, ref, newTokens)
	if newTokens > b.Capacity {
		newTokens = b.Capacity
	}

	key := fmt.Sprintf("rate_limit:%s", userID)
	_, err := b.client.HSet(ctx, key, "capacity", newTokens, "last_refill_time", time.Now().Format(time.RFC3339)).Result()
	if err != nil {
		log.Printf("failed to set new capacity: %v", err)
		return -1
	}
	return newTokens
}

func (b *TokenBucket) AllowRequest(userID string, ctx context.Context) bool {
	key := fmt.Sprintf("rate_limit:%s", userID)
	token, lastRefill, err := b.GetLastValues(userID, ctx)
	if err != nil {
		log.Printf("error: %v", err)
		return false
	}

	// if we got tokens + last refill make refill if possible

	newTokens := b.RefillTokens(userID, ctx, token, lastRefill)
	if newTokens > 0 {
		res, err := b.client.HIncrBy(ctx, key, "capacity", -1).Result()
		if err != nil {
			log.Printf("failed to incr capacity: %v", err)
			return false
		}
		if res == 0 {
			log.Print("increase err = nil but its not increase")
			return false
		}
		return true
	}

	return false
}
