package main

import (
	"fmt"
	"net"

	"github.com/rs/zerolog/log"
	"github.com/vincentcreusot/leastconn-lb/balancer"
)

func main() {
	upstreams := []string{"localhost:9801", "localhost:9802"}
	// Listen for incoming connections on port 8888
	listener, err := net.Listen("tcp", "0.0.0.0:8888")
	if err != nil {
		fmt.Printf("Error listening: %v\n", err)
		return
	}
	defer listener.Close()

	fmt.Println("Forwarder is listening on port 8888...")
	balance := balancer.NewBalancer(5, 10, upstreams)

	errorsChan := make(chan []error, 0)
	// Accept incoming connections and forward them to upstream servers
	for {
		clientConn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}

		go balance.Balance(clientConn, clientConn.LocalAddr().String(), upstreams, errorsChan)
		go displayErrors(errorsChan)
	}

}

func displayErrors(errorsChan chan []error) {
	errs := <-errorsChan
	for _, err := range errs {
		log.Debug().Err(err)
	}
}
