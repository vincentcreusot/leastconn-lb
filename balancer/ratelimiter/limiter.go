package ratelimiter

import (
	"sync"

	"golang.org/x/time/rate"
)

type IRateLimiter interface {
	Allow(clientId string) bool
}
type client struct {
	limiter *rate.Limiter
	// can include a lastRequestTime to permit cleaning periodically
}

type Ratelimiter struct {
	clients map[string]*client
	burst   int
	rate    int
	mu      sync.Mutex
}

func NewRateLimiter(burst, rate int) *Ratelimiter {
	return &Ratelimiter{
		clients: make(map[string]*client),
		burst:   burst,
		rate:    rate,
	}
}

func (r *Ratelimiter) Allow(clientId string) bool {
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
