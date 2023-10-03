package balancer

import (
	"net"

	"github.com/rs/zerolog/log"
	"github.com/vincentcreusot/leastconn-lb/balancer/forwarder"
	"github.com/vincentcreusot/leastconn-lb/balancer/ratelimiter"
)

type IBalancer interface {
	Balance(conn *net.TCPConn, clientId string, allowedUpstreams []string, errorsChan chan []error)
}
type Balancer struct {
	forwarder   forwarder.IForwarder
	rateLimiter ratelimiter.IRateLimiter
}

func NewBalancer(burst int, rate int, upstreams []string) *Balancer {
	b := Balancer{
		forwarder:   forwarder.NewForwarder(upstreams),
		rateLimiter: ratelimiter.NewRateLimiter(burst, rate),
	}
	return &b
}

func (b *Balancer) Balance(conn net.Conn, clientId string, allowedUpstreams []string, errorsChan chan []error) {
	if b.rateLimiter.Allow(clientId) {
		b.forwarder.Forward(conn, allowedUpstreams, errorsChan)
	} else {
		log.Debug().Str("client", clientId).Msg("limited")
		conn.Close()
	}
}
