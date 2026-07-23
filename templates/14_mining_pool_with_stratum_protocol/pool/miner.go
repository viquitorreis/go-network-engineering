package pool

import (
	"bufio"
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
	// TODO: filtrar shares dentro da janela, somar dificuldades, dividir
	return 0
}

// SendMessage serializa e envia uma mensagem para o minerador.
// Usa bufio.Writer — precisa de Flush() depois ou o dado fica no buffer.
func (m *MinerState) SendMessage(v any) error {
	// TODO: json.Marshal, Write, '\n', Flush
	// Lembra: o Writer não é thread-safe sozinho — o mu protege
	return nil
}
