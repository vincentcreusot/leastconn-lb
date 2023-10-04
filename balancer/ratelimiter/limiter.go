package ratelimiter

import (
	"sync"

	"golang.org/x/time/rate"
)

type RateLimiter interface {
	Allow(clientId string) bool
}
type client struct {
	limiter *rate.Limiter
	// can include a lastRequestTime to permit cleaning periodically
}

type ratelimit struct {
	clients map[string]*client
	burst   int
	rate    int
	mu      sync.Mutex
}

func NewRateLimiter(burst, rate int) *ratelimit {
	return &ratelimit{
		clients: make(map[string]*client),
		burst:   burst,
		rate:    rate,
	}
}

func (r *ratelimit) Allow(clientId string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.clients[clientId]
	if !ok {
		c = &client{
			limiter: rate.NewLimiter(rate.Limit(r.rate), r.burst),
		}
		r.clients[clientId] = c
	}
	return c.limiter.Allow()
}
