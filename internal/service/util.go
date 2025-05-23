package service

import (
	"context"
	"log"
	"time"
)

func (s *UserService) asyncEnsureRedisSet(ctx context.Context, key string, value any, t time.Duration) {
	select {
	case <-ctx.Done():
		log.Println("context done, skipping Redis set")
		return
	default:
		maxRetryTimes := 5
		for i := range maxRetryTimes {
			err := s.redis.Set(ctx, key, value, t).Err()
			if err == nil {
				log.Printf("successfully set Redis key %s", key)
				return
			}
			log.Printf("failed to set Redis key %s, retrying... (%d/%d)", key, i+1, maxRetryTimes)
		}
		log.Printf("failed to set Redis key %s after %d attempts", key, maxRetryTimes)
	}
}
