package bwqos

import (
	"net"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Listener is a net.Listener wrapper
type Listener struct {
	mu       sync.RWMutex
	connList map[*Conn]struct{}

	commonLimit int
	connLimit   int

	commonLimiter *rate.Limiter

	net.Listener
}

// NewListener creates new Listener
func NewListener(listenerLimit, connLimit int) *Listener {
	return &Listener{
		commonLimit: listenerLimit,
		connLimit:   connLimit,
		connList:    make(map[*Conn]struct{}),
	}
}

// addConn adds Conn to the connections list
func (ll *Listener) addConn(conn *Conn) {
	ll.mu.Lock()
	defer ll.mu.Unlock()
	ll.connList[conn] = struct{}{}
}

// deleteConn deletes Conn from the connections list
func (ll *Listener) deleteConn(conn *Conn) {
	ll.mu.Lock()
	defer ll.mu.Unlock()
	delete(ll.connList, conn)
}

// getConnCount returns connections list count
func (ll *Listener) getConnCount() int {
	ll.mu.RLock()
	defer ll.mu.RUnlock()
	return len(ll.connList)
}

// Listen initializes Listener .
// This method accepts the same parameters as net.Listen()
func (ll *Listener) Listen(network, address string) (net.Listener, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		return l, err
	}
	ll.commonLimiter = rate.NewLimiter(rate.Limit(ll.commonLimit), ll.commonLimit)
	ll.commonLimiter.AllowN(time.Now(), ll.commonLimit) // to burn all tokens
	ll.Listener = l
	return ll, nil
}

// SetListenerLimit sets per server limit
func (ll *Listener) SetListenerLimit(n int) {
	ll.commonLimit = n
	ll.commonLimiter.SetLimit(rate.Limit(n))
	ll.commonLimiter.SetBurst(n)
}

// SetConnLimit sets per connection limit
func (ll *Listener) SetConnLimit(n int) {
	ll.connLimit = n
}

// Accept creates new Conn
func (ll *Listener) Accept() (net.Conn, error) {
	conn, err := ll.Listener.Accept()
	if err != nil {
		return conn, err
	}
	lconn := newConn(conn, ll)
	defer ll.addConn(lconn)
	return lconn, nil
}
