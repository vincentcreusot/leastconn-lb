package forwarder

import (
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

const terminationDelay = 2 * time.Second
const maxRetry = 3
const unhealthyTimeout = 30 * time.Second

// Forwarder forwards connections from src to dst using the least connections algorithm.
type Forwarder struct {
	upstreams map[string]*atomic.Int32
	unhealthy map[string]time.Time
	mu        sync.Mutex
}

type IForwarder interface {
	Forward(src net.Conn, allowedUpstreams []string, errorsChan chan []error)
}

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
		unhealthy: make(map[string]time.Time),
	}
}

// Forward forwards the connection src to the destination with the least connections.
func (f *Forwarder) Forward(src net.Conn, allowedUpstreams []string, errorsChan chan []error) {
	if src != nil {
		for i := 0; i < maxRetry; i++ {
			log.Debug().Str("remoteAddr", src.RemoteAddr().String()).Msg("Retrying")
			dst := f.getLeastConn(allowedUpstreams)
			if f.forward(src, dst, errorsChan) {
				return
			}
		}

		// Retries exceeded, return error
		errorsChan <- []error{fmt.Errorf("max retries exceeded for %s", src.RemoteAddr())}
		return
	}
	errorsChan <- []error{fmt.Errorf("connection is null")}
}

func (f *Forwarder) getLeastConn(allowed []string) string {
	f.mu.Lock()
	defer f.mu.Unlock()

	leastUsed := ""
	var leastCount int32
	for _, dst := range allowed {
		if f.isUnhealthy(dst) {
			continue
		}
		count := f.upstreams[dst].Load()
		if leastUsed == "" || count < leastCount {

			leastUsed = dst
			leastCount = count
		}
	}

	return leastUsed
}

func (f *Forwarder) forward(src net.Conn, dst string, errorsChan chan []error) bool {
	defer src.Close()
	log.Debug().Str("destination", dst).Msg("Forwarding to")

	dstConn, err := net.Dial("tcp", dst)
	if err != nil {
		f.unhealthy[dst] = time.Now()
		src.Close()
		return false
	}
	defer dstConn.Close()

	f.incrementConnectionCount(dst)
	internalErrChan := make(chan error, 2)
	go f.copyData(dstConn, src, internalErrChan)
	go f.copyData(src, dstConn, internalErrChan)
	errorsSlice := make([]error, 0)
	// TODO find the reason why it's needed
	dstConn.SetReadDeadline(time.Now().Add(terminationDelay))
	internalErr := <-internalErrChan

	if internalErr != nil {
		errorsSlice = append(errorsSlice, internalErr)
	}
	internalErr = <-internalErrChan
	if internalErr != nil {
		errorsSlice = append(errorsSlice, internalErr)
	}
	f.decrementConnectionCount(dst)
	errorsChan <- errorsSlice
	return true
}

// Function to copy data between two connections
func (f *Forwarder) copyData(dst io.WriteCloser, src io.Reader, errChan chan error) {
	_, err := io.Copy(dst, src)
	// hack to remove normal close from errors
	e, ok := err.(*net.OpError)
	if ok && (e.Err.Error() == "use of closed network connection" || e.Err.Error() == "i/o timeout") {
		err = nil
	}
	dst.Close()
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
		log.Warn().Msg("Negative counter detected!")
	}
	val.Add(-1)
	f.mu.Unlock()
}

func (f *Forwarder) isUnhealthy(upstream string) bool {
	unhealthyTime, found := f.unhealthy[upstream]
	if !found {
		return false
	}
	if time.Since(unhealthyTime) > unhealthyTimeout {
		delete(f.unhealthy, upstream)
		return false
	}
	log.Debug().Str("upstream", upstream).Msg("Unhealthy")
	return true
}
