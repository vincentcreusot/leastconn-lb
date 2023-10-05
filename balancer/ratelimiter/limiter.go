package ratelimiter

import (
	"sync"

	"time"

	"golang.org/x/time/rate"
)

type Clock interface {
	Now() time.Time
}

type RateLimiter interface {
	Allow(clientId string) bool
}
type client struct {
	limiter *rate.Limiter
}

type ratelimit struct {
	clients map[string]*client
	burst   int
	rate    int
	mu      sync.Mutex
	clock   Clock
}

func NewRateLimiter(burst, rate int) *ratelimit {
	return &ratelimit{
		clients: make(map[string]*client),
		burst:   burst,
		rate:    rate,
		clock:   &realtime{},
	}
}

func (r *ratelimit) Allow(clientId string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.clients[clientId]
	if !ok {
		l := rate.NewLimiter(rate.Limit(r.rate), r.burst)
		c = &client{
			limiter: l,
		}
		r.clients[clientId] = c
	}
	return c.limiter.AllowN(r.clock.Now(), 1)
}

type realtime struct{}

func (r *realtime) Now() time.Time {
	return time.Now()
}
