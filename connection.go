package bwqos

import (
	"context"
	"fmt"
	"math"
	"net"
	"time"

	"golang.org/x/time/rate"
)

// Conn is a net.Conn wrapper created for shaping of per connection and per listener traffic .
type Conn struct {
	listener *Listener
	limiter  *rate.Limiter

	net.Conn
}

// newConn creates new Conn .
// Also it creates new connection limiter and burns all initial burst tokens .
func newConn(conn net.Conn, listener *Listener) *Conn {
	connLimit := listener.connLimit
	localLimiter := rate.NewLimiter(rate.Limit(connLimit), connLimit)
	localLimiter.AllowN(time.Now(), connLimit) // to burn all tokens

	return &Conn{
		listener: listener,
		limiter:  localLimiter,
		Conn:     conn,
	}
}

// Write writes data to connection by chunks.
// Chunks size (take) is calculated according to the per connection limit, listener limit and connections count.
func (lc *Conn) Write(b []byte) (N int, err error) {
	for left := 0; left < len(b); {
		take := lc.getTake()

		right := left + take
		if right > len(b) {
			right = len(b)
		}
		take = right - left

		n, err := lc.Conn.Write(b[left:right])
		N = N + n
		if err != nil {
			return N, err
		}
		if err = lc.limiter.WaitN(context.Background(), take); err != nil {
			return N, err
		}
		if err = lc.listener.commonLimiter.WaitN(context.Background(), take); err != nil {
			return N, err
		}
		left = left + take
	}

	fmt.Println(N)
	return
}

// Close deletes Conn from connections list and closes underlying connection.
func (lc *Conn) Close() error {
	fmt.Println("conn closed")
	defer lc.listener.deleteConn(lc)
	return lc.Conn.Close()
}

// getTake returns number of bytes which will be written during next round during Write.
// It also checks if per connection limit was changed and updates limiter if necessary.
// "take" is calculated as min value between "connection limit" and "listener limit"/"connections count".
// It lets us to share per listener limit between all active connections.
func (lc *Conn) getTake() int {
	commonLimit := lc.listener.commonLimit
	connLimit := lc.listener.connLimit
	connNumber := lc.listener.getConnCount()
	// Every time to check if per connection bandwidth was changed
	if int(lc.limiter.Limit()) != connLimit {
		lc.limiter.SetLimit(rate.Limit(connLimit))
		lc.limiter.SetBurst(connLimit)
		lc.limiter.AllowN(time.Now(), connLimit) // to burn burst
	}

	take := int(math.Round(float64(commonLimit) / float64(connNumber)))
	if take > connLimit {
		take = connLimit
	}
	fmt.Println("take: ", take, "|connNumber: ", connNumber)
	return take
}
