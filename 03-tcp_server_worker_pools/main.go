package main

import (
	"log"
	"time"
)

func main() {
	config := ServerConfig{
		Addr:        ":8080",
		Workers:     4,
		QueueSize:   10,
		IdleTimeout: 30 * time.Second,
	}

	server := NewTCPServer(config, EchoHandler)

	log.Printf("Starting TCP server on %s with %d workers, queue size %d\n",
		config.Addr, config.Workers, config.QueueSize)

	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}

func EchoHandler(conn *Connection) error {
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return err
		}

		if _, err := conn.Write(buf[:n]); err != nil {
			return err
		}
	}
}
