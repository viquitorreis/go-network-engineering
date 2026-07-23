package market

import (
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
	// TODO
	return nil
}

func (m *Marketplace) RegisterMiner(minerID string, hashrateTHs float64) error {
	// TODO
	return nil
}

func (m *Marketplace) UnregisterMiner(minerID string) {
	// TODO
}

func (m *Marketplace) TotalHashrate() float64 {
	// TODO: RLock, somar hashrates
	return 0
}

func (m *Marketplace) Revenue(minerID string) float64 {
	// TODO
	return 0
}

// Tick é chamado periodicamente (ex: a cada minuto) para calcular e distribuir revenue.
// Revenue de cada minerador = (seu hashrate / total hashrate) * pricePerTH * totalHashrate * (1/60 hora)
func (m *Marketplace) Tick(durationHours float64) {
	// TODO: Lock, calcular proporção de cada minerador, atualizar EarnedUSD
}
