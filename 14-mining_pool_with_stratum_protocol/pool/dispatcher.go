package pool

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"miningpool/protocol"
)

// Dispatcher é responsável por gerar jobs de mineração e distribuí-los
// para todos os mineradores conectados via BroadcastJob do servidor.
// Ele é intencionalmente burro sobre o protocolo TCP — só conhece jobs.
type Dispatcher struct {
	server   *Server       // referência para chamar BroadcastJob
	interval time.Duration // intervalo entre novos jobs

	// currentDifficulty simula ajuste dinâmico de dificuldade.
	// Em Bitcoin real, a dificuldade é ajustada a cada 2016 blocos.
	currentDifficulty uint64
}

func NewDispatcher(server *Server, interval time.Duration, initialDifficulty uint64) *Dispatcher {
	return &Dispatcher{
		server:            server,
		interval:          interval,
		currentDifficulty: initialDifficulty,
	}
}

// Run inicia o loop do dispatcher. Deve ser chamado em uma goroutine separada.
// Para quando o context for cancelado — usa o mesmo padrão de shutdown do servidor.
func (d *Dispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			job := d.generateJob()
			d.server.BroadcastJob(job)
		}
	}
}

// generateJob cria um novo job de mineração com um JobID único e
// um PrevHash simulado. Em produção isso viria do nó Bitcoin completo.
func (d *Dispatcher) generateJob() *protocol.NotifyParams {
	return &protocol.NotifyParams{
		JobID:      fmt.Sprintf("job-%d", time.Now().UnixNano()),
		PrevHash:   fmt.Sprintf("%064x", rand.Uint64()), // hash simulado de 64 chars hex
		Difficulty: d.currentDifficulty,
		CleanJobs:  true, // descarta jobs anteriores — novo bloco chegou
	}
}
