package main

import (
	"context"
	"fmt"
	"time"

	"github.com/zeroqual/redis-limiter/helpers"
)

func main() {
	rdCl := helpers.InitRedis()
	defer rdCl.Close()
	fmt.Println("connected to redis!")

	//
	userID := "10023423"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	//1)make token bucket
	tokenBucket := helpers.NewTokenBucket(10, 10, 30*time.Second, rdCl)
	//2)init ratelimiter for user
	// tokenBucket.InitRateLimit(userID, ctx)

	//make 'request'if allowed

	allow := tokenBucket.AllowRequest(userID, ctx)

	if !allow {
		fmt.Println("rate limiting")
		return
	}
	fmt.Println("success")
}
