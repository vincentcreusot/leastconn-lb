package forwarder

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestForward(t *testing.T) {

	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "<html><body>server1</body></html>\n")
	}))
	defer srv1.Close()

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "<html><body>server2</body></html>\n")
	}))
	defer srv2.Close()

	servers := []string{srv1.Listener.Addr().String(), srv2.Listener.Addr().String()}

	f := NewForwarder(servers)

	// Forward
	lAddr, _ := net.ResolveTCPAddr("tcp", "localhost:0")
	listener, err := net.ListenTCP("tcp", lAddr)
	assert.NoError(t, err)
	errorsChan := make(chan []error, 1)
	go listenForTestRequest(f, listener, servers, errorsChan)
	// Write to client side
	resp, err := http.Get("http://" + listener.Addr().String() + "/anything")
	assert.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NotNil(t, body)
	assert.Equal(t, string(body), "<html><body>server1</body></html>\n")
	resp.Body.Close()
	errs := <-errorsChan
	assert.Equal(t, 0, len(errs))
	firstServerUpCount := f.upstreams[srv1.Listener.Addr().String()]
	assert.Equal(t, (int32)(0), firstServerUpCount.Load())

}

func listenForTestRequest(f *Forwarder, listener *net.TCPListener, urls []string, errorsChan chan []error) {

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err)
			continue
		}

		go f.Forward(clientConn.(*net.TCPConn), urls, errorsChan)
	}
}
