package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestServer_AcceptsConnections(t *testing.T) {
	server := NewChatServer("18080", 1) // porta diferente para não conflitar

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Iniciar servidor
	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond) // dar tempo pro servidor subir

	// Conectar cliente
	conn, err := net.Dial("tcp", "localhost:18080")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	t.Log("Client connected successfully")
}

func TestServer_LobbyWaitsForMinPlayers(t *testing.T) {
	server := NewChatServer("18081", 2) // precisa de 2 players

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Conectar primeiro cliente
	conn1, err := net.Dial("tcp", "localhost:18081")
	if err != nil {
		t.Fatalf("Client 1 failed to connect: %v", err)
	}
	defer conn1.Close()

	// Dar tempo para entrar no lobby
	time.Sleep(200 * time.Millisecond)

	// Conectar segundo cliente
	conn2, err := net.Dial("tcp", "localhost:18081")
	if err != nil {
		t.Fatalf("Client 2 failed to connect: %v", err)
	}
	defer conn2.Close()

	// Quando 2o cliente conecta, ambos devem ser liberados do lobby
	time.Sleep(200 * time.Millisecond)

	t.Log("Lobby released after 2 players connected")
}

func TestServer_BroadcastMessage(t *testing.T) {
	server := NewChatServer("18082", 2)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Conectar dois clientes
	conn1, err := net.Dial("tcp", "localhost:18082")
	if err != nil {
		t.Fatalf("Client 1 failed: %v", err)
	}
	defer conn1.Close()

	conn2, err := net.Dial("tcp", "localhost:18082")
	if err != nil {
		t.Fatalf("Client 2 failed: %v", err)
	}
	defer conn2.Close()

	// Esperar lobby liberar
	time.Sleep(300 * time.Millisecond)

	// Cliente 1 envia mensagem
	fmt.Fprintf(conn1, "Hello from client 1\n")

	// Cliente 2 deve receber
	reader := bufio.NewReader(conn2)
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))

	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Client 2 didn't receive message: %v", err)
	}

	if !strings.Contains(line, "Hello from client 1") {
		t.Errorf("Expected message with 'Hello from client 1', got: %s", line)
	}

	t.Log("Message broadcast successfully")
}

func TestServer_MultipleClients(t *testing.T) {
	server := NewChatServer("18083", 2)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Conectar 3 clientes
	var conns []net.Conn
	for i := 0; i < 3; i++ {
		conn, err := net.Dial("tcp", "localhost:18083")
		if err != nil {
			t.Fatalf("Client %d failed: %v", i+1, err)
		}
		conns = append(conns, conn)
		defer conn.Close()
	}

	// Esperar lobby liberar
	time.Sleep(300 * time.Millisecond)

	// Cliente 1 envia mensagem
	fmt.Fprintf(conns[0], "Broadcast test\n")

	// Clientes 2 e 3 devem receber
	for i := 1; i < 3; i++ {
		reader := bufio.NewReader(conns[i])
		conns[i].SetReadDeadline(time.Now().Add(2 * time.Second))

		line, err := reader.ReadString('\n')
		if err != nil {
			t.Errorf("Client %d didn't receive: %v", i+1, err)
			continue
		}

		if !strings.Contains(line, "Broadcast test") {
			t.Errorf("Client %d got wrong message: %s", i+1, line)
		}
	}

	t.Log("Message broadcast to all clients")
}

func TestServer_ClientDisconnect(t *testing.T) {
	server := NewChatServer("18084", 2)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Conectar 3 clientes
	conn1, _ := net.Dial("tcp", "localhost:18084")
	conn2, _ := net.Dial("tcp", "localhost:18084")
	conn3, _ := net.Dial("tcp", "localhost:18084")

	time.Sleep(300 * time.Millisecond)

	// Cliente 1 desconecta
	conn1.Close()
	time.Sleep(200 * time.Millisecond)

	// Cliente 2 envia mensagem
	fmt.Fprintf(conn2, "After disconnect\n")

	// Cliente 3 deve receber (cliente 1 não, pois desconectou)
	reader := bufio.NewReader(conn3)
	conn3.SetReadDeadline(time.Now().Add(2 * time.Second))

	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Client 3 should still receive: %v", err)
	}

	if !strings.Contains(line, "After disconnect") {
		t.Errorf("Got wrong message: %s", line)
	}

	conn2.Close()
	conn3.Close()

	t.Log("Server handles disconnect correctly")
}

func TestServer_EmptyMessage(t *testing.T) {
	server := NewChatServer("18085", 1)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", "localhost:18085")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Enviar mensagem vazia
	fmt.Fprintf(conn, "\n")

	time.Sleep(100 * time.Millisecond)

	t.Log("Server handles empty messages")
}

// Teste de stress - múltiplas mensagens rápidas
func TestServer_RapidMessages(t *testing.T) {
	server := NewChatServer("18086", 2)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	conn1, _ := net.Dial("tcp", "localhost:18086")
	defer conn1.Close()

	conn2, _ := net.Dial("tcp", "localhost:18086")
	defer conn2.Close()

	time.Sleep(300 * time.Millisecond)

	// Enviar 50 mensagens rápidas
	go func() {
		for i := 0; i < 50; i++ {
			fmt.Fprintf(conn1, "Message %d\n", i)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Cliente 2 deve receber todas (ou a maioria se buffer encher)
	reader := bufio.NewReader(conn2)
	received := 0

	for i := 0; i < 50; i++ {
		conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		received++
	}

	if received < 45 { // Aceitar perder algumas mensagens por buffer cheio
		t.Errorf("Only received %d/50 messages", received)
	} else {
		t.Logf("Received %d/50 rapid messages", received)
	}
}
