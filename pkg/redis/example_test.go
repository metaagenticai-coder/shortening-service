package redis_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/metaagenticai/shortening-service/pkg/redis"
)

// ExampleNewClient demonstrates how to create a new Redis client
func ExampleNewClient() {
	ctx := context.Background()
	config := redis.DefaultConfig("redis://localhost:6379/0")

	client, err := redis.NewClient(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	fmt.Println("Redis client created successfully")
}

// ExampleClient_Set demonstrates how to set a key-value pair in Redis
func ExampleClient_Set() {
	ctx := context.Background()
	config := redis.DefaultConfig("redis://localhost:6379/0")

	client, err := redis.NewClient(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	// Set a key with 1 hour expiration
	err = client.Set(ctx, "mykey", "myvalue", time.Hour)
	if err != nil {
		log.Fatalf("Failed to set key: %v", err)
	}

	fmt.Println("Key set successfully")
}

// ExampleClient_Get demonstrates how to retrieve a value from Redis
func ExampleClient_Get() {
	ctx := context.Background()
	config := redis.DefaultConfig("redis://localhost:6379/0")

	client, err := redis.NewClient(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	// First set a value
	client.Set(ctx, "mykey", "myvalue", time.Hour)

	// Then retrieve it
	value, err := client.Get(ctx, "mykey")
	if err != nil {
		log.Fatalf("Failed to get key: %v", err)
	}

	fmt.Printf("Retrieved value: %s\n", value)
}

// ExampleClient_Incr demonstrates how to increment a counter in Redis
func ExampleClient_Incr() {
	ctx := context.Background()
	config := redis.DefaultConfig("redis://localhost:6379/0")

	client, err := redis.NewClient(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	// Increment a counter (will create it if doesn't exist)
	count, err := client.Incr(ctx, "request_count")
	if err != nil {
		log.Fatalf("Failed to increment counter: %v", err)
	}

	fmt.Printf("Request count: %d\n", count)
}

// ExampleClient_HealthCheck demonstrates how to perform a health check
func ExampleClient_HealthCheck() {
	ctx := context.Background()
	config := redis.DefaultConfig("redis://localhost:6379/0")

	client, err := redis.NewClient(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	// Perform comprehensive health check
	err = client.HealthCheck(ctx)
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}

	fmt.Println("Redis is healthy")
}

// ExampleClient_TTL demonstrates how to check the time-to-live of a key
func ExampleClient_TTL() {
	ctx := context.Background()
	config := redis.DefaultConfig("redis://localhost:6379/0")

	client, err := redis.NewClient(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	// Set a key with expiration
	client.Set(ctx, "temp_key", "temp_value", 5*time.Minute)

	// Check TTL
	ttl, err := client.TTL(ctx, "temp_key")
	if err != nil {
		log.Fatalf("Failed to get TTL: %v", err)
	}

	fmt.Printf("Key will expire in: %v\n", ttl)
}

// ExampleClient_rateLimiting demonstrates how to implement rate limiting with Redis
func ExampleClient_rateLimiting() {
	ctx := context.Background()
	config := redis.DefaultConfig("redis://localhost:6379/0")

	client, err := redis.NewClient(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	// Simulate rate limiting for an IP address
	ipAddress := "192.168.1.100"
	rateLimitKey := fmt.Sprintf("ratelimit:ip:%s", ipAddress)
	maxRequests := int64(100)
	window := time.Minute

	// Increment request count
	count, err := client.Incr(ctx, rateLimitKey)
	if err != nil {
		log.Fatalf("Failed to increment rate limit counter: %v", err)
	}

	// Set expiration on first request
	if count == 1 {
		err = client.Expire(ctx, rateLimitKey, window)
		if err != nil {
			log.Fatalf("Failed to set expiration: %v", err)
		}
	}

	// Check if rate limit exceeded
	if count > maxRequests {
		fmt.Println("Rate limit exceeded")
	} else {
		fmt.Printf("Request allowed. Count: %d/%d\n", count, maxRequests)
	}
}

// ExampleClient_caching demonstrates how to use Redis for caching URL mappings
func ExampleClient_caching() {
	ctx := context.Background()
	config := redis.DefaultConfig("redis://localhost:6379/0")

	client, err := redis.NewClient(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	shortCode := "aB3xY9"
	longURL := "https://example.com/very/long/url"
	cacheKey := fmt.Sprintf("url:%s", shortCode)
	cacheTTL := time.Hour

	// Cache the URL mapping
	err = client.Set(ctx, cacheKey, longURL, cacheTTL)
	if err != nil {
		log.Fatalf("Failed to cache URL: %v", err)
	}

	// Later, retrieve from cache
	cachedURL, err := client.Get(ctx, cacheKey)
	if err != nil {
		log.Fatalf("Failed to get cached URL: %v", err)
	}

	if cachedURL != "" {
		fmt.Printf("Cache hit! URL: %s\n", cachedURL)
	} else {
		fmt.Println("Cache miss - need to query database")
	}
}

// ExampleClient_PoolStats demonstrates how to monitor connection pool statistics
func ExampleClient_PoolStats() {
	ctx := context.Background()
	config := redis.DefaultConfig("redis://localhost:6379/0")

	client, err := redis.NewClient(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	// Perform some operations
	client.Set(ctx, "key1", "value1", time.Hour)
	client.Get(ctx, "key1")

	// Get pool statistics
	stats := client.PoolStats()
	if stats != nil {
		fmt.Printf("Total connections: %d\n", stats.TotalConns)
		fmt.Printf("Idle connections: %d\n", stats.IdleConns)
		fmt.Printf("Stale connections: %d\n", stats.StaleConns)
	}
}
