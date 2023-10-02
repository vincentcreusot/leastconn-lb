package ratelimiter

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRatelimiter_Allow_Burst(t *testing.T) {

	r := NewRateLimiter(10, 1)

	// Allow for new client should succeed
	assert.True(t, r.Allow("client1"))
	// wait to refill burst
	time.Sleep(1 * time.Second)
	// Verify rate limiting
	for i := 0; i < 10; i++ {
		assert.True(t, r.Allow("client1"))
	}

	// Should be rate limited now
	assert.False(t, r.Allow("client1"))

	// Wait for bucket to refill
	time.Sleep(1 * time.Second)

	// Should allow again
	assert.True(t, r.Allow("client1"))

	// Different client should not be limited
	assert.True(t, r.Allow("client2"))
}

func TestRate(t *testing.T) {

	r := NewRateLimiter(10, 5)

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	// pass the burst
	for i := 0; i < 10; i++ {
		assert.True(t, r.Allow("client"))
	}
	// allowed rate
	for i := 0; i < 5; i++ {
		<-ticker.C
		assert.True(t, r.Allow("client"))
		fmt.Println(i)
	}

	// Rate limit should kick in
	assert.False(t, r.Allow("client"))

	// Wait for full second
	time.Sleep(1000 * time.Millisecond)

	// Should allow again
	assert.True(t, r.Allow("client"))

}
