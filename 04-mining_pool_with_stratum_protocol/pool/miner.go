package pool

import (
	"bufio"
	"encoding/json"
	"net"
	"sync"
	"time"
)

// MinerState representa o estado de um minerador conectado.
// Cada conexão TCP tem exatamente um MinerState.
type MinerState struct {
	mu sync.RWMutex

	ID        string
	Conn      net.Conn
	Writer    *bufio.Writer // buffered para não fazer syscall por mensagem
	SessionID string

	// Hashrate tracking: usamos uma janela deslizante de 60s.
	// shares aceitas nos últimos 60s * dificuldade média = hashrate estimado.
	ShareWindow []shareEntry
	Difficulty  uint64 // dificuldade atual atribuída a esse minerador

	CurrentJobID string
	ConnectedAt  time.Time
	LastShareAt  time.Time
}

type shareEntry struct {
	At         time.Time
	Difficulty uint64
}

// HashrateTHs retorna o hashrate estimado em TH/s nos últimos windowSec segundos.
// Fórmula: sum(difficulty_of_each_share) / window_seconds / 1e12
func (m *MinerState) HashrateTHs(windowSec int) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// cutoff é o o ínicio da janela, so consideramos shares depois disso
	cutoff := time.Now().Add(-time.Duration(windowSec) * time.Second)
	var total uint64
	for _, share := range m.ShareWindow {
		if share.At.After(cutoff) {
			total += share.Difficulty
		}
	}

	if total == 0 {
		return 0
	}

	// / 1e12 para converter de H/s para TH/s (unidade de hashes por segundo para terahashes por segundo, 1 TH/s é 10¹² hashes)
	return float64(total) / float64(windowSec) / 1e12
}

func (m *MinerState) SendMessage(v any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// NewEncoder.Encode ja adiciona o newline, então não precisa escrever manualmente
	if err := json.NewEncoder(m.Writer).Encode(v); err != nil {
		return err
	}

	return m.Writer.Flush()
}
