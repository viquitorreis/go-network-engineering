package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatalf("err connecting dialer: %v", err)
	}
	defer conn.Close()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go writeFrame(ctx, &conn)
	go readFrame(ctx, &conn)

	log.Println("client connected")

	<-sigCh
	cancel()
}

func writeFrame(ctx context.Context, c *net.Conn) {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		payload := []byte(scanner.Text())
		sizeBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(sizeBuf, uint32(len(payload)))

		if _, err := (*c).Write(sizeBuf); err != nil {
			log.Printf("err writing: %v", err)
			return
		}
		if _, err := (*c).Write(payload); err != nil {
			log.Printf("err writing: %v", err)
			return
		}
	}
}

func readFrame(ctx context.Context, c *net.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		buf := make([]byte, 4)
		_, err := io.ReadFull(*c, buf)
		if err != nil {
			log.Printf("err reading: %v", err)
			return
		}

		size := binary.BigEndian.Uint32(buf)

		payload := make([]byte, size)
		_, err = io.ReadFull(*c, payload)
		if err != nil {
			log.Printf("err reading: %v", err)
		}

		fmt.Println("recebido:", string(payload))
	}

}
