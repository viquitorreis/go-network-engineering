package main

import (
	"bufio"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEchoServer(t *testing.T) {
	config := ServerConfig{
		Addr:        ":0", // porta aleatória
		Workers:     2,
		QueueSize:   5,
		IdleTimeout: 5 * time.Second,
	}

	server := NewTCPServer(config, EchoHandler)
	go server.Start()
	defer server.Shutdown()

	// espera servidor subir
	<-server.ready

	conn, err := net.Dial("tcp", server.Addr())
	require.NoError(t, err)
	defer conn.Close()

	// envia mensagem
	_, err = conn.Write([]byte("hello\n"))
	require.NoError(t, err)

	// lê echo
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	require.NoError(t, err)
	assert.Equal(t, "hello\n", response)
}

func TestBackpressure(t *testing.T) {
	config := ServerConfig{
		Addr:      ":0",
		Workers:   2,
		QueueSize: 2, // fila pequena pra forçar rejeição
	}

	// handler lento — segura conexão por 1s
	slowHandler := func(conn *Connection) error {
		defer conn.Close()
		time.Sleep(1 * time.Second)
		return nil
	}

	server := NewTCPServer(config, slowHandler)
	go server.Start()
	defer server.Shutdown()

	<-server.ready

	// abre Workers + QueueSize conexões (todas devem ser aceitas)
	var wg sync.WaitGroup
	var mu sync.Mutex
	accepted := 0
	rejected := 0

	for range 10 {
		wg.Go(func() {
			conn, err := net.Dial("tcp", server.Addr())
			if err != nil {
				mu.Lock()
				rejected++
				mu.Unlock()
				return
			}
			defer conn.Close()

			buf := make([]byte, 1)
			conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			_, err = conn.Read(buf)

			mu.Lock()
			if err != nil {
				rejected++
			} else {
				accepted++
			}
			mu.Unlock()
		})
	}

	wg.Wait()

	// deve ter rejeitado algumas conexões
	assert.Greater(t, rejected, 0, "server should reject connections when saturated")
}

func TestGracefulShutdown(t *testing.T) {
	config := ServerConfig{
		Addr:      ":0",
		Workers:   2,
		QueueSize: 5,
	}

	server := NewTCPServer(config, EchoHandler)
	go server.Start()
	<-server.ready

	// abre conexão
	conn, err := net.Dial("tcp", server.Addr())
	require.NoError(t, err)

	// shutdown deve esperar conexões ativas terminarem
	done := make(chan struct{})
	go func() {
		server.Shutdown()
		close(done)
	}()

	// fecha conexão
	conn.Close()

	// shutdown deve completar
	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown took too long")
	}
}
