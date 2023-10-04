package main

import (
	"net"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/vincentcreusot/leastconn-lb/balancer"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	upstreams := []string{"localhost:9801", "localhost:9802"}
	// Listen for incoming connections on port 8888
	listener, err := net.Listen("tcp", "0.0.0.0:8888")
	if err != nil {
		log.Error().Err(err).Msg("Error listening")
		return
	}
	defer listener.Close()

	log.Info().Msg("Listening on 0.0.0.0:8888")
	balance := balancer.NewBalancer(balancer.Config{Burst: 5, Rate: 5, Upstreams: upstreams})

	errorsChan := make(chan error, 10)
	// Accept incoming connections and forward them to upstream servers
	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Warn().Err(err).Msg("Error accepting connection")
			continue
		}

		go balance.Balance(clientConn, clientConn.LocalAddr().String(), upstreams, errorsChan)
		go displayErrors(errorsChan)
	}

}

func displayErrors(errorsChan <-chan error) {
	err := <-errorsChan
	log.Error().Err(err)
}
