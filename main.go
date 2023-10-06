package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/vincentcreusot/leastconn-lb/server"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// TODO: should go into config, hardcoding for now
	upstreams := []string{"localhost:9801", "localhost:9802"}
	s, err := server.NewServer(server.Config{
		Address:        "0.0.0.0:8888",
		Upstreams:      upstreams,
		CaCertFile:     "certs/ca/ca.crt",
		ServerCertFile: "certs/server/server.crt",
		ServerKeyFile:  "certs/server/server.key.pem",
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create server")
		sigs <- syscall.SIGINT
	}
	s.Start()

	<-sigs

	log.Info().Msg("Shutting down server...")
	s.Stop()

	log.Info().Msg("Server stopped")
}
