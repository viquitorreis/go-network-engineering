package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

func main() {
	poller := NewHealthPoller()

	// Configurar callback para notificações
	poller.onStatusChange = func(endpoint string, oldStatus, newStatus bool) {
		if newStatus && !oldStatus {
			fmt.Printf("%s is now HEALTHY\n", endpoint)
		} else if !newStatus && oldStatus {
			fmt.Printf("%s is now UNHEALTHY\n", endpoint)
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
	fmt.Println("=== Final Status ===")
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
	hp.mu.Lock()
	defer hp.mu.Unlock()

	if _, ok := hp.endpoints[config.URL]; !ok {
		hp.endpoints[config.URL] = &config
		hp.statuses[config.URL] = &HealthStatus{
			Endpoint: config.URL,
		}
	}
}

// Start inicia polling de todos os endpoints
func (hp *HealthPoller) Start() {
	hp.wg.Add(1)
	go hp.aggregateResults()

	for _, endpointCfg := range hp.endpoints {
		hp.wg.Add(1)
		go hp.pollEndpoint(*endpointCfg)
	}
}

// pollEndpoint faz health check periódico de um endpoint
func (hp *HealthPoller) pollEndpoint(config EndpointConfig) {
	defer hp.wg.Done()

	ticker := time.NewTicker(config.PollInterval)
	defer ticker.Stop()

	var circuitOpenedAt time.Time

	for {
		select {
		case <-ticker.C:
			hp.mu.RLock()
			currentStatus := hp.statuses[config.URL]
			hp.mu.RUnlock()

			// quando circuit esta aberto, checamos se ja podemos tentar novamente
			if currentStatus.CircuitOpen {
				if time.Since(circuitOpenedAt) < config.RecoveryTimeout {
					continue
				}
			}

			status := hp.checkEndpoint(config)

			// acabou de abrir
			if status.CircuitOpen && !currentStatus.CircuitOpen {
				circuitOpenedAt = time.Now()
			}

			// envia resultado para o agregador
			select {
			case hp.results <- status:
			case <-hp.ctx.Done():
				return
			}

		case <-hp.ctx.Done():
			return
		}
	}
}

// checkEndpoint faz um health check
func (hp *HealthPoller) checkEndpoint(config EndpointConfig) HealthStatus {
	client := http.Client{
		Timeout: config.Timeout,
	}

	resp, err := client.Get(config.URL)

	status := HealthStatus{
		Endpoint:      config.URL,
		LastCheckTime: time.Now(),
	}

	if err != nil {
		status.Healthy = false
		status.Error = err
	} else {
		defer resp.Body.Close()
		status.Healthy = resp.StatusCode >= 200 && resp.StatusCode < 300
		if !status.Healthy {
			status.Error = fmt.Errorf("status code: %d", resp.StatusCode)
		}
	}

	hp.mu.Lock()
	defer hp.mu.Unlock()

	old := hp.statuses[config.URL]

	if status.Healthy {
		status.ConsecutiveFails = 0
	} else {
		status.ConsecutiveFails = old.ConsecutiveFails + 1
	}

	if status.ConsecutiveFails >= config.FailureThreshold {
		status.CircuitOpen = true
	}

	return status
}

// aggregateResults processa resultados e detecta mudanças de status
func (hp *HealthPoller) aggregateResults() {
	defer hp.wg.Done()

	for {
		select {
		case status := <-hp.results:
			log.Printf("[DEBUG] Aggregator received status for %s: healthy=%v\n",
				status.Endpoint, status.Healthy)

			hp.mu.Lock()

			old := hp.statuses[status.Endpoint]
			oldHealthy := old.Healthy
			hp.statuses[status.Endpoint] = &status

			hp.mu.Unlock()

			fmt.Printf("[DEBUG] Old healthy=%v, New healthy=%v\n", oldHealthy, status.Healthy)

			if hp.onStatusChange != nil && oldHealthy != status.Healthy {
				hp.onStatusChange(status.Endpoint, oldHealthy, status.Healthy)
			}

		case <-hp.ctx.Done():
			fmt.Println("ctx closed")
			return
		}
	}
}

// GetStatus retorna status atual de um endpoint
func (hp *HealthPoller) GetStatus(endpoint string) (HealthStatus, bool) {
	hp.mu.RLock()
	defer hp.mu.RUnlock()
	return *hp.statuses[endpoint], true
}

// GetAllStatuses retorna status de todos os endpoints
func (hp *HealthPoller) GetAllStatuses() map[string]*HealthStatus {
	hp.mu.RLock()
	defer hp.mu.RUnlock()
	return hp.statuses
}

// Stop para todos os health checks
func (hp *HealthPoller) Stop() {
	hp.cancel()
	hp.wg.Wait()
	close(hp.results)
}
