package service

import (
	"context"
	"log"
	"time"
)

func (s *UserService) EnsureRedisSet(ctx context.Context, key string, value string, t time.Duration) {
	select {
	case <-ctx.Done():
		log.Println("context done, skipping Redis set")
		return
	default:
		maxRetryTimes := 5
		for i := 0; i < maxRetryTimes; i++ {
			err := s.redis.Set(ctx, key, value, t).Err()
			if err == nil {
				log.Printf("successfully set Redis key %s", key)
				return
			}
			durationTime := 1 << i
			if i > 0 {
				log.Printf("wait for Redis set, retrying in %d milliseconds", durationTime)
				time.Sleep(time.Duration(durationTime) * time.Millisecond)
			}
			if i == maxRetryTimes-1 {
				log.Printf("failed to set Redis key %s after %d attempts", key, maxRetryTimes)
				return
			}
			log.Printf("failed to set Redis key %s, retrying... (%d/%d)", key, i+1, maxRetryTimes)
		}
		log.Printf("failed to set Redis key %s after %d attempts", key, maxRetryTimes)
	}
}
