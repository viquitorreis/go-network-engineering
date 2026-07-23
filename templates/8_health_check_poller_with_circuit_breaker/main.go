package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

/*
TODO - HEALTH CHECK POLLER

1. CRIAR ENDPOINT CONFIG
   - URL do endpoint
   - Interval de polling (quanto tempo entre checks)
   - Timeout do request
   - Threshold de falhas para circuit breaker abrir

2. IMPLEMENTAR HEALTH CHECKER
   - Goroutine por endpoint fazendo polling periódico
   - HTTP GET com timeout configurável
   - Classificar response: healthy (2xx), unhealthy (outros), error (timeout/network)

3. CIRCUIT BREAKER
   - Contar falhas consecutivas
   - Abrir circuit após N falhas (para de fazer requests)
   - Após X tempo, tentar novamente (half-open)
   - Se suceder, fechar circuit (volta ao normal)

4. AGREGAÇÃO DE STATUS
   - Channel para resultados de health checks
   - Goroutine agregadora mantém mapa de status atual
   - Detectar transições (healthy->unhealthy, unhealthy->healthy)
   - Notificar via callback quando status muda

5. GRACEFUL SHUTDOWN
   - Context para cancelar todas as goroutines de polling
   - Esperar todas terminarem antes de sair
*/

func main() {
	poller := NewHealthPoller()

	// Configurar callback para notificações
	poller.onStatusChange = func(endpoint string, oldStatus, newStatus bool) {
		if newStatus && !oldStatus {
			fmt.Printf("- %s is now HEALTHY\n", endpoint)
		} else if !newStatus && oldStatus {
			fmt.Printf("❌ %s is now UNHEALTHY\n", endpoint)
		}
	}

	// Adicionar endpoints para monitorar
	poller.AddEndpoint(EndpointConfig{
		URL:              "https://www.google.com",
		PollInterval:     2 * time.Second,
		Timeout:          1 * time.Second,
		FailureThreshold: 3,
		RecoveryTimeout:  10 * time.Second,
	})

	poller.AddEndpoint(EndpointConfig{
		URL:              "http://localhost:8080/health", // Vai falhar
		PollInterval:     2 * time.Second,
		Timeout:          500 * time.Millisecond,
		FailureThreshold: 3,
		RecoveryTimeout:  10 * time.Second,
	})

	poller.Start()

	// Rodar por 30 segundos
	time.Sleep(30 * time.Second)

	// Mostrar status final
	fmt.Println("\n=== Final Status ===")
	for endpoint, status := range poller.GetAllStatuses() {
		healthy := "HEALTHY"
		if !status.Healthy {
			healthy = "UNHEALTHY"
		}
		circuit := ""
		if status.CircuitOpen {
			circuit = " [CIRCUIT OPEN]"
		}
		fmt.Printf("%s: %s (fails: %d)%s\n",
			endpoint, healthy, status.ConsecutiveFails, circuit)
	}

	poller.Stop()
	fmt.Println("Done!")
}

// EndpointConfig define configuração de um endpoint monitorado
type EndpointConfig struct {
	URL              string
	PollInterval     time.Duration
	Timeout          time.Duration
	FailureThreshold int           // Falhas consecutivas para abrir circuit
	RecoveryTimeout  time.Duration // Quanto tempo esperar antes de retry quando circuit aberto
}

// HealthStatus representa estado de saúde de um endpoint
type HealthStatus struct {
	Endpoint         string
	Healthy          bool
	LastCheckTime    time.Time
	ConsecutiveFails int
	CircuitOpen      bool
	Error            error
}

// HealthPoller gerencia health checks de múltiplos endpoints
type HealthPoller struct {
	endpoints map[string]*EndpointConfig
	statuses  map[string]*HealthStatus
	mu        sync.RWMutex

	results chan HealthStatus
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	// Callback chamado quando status de endpoint muda
	onStatusChange func(endpoint string, oldStatus, newStatus bool)
}

// NewHealthPoller cria novo poller
func NewHealthPoller() *HealthPoller {
	ctx, cancel := context.WithCancel(context.Background())
	return &HealthPoller{
		endpoints: make(map[string]*EndpointConfig),
		statuses:  make(map[string]*HealthStatus),
		results:   make(chan HealthStatus, 100),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// AddEndpoint adiciona endpoint para monitorar
func (hp *HealthPoller) AddEndpoint(config EndpointConfig) {
	// TODO: implementar
	// Adicionar no map de endpoints
	// Inicializar status
}

// Start inicia polling de todos os endpoints
func (hp *HealthPoller) Start() {
	// TODO: implementar
	// Iniciar goroutine agregadora (processa results channel)
	// Para cada endpoint, iniciar goroutine de polling
}

// pollEndpoint faz health check periódico de um endpoint
func (hp *HealthPoller) pollEndpoint(config EndpointConfig) {
	// TODO: implementar
	// Loop com ticker no PollInterval
	// A cada tick:
	//   - Se circuit aberto, verificar se pode tentar (RecoveryTimeout passou)
	//   - Fazer HTTP GET com timeout
	//   - Avaliar response (2xx = healthy)
	//   - Atualizar circuit breaker state
	//   - Enviar resultado para results channel
}

// checkEndpoint faz um health check
func (hp *HealthPoller) checkEndpoint(config EndpointConfig) HealthStatus {
	// TODO: implementar
	// Criar HTTP client com timeout
	// GET no endpoint
	// Retornar HealthStatus baseado em response
	return HealthStatus{}
}

// aggregateResults processa resultados e detecta mudanças de status
func (hp *HealthPoller) aggregateResults() {
	// TODO: implementar
	// Loop recebendo do results channel
	// Atualizar statuses map
	// Detectar transições (healthy->unhealthy ou vice-versa)
	// Chamar callback se status mudou
}

// GetStatus retorna status atual de um endpoint
func (hp *HealthPoller) GetStatus(endpoint string) (HealthStatus, bool) {
	// TODO: implementar com read lock
	return HealthStatus{}, false
}

// GetAllStatuses retorna status de todos os endpoints
func (hp *HealthPoller) GetAllStatuses() map[string]HealthStatus {
	// TODO: implementar com read lock
	return nil
}

// Stop para todos os health checks
func (hp *HealthPoller) Stop() {
	// TODO: implementar
	// Cancelar context
	// Esperar WaitGroup
	// Fechar results channel
}
