package main

import (
	"log"
	"net"
	"net/http"
	"ws/server"
)

// noDelayListener is a wrapper around net.Listener that sets TCP_NODELAY to true for accepted connections
// it will disable Nagle's algorithm for low-latency communication before any I/O operations are performed on the connection
//
//	including handshake and read/write operations
type noDelayListener struct {
	net.Listener
}

func main() {
	mux := http.NewServeMux()

	hub := server.NewHub()

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		room := r.URL.Query().Get("room")
		if room == "" {
			room = "default"
		}

		server.ServeWs(hub, room, w, r)
	})

	ln, err := net.Listen("tcp", ":6969")
	if err != nil {
		log.Fatalf("error listening: %v", err)
	}

	srv := &http.Server{
		Handler: mux,
	}

	if err := srv.Serve(noDelayListener{ln}); err != nil {
		log.Fatalf("error serving: %v", err)
	}
}

func (l noDelayListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	if tc, ok := conn.(*net.TCPConn); ok {
		tc.SetNoDelay(true)
	}

	return conn, nil
}
