package bwqos

import (
	"net"
	"time"

	"golang.org/x/time/rate"
)

// Listen announces on the local network address. Under the hood it uses traffic shaping Listener .
func Listen(network, address string, listenerLimit, connLimit int) (net.Listener, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		return l, err
	}
	ll := &Listener{
		commonLimit: listenerLimit,
		connLimit:   connLimit,
		connList:    make(map[*Conn]struct{}),
	}
	ll.commonLimiter = rate.NewLimiter(rate.Limit(ll.commonLimit), ll.commonLimit)
	ll.commonLimiter.AllowN(time.Now(), ll.commonLimit) // to burn all tokens
	ll.Listener = l
	return ll, nil
}
