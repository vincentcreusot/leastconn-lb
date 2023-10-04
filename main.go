package main

import (
	"net"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/vincentcreusot/leastconn-lb/balancer"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	upstreams := []string{"localhost:9801", "localhost:9802"}
	// Listen for incoming connections on port 8888
	listener, err := net.Listen("tcp", "0.0.0.0:8888")
	if err != nil {
		log.Error().Err(err).Msg("Error listening")
		return
	}
	defer listener.Close()

	log.Info().Msg("Listening on 0.0.0.0:8888")
	balance := balancer.NewBalancer(balancer.Config{Burst: 20, Rate: 20, Upstreams: upstreams})

	// Accept incoming connections and forward them to upstream servers
	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Warn().Err(err).Msg("Error accepting connection")
			continue
		}

		go func() {
			err := balance.Balance(clientConn, clientConn.LocalAddr().String(), upstreams)
			if err != nil {
				log.Error().Err(err).Msg("Error forwarding")
			}
		}()
	}

}
