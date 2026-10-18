package queue

import (
	"context"
	// "time"

	"github.com/redis/go-redis/v9"
)

// Queue defines the interface for job queue operations.
type Queue interface {
	Push(ctx context.Context, key string, value string) error
	Pop(ctx context.Context, key string) (string, error)
	Peek(ctx context.Context, key string) (string, error)
}

// RedisQueue implements Queue using Redis.
type RedisQueue struct {
	client *redis.Client
}

// NewRedisQueue creates a new Redis queue from a plain host:port address,
// no auth or TLS. Kept for local/dev use (e.g. a bare localhost:6379).
func NewRedisQueue(addr string) *RedisQueue {
	return &RedisQueue{
		client: redis.NewClient(&redis.Options{
			Addr: addr,
		}),
	}
}

// NewRedisQueueFromURL creates a new Redis queue from a connection URL,
// e.g. redis://:password@host:port or rediss://default:password@host:port
// (rediss:// enables TLS, as required by managed providers like Upstash).
func NewRedisQueueFromURL(url string) (*RedisQueue, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	return &RedisQueue{client: redis.NewClient(opts)}, nil
}

func (q *RedisQueue) Push(ctx context.Context, key string, value string) error {
	return q.client.LPush(ctx, key, value).Err()
}

func (q *RedisQueue) Pop(ctx context.Context, key string) (string, error) {
	return q.client.RPop(ctx, key).Result()
}

func (q *RedisQueue) Peek(ctx context.Context, key string) (string, error) {
	return q.client.LIndex(ctx, key, -1).Result()
}

// Close closes the Redis connection.
func (q *RedisQueue) Close() error {
	return q.client.Close()
}



// package queue

// import (
// 	"context"
// 	"time"

// 	"github.com/redis/go-redis/v9"
// )

// // Queue defines the interface for job queue operations.
// type Queue interface {
// 	Push(ctx context.Context, key string, value string) error
// 	Pop(ctx context.Context, key string) (string, error)
// 	Peek(ctx context.Context, key string) (string, error)
// }

// // RedisQueue implements Queue using Redis.
// type RedisQueue struct {
// 	client *redis.Client
// }

// // NewRedisQueue creates a new Redis queue.
// func NewRedisQueue(addr string) *RedisQueue {
// 	return &RedisQueue{
// 		client: redis.NewClient(&redis.Options{
// 			Addr: addr,
// 		}),
// 	}
// }

// func (q *RedisQueue) Push(ctx context.Context, key string, value string) error {
// 	return q.client.LPush(ctx, key, value).Err()
// }

// func (q *RedisQueue) Pop(ctx context.Context, key string) (string, error) {
// 	return q.client.RPop(ctx, key).Result()
// }

// func (q *RedisQueue) Peek(ctx context.Context, key string) (string, error) {
// 	return q.client.LIndex(ctx, key, -1).Result()
// }

// // Close closes the Redis connection.
// func (q *RedisQueue) Close() error {
// 	return q.client.Close()
// }