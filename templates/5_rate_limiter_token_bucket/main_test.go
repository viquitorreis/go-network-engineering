package main

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenBucket_InitialTokens(t *testing.T) {
	// Bucket deve começar cheio
	tb := NewTokenBucket(10, 5, 100*time.Millisecond)

	available, capacity := tb.Stats()
	if available != capacity {
		t.Errorf("expected bucket to start full: got %d/%d", available, capacity)
	}
}

func TestTokenBucket_ConsumeAllTokens(t *testing.T) {
	tb := NewTokenBucket(5, 1, 100*time.Millisecond)

	// Deve conseguir consumir exatamente 5 tokens
	for i := 0; i < 5; i++ {
		if !tb.Allow() {
			t.Fatalf("expected token %d to be available", i+1)
		}
	}

	// Sexto deve falhar
	if tb.Allow() {
		t.Error("expected 6th request to be rejected")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	// Bucket pequeno: 5 tokens, adiciona 5 a cada 50ms
	tb := NewTokenBucket(5, 5, 50*time.Millisecond)

	// Consumir todos
	for i := 0; i < 5; i++ {
		tb.Allow()
	}

	// Verificar que está vazio
	if tb.Allow() {
		t.Error("bucket should be empty")
	}

	// Esperar tempo suficiente para refill completo
	time.Sleep(60 * time.Millisecond)

	// Deve ter 5 tokens novamente
	allowed := 0
	for i := 0; i < 10; i++ {
		if tb.Allow() {
			allowed++
		}
	}

	if allowed != 5 {
		t.Errorf("expected 5 tokens after refill, got %d", allowed)
	}
}

func TestTokenBucket_ConcurrentAccess(t *testing.T) {
	tb := NewTokenBucket(100, 10, 50*time.Millisecond)

	var allowed atomic.Int32
	var rejected atomic.Int32

	var wg sync.WaitGroup
	// 200 goroutines competindo por 100 tokens
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tb.Allow() {
				allowed.Add(1)
			} else {
				rejected.Add(1)
			}
		}()
	}

	wg.Wait()

	total := int(allowed.Load()) + int(rejected.Load())
	if total != 200 {
		t.Errorf("expected 200 total requests, got %d", total)
	}

	// Deve ter aceito aproximadamente 100 (pode variar um pouco por causa de refill)
	if allowed.Load() < 95 || allowed.Load() > 105 {
		t.Errorf("expected ~100 allowed, got %d", allowed.Load())
	}
}

func TestTokenBucket_RateLimiting(t *testing.T) {
	// 10 tokens, refill 10 a cada 100ms = 100 tokens/segundo
	tb := NewTokenBucket(10, 10, 100*time.Millisecond)

	start := time.Now()
	allowed := 0

	// Tentar consumir 50 tokens o mais rápido possível
	for allowed < 50 {
		if tb.Allow() {
			allowed++
		} else {
			time.Sleep(10 * time.Millisecond) // Pequeno sleep para não criar busy loop
		}
	}

	elapsed := time.Since(start)

	// 50 tokens a 100/segundo deveria levar ~500ms
	// Damos margem de erro (300ms a 700ms)
	if elapsed < 300*time.Millisecond || elapsed > 700*time.Millisecond {
		t.Errorf("expected ~500ms to get 50 tokens, took %v", elapsed)
	}
}

// IMPORTANTE: Rode com go test -race
func TestTokenBucket_RaceConditions(t *testing.T) {
	tb := NewTokenBucket(50, 25, 50*time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				tb.Allow()
				time.Sleep(5 * time.Millisecond)
			}
		}()
	}

	wg.Wait()
}
