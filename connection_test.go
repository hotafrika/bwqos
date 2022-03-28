package bwqos

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestConn_getTake(t *testing.T) {
	type fields struct {
		commonLimit int
		connLimit   int
		connCount   int
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "conn",
			fields: fields{
				commonLimit: 100,
				connLimit:   50,
				connCount:   1,
			},
			want: 50,
		},
		{
			name: "common",
			fields: fields{
				commonLimit: 50,
				connLimit:   100,
				connCount:   1,
			},
			want: 50,
		},
		{
			name: "common share",
			fields: fields{
				commonLimit: 100,
				connLimit:   50,
				connCount:   1,
			},
			want: 50,
		},
		{
			name: "equal",
			fields: fields{
				commonLimit: 100,
				connLimit:   50,
				connCount:   2,
			},
			want: 50,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewListener(tt.fields.commonLimit, tt.fields.connLimit)
			for i := 0; i < tt.fields.connCount; i++ {
				c := Conn{}
				l.connList[&c] = struct{}{}
			}
			lc := &Conn{
				listener: l,
				limiter:  rate.NewLimiter(rate.Limit(tt.fields.connLimit), tt.fields.connLimit),
			}
			if got := lc.getTake(); got != tt.want {
				t.Errorf("getTake() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConn_Write(t *testing.T) {
	type fields struct {
		fileSize    int
		commonLimit int
		connLimit   int
		connCount   int
	}
	tests := []struct {
		name         string
		fields       fields
		wantDuration time.Duration
	}{
		{
			name: "single 1",
			fields: fields{
				fileSize:    1000,
				commonLimit: 300,
				connLimit:   200,
				connCount:   1,
			},
			wantDuration: 5 * time.Second,
		},
		{
			name: "dual 1",
			fields: fields{
				fileSize:    1000,
				commonLimit: 400,
				connLimit:   500,
				connCount:   2,
			},
			wantDuration: 5 * time.Second,
		},
		{
			name: "long 1",
			fields: fields{
				fileSize:    300_000,
				commonLimit: 10_000,
				connLimit:   10_000,
				connCount:   1,
			},
			wantDuration: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handleFunc := func(w http.ResponseWriter, r *http.Request) {
				b := make([]byte, tt.fields.fileSize)
				w.Write(b)
			}
			l, err := Listen("tcp", "127.0.0.1:0", tt.fields.commonLimit, tt.fields.connLimit)
			if err != nil {
				t.Errorf("can't create listener for %s ", tt.name)
			}
			server := httptest.Server{
				Listener: l,
				Config: &http.Server{
					Handler:     http.HandlerFunc(handleFunc),
					ReadTimeout: time.Minute,
				},
			}

			server.Start()
			defer server.Close()

			for i := 0; i < tt.fields.connCount-1; i++ {
				go func() {
					resp, err := http.Get(server.URL)
					if err != nil {
						return
					}
					_, err = io.ReadAll(resp.Body)
					if err != nil {
						return
					}
					defer resp.Body.Close()
				}()
			}
			start := time.Now()
			resp, err := http.Get(server.URL)
			if err != nil {
				t.Errorf("can't execute GET for %s ", tt.name)
			}
			_, err = io.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("can't read body %s ", tt.name)
			}
			duration := time.Since(start)
			defer resp.Body.Close()

			low := float64(tt.wantDuration) * 0.95
			high := float64(tt.wantDuration) * 1.05

			if float64(duration) < low || float64(duration) > high {
				t.Errorf("test execution is not in range; duration = %d, want %d", duration, tt.wantDuration)
			}
		})
	}
}
