package market

import (
	"sync"
	"time"
)

// Order representa uma oferta no order book de hashrate.
type Order struct {
	ID          string
	MinerID     string  // quem está vendendo hashrate
	HashrateTHs float64 // quantidade ofertada
	PricePerTH  float64 // preço pedido (ask) ou máximo (bid)
	Side        OrderSide
	Cancelled   bool
	CreatedAt   time.Time
}

type OrderSide int

const (
	Bid OrderSide = iota // comprador: "quero comprar X TH/s por até $Y"
	Ask                  // vendedor (minerador): "tenho X TH/s por $Y"
)

// Exchange implementa HashrateMarket com order book.
// A lógica de matching é price-time priority, igual ao Challenge 11,
// mas aqui o "ativo" é hashrate em vez de um par de moedas.
type Exchange struct {
	mu sync.Mutex

	asks *askHeap // sorted by price ascending  (vendedores mais baratos primeiro)
	bids *bidHeap // sorted by price descending (compradores que pagam mais primeiro)

	orders  map[string]*Order  // orderID -> order (lookup rápido para UnregisterMiner)
	revenue map[string]float64 // minerID -> USD acumulado
}

func NewExchange() *Exchange {
	// TODO
	return nil
}

// PlaceOrder adiciona um bid ou ask no order book e tenta fazer matching.
func (e *Exchange) PlaceOrder(order *Order) error {
	// TODO: adicionar no lado correto, chamar tryMatch
	return nil
}

func (e *Exchange) tryMatch() {
	// TODO: enquanto melhor ask <= melhor bid, fazer match e distribuir revenue
	// Dica: pensa no que você fez no Challenge 11 — a lógica é a mesma,
	// só muda o que você está "comprando"
}

// RegisterMiner no contexto da exchange significa: colocar um Ask automaticamente.
func (e *Exchange) RegisterMiner(minerID string, hashrateTHs float64) error {
	// TODO: criar um Ask com preço padrão e chamar PlaceOrder
	return nil
}

func (e *Exchange) UnregisterMiner(minerID string) {
	// TODO: remover asks pendentes desse minerador
}

func (e *Exchange) TotalHashrate() float64 {
	// TODO
	return 0
}

func (e *Exchange) Revenue(minerID string) float64 {
	// TODO
	return 0
}
