package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestHealthCheck_BasicHealthy testa endpoint saudável
func TestHealthCheck_BasicHealthy(t *testing.T) {
	// Criar servidor fake que sempre retorna 200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("healthy"))
	}))
	defer server.Close()

	poller := NewHealthPoller()
	poller.AddEndpoint(EndpointConfig{
		URL:              server.URL,
		PollInterval:     100 * time.Millisecond,
		Timeout:          1 * time.Second,
		FailureThreshold: 3,
		RecoveryTimeout:  5 * time.Second,
	})

	poller.Start()
	defer poller.Stop()

	// Esperar alguns checks acontecerem
	time.Sleep(350 * time.Millisecond) // 3 checks

	status, ok := poller.GetStatus(server.URL)
	if !ok {
		t.Fatal("Status not found")
	}

	if !status.Healthy {
		t.Errorf("Expected healthy=true, got healthy=%v, error=%v", status.Healthy, status.Error)
	}

	if status.ConsecutiveFails != 0 {
		t.Errorf("Expected 0 consecutive fails, got %d", status.ConsecutiveFails)
	}

	if status.CircuitOpen {
		t.Error("Circuit should be closed for healthy endpoint")
	}
}

// TestHealthCheck_Unhealthy testa endpoint que falha
func TestHealthCheck_Unhealthy(t *testing.T) {
	// Servidor que sempre retorna 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	poller := NewHealthPoller()
	poller.AddEndpoint(EndpointConfig{
		URL:              server.URL,
		PollInterval:     100 * time.Millisecond,
		Timeout:          1 * time.Second,
		FailureThreshold: 3,
		RecoveryTimeout:  5 * time.Second,
	})

	poller.Start()
	defer poller.Stop()

	// Esperar alguns checks
	time.Sleep(150 * time.Millisecond)

	status, ok := poller.GetStatus(server.URL)
	if !ok {
		t.Fatal("Status not found")
	}

	if status.Healthy {
		t.Error("Expected unhealthy endpoint")
	}

	if status.ConsecutiveFails == 0 {
		t.Error("Expected consecutive fails > 0")
	}
}

// TestCircuitBreaker_Opens testa que circuit abre após threshold de falhas
func TestCircuitBreaker_Opens(t *testing.T) {
	// Servidor que sempre falha
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	poller := NewHealthPoller()
	poller.AddEndpoint(EndpointConfig{
		URL:              server.URL,
		PollInterval:     100 * time.Millisecond,
		Timeout:          1 * time.Second,
		FailureThreshold: 3,
		RecoveryTimeout:  5 * time.Second,
	})

	poller.Start()
	defer poller.Stop()

	// Esperar 4 checks (suficiente para abrir circuit após 3 falhas)
	time.Sleep(450 * time.Millisecond)

	status, ok := poller.GetStatus(server.URL)
	if !ok {
		t.Fatal("Status not found")
	}

	if !status.CircuitOpen {
		t.Errorf("Circuit should be open after %d consecutive fails (threshold=%d)",
			status.ConsecutiveFails, 3)
	}

	if status.ConsecutiveFails < 3 {
		t.Errorf("Expected at least 3 consecutive fails, got %d", status.ConsecutiveFails)
	}
}

// TestCircuitBreaker_Recovery testa que circuit fecha após recovery timeout
func TestCircuitBreaker_Recovery(t *testing.T) {
	requestCount := atomic.Int32{}

	// Servidor que falha primeiro, depois fica saudável
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)
		if count <= 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	poller := NewHealthPoller()
	poller.AddEndpoint(EndpointConfig{
		URL:              server.URL,
		PollInterval:     100 * time.Millisecond,
		Timeout:          1 * time.Second,
		FailureThreshold: 3,
		RecoveryTimeout:  500 * time.Millisecond, // Recovery curto para teste rápido
	})

	poller.Start()
	defer poller.Stop()

	// Esperar circuit abrir (3 falhas)
	time.Sleep(350 * time.Millisecond)

	status, _ := poller.GetStatus(server.URL)
	if !status.CircuitOpen {
		t.Error("Circuit should be open after initial failures")
	}

	// Esperar recovery timeout passar + alguns checks
	time.Sleep(800 * time.Millisecond)

	status, _ = poller.GetStatus(server.URL)

	// Após recovery, endpoint deveria estar healthy novamente
	if !status.Healthy {
		t.Errorf("Expected healthy after recovery, got healthy=%v", status.Healthy)
	}

	if status.ConsecutiveFails != 0 {
		t.Errorf("Expected 0 fails after recovery, got %d", status.ConsecutiveFails)
	}
}

func TestStatusChangeCallback(t *testing.T) {
	var mu sync.Mutex
	callbackCalls := []struct {
		oldStatus bool
		newStatus bool
	}{}

	requestCount := atomic.Int32{}

	// Servidor: primeiras 2 requisições OK, depois falha
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)
		if count <= 2 {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer server.Close()

	poller := NewHealthPoller()

	// Callback que guarda todas as transições
	poller.onStatusChange = func(endpoint string, oldStatus, newStatus bool) {
		mu.Lock()
		defer mu.Unlock()
		callbackCalls = append(callbackCalls, struct {
			oldStatus bool
			newStatus bool
		}{oldStatus, newStatus})
		t.Logf("Transition: old=%v new=%v", oldStatus, newStatus)
	}

	poller.AddEndpoint(EndpointConfig{
		URL:              server.URL,
		PollInterval:     100 * time.Millisecond,
		Timeout:          1 * time.Second,
		FailureThreshold: 1, // Threshold 1 para falhar mais rápido
		RecoveryTimeout:  5 * time.Second,
	})

	poller.Start()
	defer poller.Stop()

	// Esperar tempo suficiente para 4 checks acontecerem
	time.Sleep(450 * time.Millisecond)

	mu.Lock()
	calls := callbackCalls
	mu.Unlock()

	// Deve ter pelo menos 2 transições:
	// 1. false->true (primeiro check bem-sucedido)
	// 2. true->false (terceiro check falha)
	if len(calls) < 2 {
		t.Fatalf("Expected at least 2 transitions, got %d: %+v", len(calls), calls)
	}

	// Primeira transição deve ser false->true
	if calls[0].oldStatus != false || calls[0].newStatus != true {
		t.Errorf("First transition should be false->true, got %v->%v",
			calls[0].oldStatus, calls[0].newStatus)
	}

	// Segunda transição deve ser true->false
	if calls[1].oldStatus != true || calls[1].newStatus != false {
		t.Errorf("Second transition should be true->false, got %v->%v",
			calls[1].oldStatus, calls[1].newStatus)
	}
}

// TestMultipleEndpoints testa monitoramento de múltiplos endpoints
func TestMultipleEndpoints(t *testing.T) {
	// Servidor healthy
	healthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthyServer.Close()

	// Servidor unhealthy
	unhealthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer unhealthyServer.Close()

	poller := NewHealthPoller()

	poller.AddEndpoint(EndpointConfig{
		URL:              healthyServer.URL,
		PollInterval:     100 * time.Millisecond,
		Timeout:          1 * time.Second,
		FailureThreshold: 3,
		RecoveryTimeout:  5 * time.Second,
	})

	poller.AddEndpoint(EndpointConfig{
		URL:              unhealthyServer.URL,
		PollInterval:     100 * time.Millisecond,
		Timeout:          1 * time.Second,
		FailureThreshold: 3,
		RecoveryTimeout:  5 * time.Second,
	})

	poller.Start()
	defer poller.Stop()

	time.Sleep(250 * time.Millisecond)

	statuses := poller.GetAllStatuses()

	if len(statuses) != 2 {
		t.Fatalf("Expected 2 endpoints, got %d", len(statuses))
	}

	healthyStatus := statuses[healthyServer.URL]
	unhealthyStatus := statuses[unhealthyServer.URL]

	if !healthyStatus.Healthy {
		t.Errorf("Healthy server should be healthy, got %v", healthyStatus.Healthy)
	}

	if unhealthyStatus.Healthy {
		t.Errorf("Unhealthy server should be unhealthy, got %v", unhealthyStatus.Healthy)
	}
}

// TestGracefulShutdown testa que Stop() para todas as goroutines
func TestGracefulShutdown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	poller := NewHealthPoller()
	poller.AddEndpoint(EndpointConfig{
		URL:              server.URL,
		PollInterval:     100 * time.Millisecond,
		Timeout:          1 * time.Second,
		FailureThreshold: 3,
		RecoveryTimeout:  5 * time.Second,
	})

	poller.Start()

	// Deixar rodar um pouco
	time.Sleep(250 * time.Millisecond)

	// Stop deve retornar rapidamente
	done := make(chan bool)
	go func() {
		poller.Stop()
		done <- true
	}()

	select {
	case <-done:
		// Sucesso
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() timed out - goroutines not stopping")
	}
}

// TestTimeout testa que requests respeitam timeout
func TestTimeout(t *testing.T) {
	// Servidor que demora muito para responder
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Muito mais que o timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	poller := NewHealthPoller()
	poller.AddEndpoint(EndpointConfig{
		URL:              server.URL,
		PollInterval:     100 * time.Millisecond,
		Timeout:          100 * time.Millisecond, // Timeout curto
		FailureThreshold: 3,
		RecoveryTimeout:  5 * time.Second,
	})

	poller.Start()
	defer poller.Stop()

	// Esperar alguns checks
	time.Sleep(250 * time.Millisecond)

	status, ok := poller.GetStatus(server.URL)
	if !ok {
		t.Fatal("Status not found")
	}

	// Deve estar unhealthy por timeout
	if status.Healthy {
		t.Error("Expected unhealthy due to timeout")
	}

	if status.Error == nil {
		t.Error("Expected error to be set")
	}
}

// IMPORTANTE: Rodar com go test -race
func TestRaceConditions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	poller := NewHealthPoller()

	// Adicionar múltiplos endpoints
	for i := 0; i < 5; i++ {
		poller.AddEndpoint(EndpointConfig{
			URL:              fmt.Sprintf("%s/%d", server.URL, i),
			PollInterval:     50 * time.Millisecond,
			Timeout:          500 * time.Millisecond,
			FailureThreshold: 3,
			RecoveryTimeout:  5 * time.Second,
		})
	}

	poller.Start()

	// Enquanto polling acontece, ler statuses concorrentemente
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				poller.GetAllStatuses()
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()
	}

	// Esperar readers terminarem
	for i := 0; i < 10; i++ {
		<-done
	}

	poller.Stop()
}
