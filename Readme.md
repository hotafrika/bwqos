This repo implements custom wrappers for net.Listener and net.Conn which allows you to set up traffic limit per server (per listener) and per connection.

Under the hood wrappers use rate.Limiter's "token bucket" mechanism from golang.org/x/time package. See https://en.wikipedia.org/wiki/Token_bucket for more about token buckets.

---

To create limited listener you can use:
```go
func Listen(network, address string, listenerLimit, connLimit int) (net.Listener, error)
```
or in two steps:
```go
func NewListener(listenerLimit, connLimit int) *Listener

func (ll *Listener) Listen(network, address string) (net.Listener, error)
```

---

You can change limits (per listener and per connection) in runtime using these methods:
```go
func (ll *Listener) SetListenerLimit(n int)

func (ll *Listener) SetConnLimit(n int)
```
Changes for every connection will be applied during next `Write()` operation.

---

Example usage for http server:
```go
package main

import (
	"fmt"
	"net/http"
	"github.com/hotafrika/bwqos"
)

func main() {
	listenerLimit := 100_000
	connLimit := 10_000
	l, err := bwqos.Listen("tcp", ":8080", listenerLimit, connLimit)
	if err != nil {
		panic(err)
	}

	err = http.Serve(l, http.HandlerFunc(LoadFile))
	if err != nil {
		panic(err)
	}
}

func LoadFile(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 1_000_000)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `inline; filename="myfile.txt"`)
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

```