package forwarder

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const terminationDelay = 1 * time.Second

// Forwarder forwards connections from src to dst using the least connections algorithm.
type Forwarder struct {
	upstreams map[string]*atomic.Int32
	mu        sync.Mutex
}

type IForwarder interface {
	Forward(src net.Conn, allowedUpstreams []string, errChan chan error)
}

// TODO add slog logger

// NewForwarder creates a new Forwarder.
func NewForwarder(upstreams []string) *Forwarder {
	urlMap := make(map[string]*atomic.Int32)
	for _, upstream := range upstreams {
		atomicZero := atomic.Int32{}
		atomicZero.Store(0)
		urlMap[upstream] = &atomicZero
	}

	return &Forwarder{
		upstreams: urlMap,
	}
}

// Forward forwards the connection src to the destination with the least connections.
func (f *Forwarder) Forward(src net.Conn, allowedUpstreams []string, errChan chan error) {
	if src != nil {
		dst := f.getLeastConn(allowedUpstreams)
		f.forward(src, dst, errChan)
	}
}

func (f *Forwarder) getLeastConn(allowed []string) string {
	f.mu.Lock()
	defer f.mu.Unlock()

	leastUsed := ""
	var leastCount int32
	for _, dst := range allowed {
		count := f.upstreams[dst].Load()
		if leastUsed == "" || count < leastCount {
			leastUsed = dst
			leastCount = count
		}
	}

	return leastUsed
}

func (f *Forwarder) forward(src net.Conn, dst string, errChan chan error) {
	defer src.Close()
	log.Debug().Str("destination", dst).Msg("Forwarding to")
	dstConn, err := net.Dial("tcp", dst)
	if err != nil {
		errChan <- err
		src.Close()
		return
	}
	defer dstConn.Close()

	f.incrementConnectionCount(dst)
	internalErrChan := make(chan error, 2)
	go f.copyData(dstConn, src, internalErrChan)
	go f.copyData(src, dstConn, internalErrChan)
	dstConn.SetReadDeadline(time.Now().Add(terminationDelay))
	log.Debug().Msg("Waiting for channel")
	internalErr := <-internalErrChan
	if internalErr != nil {
		err = internalErr
		log.Warn().Err(err).Msg("Error copying data")
	}

	log.Debug().Msg("Waiting for channel")
	internalErr = <-internalErrChan
	if internalErr != nil {
		err = internalErr
		log.Warn().Err(err).Msg("Error copying data")
	}
	f.decrementConnectionCount(dst)
	errChan <- err
}

// Function to copy data between two connections
func (f *Forwarder) copyData(dst io.WriteCloser, src io.Reader, errChan chan error) {
	log.Debug().Msg("Copying data")
	written, err := io.Copy(dst, src)
	log.Debug().Int64("written", written).Msg("Written")
	dst.Close()
	log.Debug().Err(err).Msg("Sending to channel")
	errChan <- err

}

// Function to increment the connection count for an upstream server
func (f *Forwarder) incrementConnectionCount(upstream string) {
	f.mu.Lock()
	val := f.upstreams[upstream]
	val.Add(1)
	f.mu.Unlock()
}

// Function to decrement the connection count for an upstream server
func (f *Forwarder) decrementConnectionCount(upstream string) {
	f.mu.Lock()
	val := f.upstreams[upstream]
	if val.Load() < 1 {
		fmt.Println("Negative counter detected!")
	}
	val.Add(-1)
	f.mu.Unlock()
}
