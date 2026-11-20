package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// client tracks the token-bucket limiter for one caller, plus when it was
// last seen so idle entries can be evicted instead of growing the map
// forever.
type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	mu      sync.Mutex
	clients = make(map[string]*client)
)

// RateLimiter returns Gin middleware that allows each client IP 1 request
// per second, with a burst capacity of 5. A client that exceeds this gets
// a 429 response instead of reaching the handler.
func RateLimiter() gin.HandlerFunc {
	go cleanupClients()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()
		cl, exists := clients[ip]
		if !exists {
			cl = &client{limiter: rate.NewLimiter(1, 5)}
			clients[ip] = cl
		}
		cl.lastSeen = time.Now()
		allowed := cl.limiter.Allow()
		mu.Unlock()

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded: 1 request per second, burst of 5",
			})
			return
		}

		c.Next()
	}
}

// cleanupClients periodically drops limiters for clients that haven't been
// seen in a while, so a long-running server doesn't accumulate an entry per
// distinct IP forever.
func cleanupClients() {
	for {
		time.Sleep(time.Minute)
		mu.Lock()
		for ip, cl := range clients {
			if time.Since(cl.lastSeen) > 3*time.Minute {
				delete(clients, ip)
			}
		}
		mu.Unlock()
	}
}
