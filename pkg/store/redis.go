package store

import (
	"context"
	"crypto/tls"
	"log"

	"github.com/go-redis/redis/v8"
)

// Using the go-redis library
func InitRedis(url string) *redis.Client {
	// Parse the URL without the "?ssl=true" parameter
	opt, err := redis.ParseURL(url)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}

	// Explicitly enable TLS for Upstash
	// This replaces the need for "?ssl=true" in the URL
	opt.TLSConfig = &tls.Config{
		InsecureSkipVerify: true, // Upstash uses managed certificates
	}

	client := redis.NewClient(opt)

	// Test the connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}

	return client
}
