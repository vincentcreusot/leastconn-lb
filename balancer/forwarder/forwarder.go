package forwarder

import (
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	terminationDelay = 2 * time.Second
	maxRetry         = 3
	unhealthyTimeout = 30 * time.Second
)

// forward forwards connections from src to dst using the least connections algorithm.
type forward struct {
	upstreams   map[string]*atomic.Int32
	unhealthy   map[string]time.Time
	healthMutex sync.Mutex
}

type Forwarder interface {
	Forward(src net.Conn, allowedUpstreams []string) error
}

// NewForwarder creates a new Forwarder.
func NewForwarder(upstreams []string) *forward {
	urlMap := make(map[string]*atomic.Int32)
	for _, upstream := range upstreams {
		atomicZero := atomic.Int32{}
		atomicZero.Store(0)
		urlMap[upstream] = &atomicZero
	}

	return &forward{
		upstreams: urlMap,
		unhealthy: make(map[string]time.Time),
	}
}

// Forward forwards the connection src to the destination with the least connections.
func (f *forward) Forward(src net.Conn, allowedUpstreams []string) error {
	if src != nil {
		for i := 0; i < maxRetry; i++ {
			if i > 0 {
				log.Debug().Str("remoteAddr", src.RemoteAddr().String()).Msg("Retrying")
			}
			dst := f.getLeastConn(allowedUpstreams)
			if success, err := f.forward(src, dst); success {
				return err
			}
		}

		// Retries exceeded, return error
		return fmt.Errorf("max retries exceeded for %s", src.RemoteAddr())
	}
	return fmt.Errorf("connection is null")
}

func (f *forward) getLeastConn(allowed []string) string {
	leastUsed := ""
	var leastCount int32 = math.MaxInt32
	for _, dst := range allowed {
		if f.isUnhealthy(dst) {
			continue
		}
		count := f.upstreams[dst].Load()
		if count < leastCount {

			leastUsed = dst
			leastCount = count
		}
	}

	return leastUsed
}

// forward returns false if the upstream is unhealthy
func (f *forward) forward(src net.Conn, dst string) (bool, error) {
	defer src.Close()
	log.Debug().Str("destination", dst).Msg("Forwarding to")

	dstConn, err := net.Dial("tcp", dst)
	if err != nil {

		log.Debug().Str("upstream", dst).Msg("Marking as unhealthy")
		return false, nil
	}
	defer dstConn.Close()

	f.incrementConnectionCount(dst)
	var wg sync.WaitGroup
	var err1, err2 error
	wg.Add(1)
	go func() {
		defer wg.Done()
		err1 = f.copyData(dstConn, src)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		err2 = f.copyData(src, dstConn)
	}()

	// TODO find the reason why it's needed
	dstConn.SetReadDeadline(time.Now().Add(terminationDelay))
	wg.Wait()
	f.decrementConnectionCount(dst)

	return true, errors.Join(err1, err2)
}

// Function to copy data between two connections
func (f *forward) copyData(dst io.WriteCloser, src io.Reader) error {
	_, err := io.Copy(dst, src)
	// hack to remove normal close from errors
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if strings.HasSuffix(opErr.Error(), "use of closed network connection") || strings.HasSuffix(opErr.Error(), "i/o timeout") {
			err = nil
		}
	}
	dst.Close()
	return err
}

// Function to increment the connection count for an upstream server
func (f *forward) incrementConnectionCount(upstream string) {
	val := f.upstreams[upstream]
	val.Add(1)
}

// Function to decrement the connection count for an upstream server
func (f *forward) decrementConnectionCount(upstream string) {
	val := f.upstreams[upstream]
	if val.Load() < 1 {
		log.Warn().Msg("Negative counter detected!")
	}
	val.Add(-1)
}

func (f *forward) setUnhealthy(upstream string) {
	f.healthMutex.Lock()
	defer f.healthMutex.Unlock()
	f.unhealthy[upstream] = time.Now()
}
func (f *forward) isUnhealthy(upstream string) bool {
	f.healthMutex.Lock()
	defer f.healthMutex.Unlock()
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
