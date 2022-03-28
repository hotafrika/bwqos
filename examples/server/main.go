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
	n, err := w.Write(b)
	fmt.Println(n, err)
}
