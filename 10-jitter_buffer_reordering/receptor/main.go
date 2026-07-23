package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	udpAddr, err := net.ResolveUDPAddr("udp4", "localhost:8080")
	if err != nil {
		log.Fatalf("err resolving udp addr: %v", err)
	}

	udpConn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		log.Fatalf("err opening udp conn: %v", err)
	}
	defer udpConn.Close()

	log.Println("receptor up and running")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	receptor := NewReceptor(ctx, udpConn)
	player := NewPlayer()

	go func() {
		<-sigCh
		cancel()
		log.Println("bye")

		player.Report()
	}()

	go player.Bootstrap(receptor.Jb)
	go receptor.handleRead()

	<-ctx.Done()
}
