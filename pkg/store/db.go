package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq" // Postgres driver
)

type Store struct {
	DB  *sql.DB
	RDB *redis.Client
	Ctx context.Context
}

func initSchema(db *sql.DB) error {
	schema := `
		CREATE TABLE IF NOT EXISTS clips (
			room_id TEXT PRIMARY KEY,
			content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_clips_updated_at ON clips(updated_at);`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}
	return err
}

// NewStore initializes both Postgres and Redis connections
func NewStore(pgConnStr, redisAddr string) (*Store, error) {
	ctx := context.Background()

	var db *sql.DB
	var err error

	// 1. Setup PostgreSQL
	// Retry Postgres connection 5 times
	for i := 0; i < 5; i++ {
		db, err = sql.Open("postgres", pgConnStr)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Printf("Waiting for Postgres... (attempt %d/5)", i+1)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return nil, err
	}

	// 2. Setup Redis
	// rdb := redis.NewClient(&redis.Options{
	// 	Addr: redisAddr,
	// })
	rdb := InitRedis(redisAddr)

	// Verify Redis connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	log.Println("Successfully connected to Postgres and Redis")

	// initilize the db schema
	if err := initSchema(db); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &Store{
		DB:  db,
		RDB: rdb,
		Ctx: ctx,
	}, nil
}

// SaveContent updates Redis (instant) and Postgres (persistent)
func (s *Store) SaveContent(roomID, content string) error {
	// Set in Redis with 1 Hour TTL
	// If Redis fails, stop everything and tell the user "Error"
	if err := s.RDB.Set(s.Ctx, "room:"+roomID, content, 1*time.Hour).Err(); err != nil {
		log.Printf("Redis save error: %v", err)
	}

	query := `
        INSERT INTO clips (room_id, content, updated_at)
        VALUES ($1, $2, NOW())
        ON CONFLICT (room_id) 
        DO UPDATE SET content = EXCLUDED.content, updated_at = NOW()`

	// If Postgres fails, stop and tell the user "Error"
	if _, err := s.DB.Exec(query, roomID, content); err != nil {
		return fmt.Errorf("postgres save error: %w", err)
	}

	log.Printf("Room [%s] saved successfully", roomID)

	// If Redis failed but Postgres worked, the user gets a "Success".
	return nil
}

// GetContent fetches the latest text (checks Redis first, then Postgres)
func (s *Store) GetContent(roomID string) (string, error) {
	// Try Redis first
	val, err := s.RDB.Get(s.Ctx, "room:"+roomID).Result()
	if err == nil {
		return val, nil
	}

	// If not in Redis, check Postgres
	var content string
	err = s.DB.QueryRow("SELECT content FROM clips WHERE room_id = $1", roomID).Scan(&content)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // Room doesn't exist yet
		}
		return "", err
	}
	log.Println("Successfully fetched from Postgres")
	return content, nil
}

func (s *Store) DeleteContent(roomID string) error {
	query := `DELETE FROM clips WHERE room_id = $1`
	_, err := s.DB.Exec(query, roomID)
	log.Println("Successfully deleted from Postgres")
	return err
}

// StartCleanupWorker runs in the background to delete old clips
func (s *Store) StartCleanupWorker(interval time.Duration, maxAge time.Duration) {
	ticker := time.NewTicker(interval)
	log.Printf("Cleanup worker started: interval=%v, maxAge=%v", interval, maxAge)

	for range ticker.C {
		result, err := s.DB.Exec(
			"DELETE FROM clips WHERE updated_at < NOW() - $1::interval",
			maxAge.String(),
		)
		if err != nil {
			log.Printf("Error during cleanup: %v", err)
			continue
		}

		rows, _ := result.RowsAffected()
		if rows > 0 {
			log.Printf("Cleanup complete: deleted %d expired clips", rows)
		}
	}
}
