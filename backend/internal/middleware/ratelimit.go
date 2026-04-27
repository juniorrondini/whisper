package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func LoginRateLimit(redis *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if redis == nil {
			c.Next()
			return
		}
		key := "rate:login:" + c.ClientIP()
		count, err := redis.Incr(c.Request.Context(), key).Result()
		if err == nil && count == 1 {
			redis.Expire(c.Request.Context(), key, time.Minute)
		}
		if err == nil && count > 10 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many login attempts"})
			return
		}
		c.Next()
	}
}
