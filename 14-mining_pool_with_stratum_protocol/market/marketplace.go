package market

import (
	"log/slog"
	"sync"
)

// Marketplace implementa HashrateMarket com modelo de preço fixo.
// Mineradores registram hashrate, recebem pagamento proporcional ao total.
// É o modelo mais simples: sem order book, sem matching.
type Marketplace struct {
	mu sync.RWMutex

	pricePerTH float64 // USD por TH/s por hora
	miners     map[string]*minerRecord
}

type minerRecord struct {
	HashrateTHs float64
	EarnedUSD   float64
}

func NewMarketplace(pricePerTH float64) *Marketplace {
	return &Marketplace{
		pricePerTH: pricePerTH,
		miners:     make(map[string]*minerRecord),
	}
}

func (m *Marketplace) RegisterMiner(minerID string, hashrateTHs float64) error {
	m.mu.Lock()
	slog.Info("RegisterMiner called", "minerID", minerID, "hashrate", hashrateTHs)

	if val, ok := m.miners[minerID]; ok {
		m.miners[minerID] = &minerRecord{
			HashrateTHs: hashrateTHs,
			EarnedUSD:   val.EarnedUSD,
		}
	} else {
		m.miners[minerID] = &minerRecord{
			HashrateTHs: hashrateTHs,
			EarnedUSD:   0,
		}
	}
	m.mu.Unlock()
	return nil
}

func (m *Marketplace) UnregisterMiner(minerID string) {
	m.mu.Lock()
	delete(m.miners, minerID)
	m.mu.Unlock()
}

func (m *Marketplace) TotalHashrate() float64 {
	total := 0.0
	m.mu.RLock()
	for _, record := range m.miners {
		total += record.HashrateTHs
	}
	m.mu.RUnlock()
	return total
}

func (m *Marketplace) Revenue(minerID string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if record, ok := m.miners[minerID]; ok {
		return record.EarnedUSD
	}
	return 0
}

// Tick é chamado periodicamente (ex: a cada minuto) para calcular e distribuir revenue.
// Revenue de cada minerador = seu hashrate * pricePerTH * totalHashrate
func (m *Marketplace) Tick(durationHours float64) {
	// TODO: Lock, calcular proporção de cada minerador, atualizar EarnedUSD
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, record := range m.miners {
		record.EarnedUSD += record.HashrateTHs * m.pricePerTH * durationHours
	}
}
