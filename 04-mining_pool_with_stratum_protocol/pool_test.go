package main_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"miningpool/market"
	"miningpool/pool"
	"miningpool/protocol"
)

func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// TestConcurrentMiners garante que 50 mineradores conectando simultaneamente
// não causam race conditions no mapa de miners.
func TestConcurrentMiners(t *testing.T) {
	mp := market.NewMarketplace(0.05)
	srv := pool.NewServer(":0", mp) // porta 0 = OS escolhe porta livre

	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("server error: %v", err)
		}
	}()
	defer srv.Shutdown(context.Background())

	time.Sleep(time.Millisecond * 50)

	adrr := srv.Addr()

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			conn, err := net.Dial("tcp", adrr)
			if err != nil {
				t.Errorf("miner %d connect error: %v", id, err)
				return
			}
			defer conn.Close()

			msg := protocol.Message{
				ID: func() *int {
					num := id
					return &num
				}(),
				Method: protocol.Subscribe,
				Params: mustMarshal(protocol.SubscribeParams{
					UserAgent: fmt.Sprintf("test-miner-%d", id),
				}),
			}

			b, _ := json.Marshal(msg)
			b = append(b, '\n')
			_, err = conn.Write(b)
			if err != nil {
				t.Errorf("miner %d write error: %v", id, err)
			}

			scanner := bufio.NewScanner(conn)
			if scanner.Scan() {
				var resp protocol.Response
				if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
					t.Errorf("miner %d response unmarshal error: %v", id, err)
				}
			} else if err := scanner.Err(); err != nil {
				t.Errorf("miner %d read error: %v", id, err)
			}
		}(i)
	}
	wg.Wait()
}

// TestShareValidation testa a validação de shares sem TCP.
func TestShareValidation(t *testing.T) {
	tests := []struct {
		hash       string
		difficulty uint64
		valid      bool
	}{
		{"0000abcd1234", 4, true},  // 4 leading zeros
		{"000abcd1234f", 4, false}, // só 3 leading zeros
		{"0000000012ab", 7, true},  // 7 leading zeros
	}

	for _, tc := range tests {
		got := protocol.ValidateShare(tc.hash, tc.difficulty)
		if got != tc.valid {
			t.Errorf("ValidateShare(%q, %d) = %v, want %v",
				tc.hash, tc.difficulty, got, tc.valid)
		}
	}
}

// TestMarketplaceTick verifica distribuição proporcional de revenue.
func TestMarketplaceTick(t *testing.T) {
	mp := market.NewMarketplace(1.0) // $1/TH/s para math simples

	mp.RegisterMiner("alice", 100) // alice tem 100 TH/s
	mp.RegisterMiner("bob", 300)   // bob tem 300 TH/s — 3x mais

	mp.Tick(1.0) // simula 1 hora

	// alice deve ter 25% do revenue total (100/400 * 1.0 * 400 = 100)
	// bob deve ter 75% do revenue total
	aliceRev := mp.Revenue("alice")
	bobRev := mp.Revenue("bob")

	if bobRev/aliceRev < 2.9 || bobRev/aliceRev > 3.1 {
		t.Errorf("proporção incorreta: alice=%.2f, bob=%.2f, ratio=%.2f",
			aliceRev, bobRev, bobRev/aliceRev)
	}
}

// TestBroadcastNoDeadlock garante que BroadcastJob não causa deadlock
// quando um minerador lento bloqueia.
func TestBroadcastNoDeadlock(t *testing.T) {
	// TODO: criar server com um minerador "lento" (buffer cheio),
	// chamar BroadcastJob, verificar que completa em < 1s
	done := make(chan struct{})
	go func() {
		// TODO: chamar BroadcastJob aqui
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(1 * time.Second):
		t.Fatal("BroadcastJob deadlocked com minerador lento")
	}
}

// TestExchangeMatching verifica que bids e asks fazem match corretamente.
func TestExchangeMatching(t *testing.T) {
	ex := market.NewExchange()

	// Bob quer comprar 100 TH/s por até $0.06
	ex.PlaceOrder(&market.Order{
		ID:          "bid-1",
		MinerID:     "buyer-bob",
		HashrateTHs: 100,
		PricePerTH:  0.06,
		Side:        market.Bid,
	})

	// Alice tem 100 TH/s e aceita $0.05
	ex.PlaceOrder(&market.Order{
		ID:          "ask-1",
		MinerID:     "alice",
		HashrateTHs: 100,
		PricePerTH:  0.05,
		Side:        market.Ask,
	})

	// Deve ter feito match — alice deve ter revenue > 0
	if ex.Revenue("alice") == 0 {
		t.Error("match não aconteceu: alice não recebeu revenue")
	}
}
