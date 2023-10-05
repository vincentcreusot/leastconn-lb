package server

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/vincentcreusot/leastconn-lb/balancer"
)

type Server interface {
	Start()
	Stop()
}

type serve struct {
	wg         sync.WaitGroup
	listener   net.Listener
	shutdown   chan struct{}
	connection chan *tls.Conn
	balancer   balancer.Balancer
	authScheme *AuthScheme
}

func NewServer(address string, upstreams []string) (Server, error) {
	tlsConfig, err := getTlsConfig()
	if err != nil {
		return nil, err
	}
	listener, err := tls.Listen("tcp", address, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on address %s: %w", address, err)
	}
	balance := balancer.NewBalancer(balancer.Config{Burst: 20, Rate: 20, Upstreams: upstreams})
	auth := NewAuthScheme()
	return &serve{
		listener:   listener,
		shutdown:   make(chan struct{}),
		connection: make(chan *tls.Conn, 100),
		balancer:   balance,
		authScheme: auth,
	}, nil
}

func getTlsConfig() (*tls.Config, error) {
	caCert, _ := os.ReadFile("ca.crt")
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	cer, err := tls.LoadX509KeyPair("certs/cert.pem", "certsk/ey.pem")
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
		Certificates: []tls.Certificate{cer},
		ClientAuth:   tls.RequireAndVerifyClientCert, // set mutual tls
		ClientCAs:    caCertPool,
	}

	return cfg, nil
}

func (s *serve) acceptConnections() {
	defer s.wg.Done()

	for {
		select {
		case <-s.shutdown:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				continue
			}
			s.connection <- conn.(*tls.Conn)
		}
	}
}

func (s *serve) handleConnections() {
	defer s.wg.Done()

	for {
		select {
		case <-s.shutdown:
			return
		case conn := <-s.connection:
			go s.handleConnection(conn)
		}
	}
}

func (s *serve) handleConnection(conn *tls.Conn) {
	defer conn.Close()
	// TODO understand the list of Peer Certificates and see which one to take
	// index 0 should be the leaf certificate so the host one
	clientId := conn.ConnectionState().PeerCertificates[0].Subject.CommonName

	allowed := s.authScheme.GetAllowedUpstreams(clientId)
	if allowed == nil {
		log.Warn().Str("client", clientId).Msg("Client not allowed")
		return
	}
	err := s.balancer.Balance(conn, clientId, allowed)
	if err != nil {
		if errors.Is(err, &balancer.RateLimiterError{}) {
			log.Warn().Err(err).Msg("Connection rate limited")
		}
		log.Error().Err(err).Msg("Error forwarding")
	}
}

// Start start accepting and handling connections.
func (s *serve) Start() {
	s.wg.Add(2)
	go s.acceptConnections()
	go s.handleConnections()
}

// Stop stop accepting and handling connections.
func (s *serve) Stop() {
	close(s.shutdown)
	s.listener.Close()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(time.Second):
		log.Warn().Msg("Timed out waiting for connections to finish.")
		return
	}
}
