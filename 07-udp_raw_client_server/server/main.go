package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"net"
)

func main() {
	s, err := net.ResolveUDPAddr("udp4", ":8080")
	if err != nil {
		fmt.Println(err)
		return
	}

	udpConn, err := net.ListenUDP("udp4", s)
	if err != nil {
		log.Fatalf("error serving udp: %v", err)
	}
	defer udpConn.Close()

	log.Println("up and running")

	server := NewServer()

	for {
		buf := make([]byte, 512)
		n, addr, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			log.Println("err reading from client", err)
		}

		server.addClient(*addr)

		if rand.Float64() < 0.1 {
			continue
		}

		udpConn.WriteToUDP(buf[:n], addr)
		server.totalMsg++
	}
}

type Server struct {
	// string = client ip
	Clients  map[string]bool
	totalMsg int
}

func NewServer() *Server {
	return &Server{
		Clients: map[string]bool{},
	}
}

func (s *Server) addClient(addr net.UDPAddr) {
	if _, ok := s.Clients[addr.String()]; ok {
		return
	}

	s.Clients[addr.String()] = true
}
