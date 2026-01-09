package store

import (
	"context"
	"crypto/tls"
	"log"

	"github.com/go-redis/redis/v8"
)

// Using the go-redis library
func InitRedis(url string) *redis.Client {
	// 1. This helper parses the URL and extracts the password/host correctly
	opt, err := redis.ParseURL(url)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}

	// 2. Upstash usually requires TLS. Let's ensure it's set.
	if opt.TLSConfig == nil {
		opt.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	client := redis.NewClient(opt)

	// 3. Test the connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	return client
}
