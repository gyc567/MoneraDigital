// internal/middleware/rate_limit.go
package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter 速率限制器
type RateLimiter struct {
	store  map[string][]time.Time
	mu     sync.RWMutex
	limit  int
	window time.Duration
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		store:  make(map[string][]time.Time),
		limit:  limit,
		window: window,
	}

	// 启动清理过期时间戳的后台任务
	go rl.cleanupExpiredTimestamps()

	return rl
}

// IsAllowed 检查是否允许请求
func (rl *RateLimiter) IsAllowed(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	timestamps := rl.store[key]

	// 清理过期的时间戳
	var valid []time.Time
	for _, ts := range timestamps {
		if now.Sub(ts) < rl.window {
			valid = append(valid, ts)
		}
	}

	// 检查是否超过限制
	if len(valid) >= rl.limit {
		rl.store[key] = valid
		return false
	}

	// 添加当前时间戳
	valid = append(valid, now)
	rl.store[key] = valid
	return true
}

// cleanupExpiredTimestamps 清理过期的时间戳
func (rl *RateLimiter) cleanupExpiredTimestamps() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, timestamps := range rl.store {
			var valid []time.Time
			for _, ts := range timestamps {
				if now.Sub(ts) < rl.window {
					valid = append(valid, ts)
				}
			}

			if len(valid) == 0 {
				delete(rl.store, key)
			} else {
				rl.store[key] = valid
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware 速率限制中间件
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取客户端 IP
		clientIP := c.ClientIP()

		// 检查是否允许请求
		if !limiter.IsAllowed(clientIP) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests",
				"code":  "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// PerEndpointRateLimiter 每个端点的速率限制器
type PerEndpointRateLimiter struct {
	limiters map[string]*RateLimiter
	mu       sync.RWMutex
}

// NewPerEndpointRateLimiter 创建每个端点的速率限制器
func NewPerEndpointRateLimiter() *PerEndpointRateLimiter {
	return &PerEndpointRateLimiter{
		limiters: make(map[string]*RateLimiter),
	}
}

// AddEndpoint 添加端点限制
func (p *PerEndpointRateLimiter) AddEndpoint(endpoint string, limit int, window time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.limiters[endpoint] = NewRateLimiter(limit, window)
}

// Middleware 返回中间件
func (p *PerEndpointRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		p.mu.RLock()
		limiter, exists := p.limiters[c.Request.URL.Path]
		p.mu.RUnlock()

		if !exists {
			// 使用默认限制（5 请求/分钟）
			limiter = NewRateLimiter(5, 1*time.Minute)
		}

		clientIP := c.ClientIP()
		key := fmt.Sprintf("%s:%s", c.Request.URL.Path, clientIP)

		if !limiter.IsAllowed(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests",
				"code":  "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
