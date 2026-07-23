package main

import (
	"log"
	"net/http"
	"ws/server"
)

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

	if err := http.ListenAndServe(":6969", mux); err != nil {
		log.Fatalf("error listing to server: %v", err)
	}
}
