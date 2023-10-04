package balancer

import (
	"fmt"
	"net"

	"github.com/rs/zerolog/log"
	"github.com/vincentcreusot/leastconn-lb/balancer/forwarder"
	"github.com/vincentcreusot/leastconn-lb/balancer/ratelimiter"
)

// Balancer provides load balancing functionality
type Balancer interface {
	Balance(conn *net.TCPConn, clientId string, allowedUpstreams []string, errorsChan chan []error)
}

// Config holds balancer configuration
type Config struct {
	Burst     int
	Rate      int
	Upstreams []string
}

type balance struct {
	forwarder   forwarder.Forwarder
	rateLimiter ratelimiter.IRateLimiter
}

func NewBalancer(c Config) *balance {
	b := balance{
		forwarder:   forwarder.NewForwarder(c.Upstreams),
		rateLimiter: ratelimiter.NewRateLimiter(c.Burst, c.Rate),
	}
	return &b
}

func (b *balance) Balance(conn net.Conn, clientId string, allowedUpstreams []string, errorsChan chan error) {
	var err error
	if b.rateLimiter.Allow(clientId) {
		err = b.forwarder.Forward(conn, allowedUpstreams)
	} else {
		log.Debug().Str("client", clientId).Msg("limited")
		err = fmt.Errorf("client rate limited")
		conn.Close()
	}
	errorsChan <- err
}
