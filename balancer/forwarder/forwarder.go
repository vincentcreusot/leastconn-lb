package forwarder

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"slices"

	"github.com/rs/zerolog/log"
)

const (
	terminationDelay = 2 * time.Second
	maxRetry         = 3
	unhealthyTimeout = 30 * time.Second
)

type upstream struct {
	addr string
	load *atomic.Int32
}

// forward forwards connections from src to dst using the least connections algorithm.
type forward struct {
	ups         []*upstream
	unhealthy   map[string]time.Time
	healthMutex sync.Mutex
}

type Forwarder interface {
	Forward(src net.Conn, allowedUpstreams []string) error
}

// NewForwarder creates a new Forwarder.
func NewForwarder(upstreams []string) *forward {
	upstreamSlice := make([]*upstream, len(upstreams))
	for i, u := range upstreams {
		atomicZero := atomic.Int32{}
		atomicZero.Store(0)
		upstreamSlice[i] = &upstream{addr: u, load: &atomicZero}
	}

	return &forward{
		ups:       upstreamSlice,
		unhealthy: make(map[string]time.Time),
	}
}

// Forward forwards the connection src to the destination with the least connections.
func (f *forward) Forward(src net.Conn, allowedUpstreams []string) error {
	if src == nil {
		return fmt.Errorf("connection is null")
	}

	for i := 0; i < maxRetry; i++ {
		if i > 0 {
			log.Debug().Str("remoteAddr", src.RemoteAddr().String()).Msg("Retrying")
		}
		dst := f.getLeastConn(allowedUpstreams)
		if dst == nil {
			continue
		}
		if success, err := f.forward(src, dst); success {
			return err
		}
	}

	// Retries exceeded, return error
	return fmt.Errorf("max retries exceeded for %s", src.RemoteAddr())

}

func (f *forward) getLeastConn(allowed []string) *upstream {
	upstreams := append(f.ups[:0:0], f.ups...)
	sort.SliceStable(upstreams, func(i, j int) bool {
		return upstreams[i].load.Load() < upstreams[j].load.Load()
	})
	for _, v := range upstreams {
		if slices.Contains(allowed, v.addr) {
			return v
		}
	}
	return nil
}

// forward returns false if the upstream is unhealthy
func (f *forward) forward(src net.Conn, dst *upstream) (bool, error) {
	defer src.Close()
	log.Debug().Str("destination", dst.addr).Msg("Forwarding to")

	dstConn, err := net.Dial("tcp", dst.addr)
	if err != nil {
		f.setUnhealthy(dst.addr)
		log.Debug().Str("upstream", dst.addr).Msg("Marking as unhealthy")
		return false, nil
	}
	defer dstConn.Close()

	incrementConnectionCount(dst)
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
	decrementConnectionCount(dst)

	return true, errors.Join(err1, err2)
}

// Function to copy data between two connections
func (f *forward) copyData(dst io.WriteCloser, src io.Reader) error {
	defer dst.Close()
	_, err := io.Copy(dst, src)
	// hack to remove normal close from errors
	if os.IsTimeout(err) || errors.Is(err, net.ErrClosed) {
		err = nil
	}
	return err
}

// Function to increment the connection count for an upstream server
func incrementConnectionCount(u *upstream) {
	u.load.Add(1)
}

// Function to decrement the connection count for an upstream server
func decrementConnectionCount(u *upstream) {
	u.load.Add(-1)
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
