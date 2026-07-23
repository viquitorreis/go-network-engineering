package market

import (
	"container/heap"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
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

const defaultAskPrice = 0.05

const (
	Bid OrderSide = iota // comprador: "quero comprar X TH/s por até $Y"
	Ask                  // vendedor (minerador): "tenho X TH/s por $Y"
)

// Exchange implementa HashrateMarket com order book.
// A lógica de matching é price-time priority, igual ao Challenge 11,
// mas aqui o "ativo" é hashrate em vez de um par de moedas.
type Exchange struct {
	mu sync.RWMutex

	asks *askHeap // sorted by price ascending  (vendedores mais baratos primeiro)
	bids *bidHeap // sorted by price descending (compradores que pagam mais primeiro)

	orders  map[string]*Order  // orderID -> order (lookup rápido para UnregisterMiner)
	revenue map[string]float64 // minerID -> USD acumulado
}

func NewExchange() *Exchange {
	ah := &askHeap{}
	bh := &bidHeap{}
	heap.Init(ah)
	heap.Init(bh)

	return &Exchange{
		asks:    ah,
		bids:    bh,
		orders:  make(map[string]*Order),
		revenue: make(map[string]float64),
	}
}

// PlaceOrder adiciona um bid ou ask no order book e tenta fazer matching.
func (e *Exchange) PlaceOrder(order *Order) error {
	// TODO: adicionar no lado correto, chamar tryMatch
	e.mu.Lock()
	defer e.mu.Unlock()
	if order != nil && order.ID != "" {
		switch order.Side {
		case Bid:
			heap.Push(e.bids, order)
		case Ask:
			heap.Push(e.asks, order)
		default:
			slog.Warn("unknown order side", "side", order.Side)
		}

		e.tryMatch()
		return nil
	}
	return errors.New("error while placing order. Either order or order id is nil")
}

func (e *Exchange) tryMatch() {
	for e.asks.Len() > 0 && e.bids.Len() > 0 && e.asks.Peek().PricePerTH <= e.bids.Peek().PricePerTH {
		if e.asks.Peek().Cancelled {
			heap.Pop(e.asks)
			continue
		}

		if e.bids.Peek().Cancelled {
			heap.Pop(e.bids)
			continue
		}

		// sem match
		if e.asks.Peek().PricePerTH > e.bids.Peek().PricePerTH {
			break
		}

		bestBid := e.bids.Peek()
		bestAsk := e.asks.Peek()

		quantity := min(bestBid.HashrateTHs, bestAsk.HashrateTHs)
		bestBid.HashrateTHs -= quantity
		bestAsk.HashrateTHs -= quantity

		if bestBid.HashrateTHs == 0 {
			heap.Pop(e.bids)
		}

		if bestAsk.HashrateTHs == 0 {
			heap.Pop(e.asks)
		}

		e.revenue[bestAsk.MinerID] += quantity * bestAsk.PricePerTH
	}
}

// RegisterMiner no contexto da exchange significa: colocar um Ask automaticamente.
func (e *Exchange) RegisterMiner(minerID string, hashrateTHs float64) error {
	if err := e.PlaceOrder(&Order{
		ID:          uuid.New().String(),
		MinerID:     minerID,
		HashrateTHs: hashrateTHs,
		PricePerTH:  defaultAskPrice,
		Side:        Ask,
		CreatedAt:   time.Now(),
	}); err != nil {
		return err
	}
	return nil
}

func (e *Exchange) UnregisterMiner(minerID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.revenue[minerID]; !ok {
		return
	}

	for id, order := range e.orders {
		if order.MinerID == minerID {
			order.Cancelled = true
			delete(e.orders, id)
		}
	}

	// reutilizando a memória que o heap foi alocado
	filteredHeap := (*e.asks)[:0]
	for _, order := range *e.asks {
		if !order.Cancelled {
			filteredHeap = append(filteredHeap, order)
		}
	}

	*e.asks = filteredHeap
	heap.Init(e.asks)
}

func (e *Exchange) TotalHashrate() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	total := 0.0
	for _, order := range *e.asks {
		total += order.HashrateTHs
	}
	return total
}

func (e *Exchange) Revenue(minerID string) float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if val, ok := e.revenue[minerID]; ok {
		return val
	}
	return 0
}

type askHeap []*Order
type bidHeap []*Order

func (h *askHeap) Push(x any) {
	*h = append(*h, x.(*Order))
}

func (h *askHeap) Pop() (x any) {
	x, *h = (*h)[len(*h)-1], (*h)[:len(*h)-1]
	return x
}

func (h *askHeap) Swap(i, j int) {
	(*h)[i], (*h)[j] = (*h)[j], (*h)[i]
}

func (h *askHeap) Less(i, j int) bool {
	return (*h)[i].PricePerTH < (*h)[j].PricePerTH
}

func (h *askHeap) Len() int {
	return len(*h)
}

func (h *askHeap) Peek() *Order {
	return (*h)[0]
}

func (h *bidHeap) Push(x any) {
	*h = append(*h, x.(*Order))
}

func (h *bidHeap) Pop() (x any) {
	x, *h = (*h)[len(*h)-1], (*h)[:len(*h)-1]
	return x
}

func (h *bidHeap) Swap(i, j int) {
	(*h)[i], (*h)[j] = (*h)[j], (*h)[i]
}

func (h *bidHeap) Less(i, j int) bool {
	return (*h)[i].PricePerTH > (*h)[j].PricePerTH
}

func (h *bidHeap) Len() int {
	return len(*h)
}

func (h *bidHeap) Peek() *Order {
	return (*h)[0]
}
