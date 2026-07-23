package main

import (
	"context"
	"math/rand"
	"testing"
	"time"
)

// -SkipList ---

func TestSkipList_Front_ReturnsSmallestScore(t *testing.T) {
	sl := NewSkipList(16, 0.5, rand.New(rand.NewSource(1)))

	sl.Insert(5, &Packet{SeqNumber: 5})
	sl.Insert(2, &Packet{SeqNumber: 2})
	sl.Insert(8, &Packet{SeqNumber: 8})

	front, ok := sl.Front()
	if !ok {
		t.Fatal("esperava Front() com dados, veio vazio")
	}
	if front.SeqNumber != 2 {
		t.Errorf("Front().SeqNumber = %d, esperado 2 (o menor inserido)", front.SeqNumber)
	}
}

func TestSkipList_PopFront_DrainsInAscendingOrder(t *testing.T) {
	sl := NewSkipList(16, 0.5, rand.New(rand.NewSource(1)))

	sl.Insert(3, &Packet{SeqNumber: 3})
	sl.Insert(1, &Packet{SeqNumber: 1})
	sl.Insert(2, &Packet{SeqNumber: 2})

	var got []uint64
	for {
		p, ok := sl.PopFront()
		if !ok {
			break
		}
		got = append(got, p.SeqNumber)
	}

	want := []uint64{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("PopFront devolveu %v, esperado %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("posição %d: got %d, want %d", i, got[i], want[i])
		}
	}
}

func TestSkipList_PopFrontWithCond_RespectsCondition(t *testing.T) {
	sl := NewSkipList(16, 0.5, rand.New(rand.NewSource(1)))
	sl.Insert(1, &Packet{SeqNumber: 1})

	_, ok := sl.PopFrontWithCond(func(p *Packet) bool { return false })
	if ok {
		t.Fatal("PopFrontWithCond não deveria remover quando a condição é false")
	}
	if sl.Size() != 1 {
		t.Fatalf("elemento deveria continuar na lista, size = %d", sl.Size())
	}

	p, ok := sl.PopFrontWithCond(func(p *Packet) bool { return true })
	if !ok || p.SeqNumber != 1 {
		t.Fatalf("PopFrontWithCond deveria remover quando a condição é true, got ok=%v p=%+v", ok, p)
	}
	if sl.Size() != 0 {
		t.Fatalf("elemento deveria ter sido removido, size = %d", sl.Size())
	}
}

// -JitterBuffer ---

// recvWithTimeout lê do channel com um prazo, pra não travar o teste
// se o buffer nunca liberar nada (bug real ficaria evidente como timeout).
func recvWithTimeout(t *testing.T, ch <-chan *Packet, timeout time.Duration) *Packet {
	t.Helper()
	select {
	case p := <-ch:
		return p
	case <-time.After(timeout):
		t.Fatal("timeout esperando pacote do JitterBuffer")
		return nil
	}
}

func TestJitterBuffer_ReordersOutOfSequenceArrival(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// deadline curto, e os pacotes já entram "vencidos" (timestamp no
	// passado) pra forçar liberação rápida e determinística no teste,
	// sem depender de esperar o deadline real passar.
	deadline := 20 * time.Millisecond
	jb := NewJitterBuffer(ctx, deadline)

	expired := time.Now().Add(-time.Second).UnixMilli()

	// insere fora de ordem de propósito: seq 2, depois 0, depois 1
	jb.Insert(2, &Packet{SeqNumber: 2, Timestamp: uint64(expired)})
	jb.Insert(0, &Packet{SeqNumber: 0, Timestamp: uint64(expired)})
	jb.Insert(1, &Packet{SeqNumber: 1, Timestamp: uint64(expired)})

	var got []uint64
	for i := 0; i < 3; i++ {
		p := recvWithTimeout(t, jb.out, 500*time.Millisecond)
		got = append(got, p.SeqNumber)
	}

	want := []uint64{0, 1, 2}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ordem de liberação = %v, esperado %v (reordenado, não ordem de chegada)", got, want)
		}
	}
}

func TestJitterBuffer_ReleasesAfterDeadline_EvenWithMissingPacket(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deadline := 20 * time.Millisecond
	jb := NewJitterBuffer(ctx, deadline)

	expired := time.Now().Add(-time.Second).UnixMilli()

	// seq 1 NUNCA chega (simula perda definitiva) só manda 0 e 2
	jb.Insert(0, &Packet{SeqNumber: 0, Timestamp: uint64(expired)})
	jb.Insert(2, &Packet{SeqNumber: 2, Timestamp: uint64(expired)})

	first := recvWithTimeout(t, jb.out, 500*time.Millisecond)
	second := recvWithTimeout(t, jb.out, 500*time.Millisecond)

	if first.SeqNumber != 0 || second.SeqNumber != 2 {
		t.Fatalf("esperava liberar seq 0 e depois seq 2 (sem travar esperando o 1 perdido), veio %d e %d",
			first.SeqNumber, second.SeqNumber)
	}
}

func TestJitterBuffer_HoldsPacketUntilDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deadline := 150 * time.Millisecond
	jb := NewJitterBuffer(ctx, deadline)

	// timestamp de AGORA não deveria liberar antes do deadline passar
	jb.Insert(0, &Packet{SeqNumber: 0, Timestamp: uint64(time.Now().UnixMilli())})

	select {
	case p := <-jb.out:
		t.Fatalf("pacote foi liberado cedo demais (seq %d), antes do deadline de %v vencer", p.SeqNumber, deadline)
	case <-time.After(50 * time.Millisecond):
		// esperado: nada liberado ainda nos primeiros 50ms de uma janela de 150ms
	}

	// agora espera o suficiente pro deadline vencer de verdade
	p := recvWithTimeout(t, jb.out, 300*time.Millisecond)
	if p.SeqNumber != 0 {
		t.Errorf("esperava seq 0 liberado após o deadline, veio seq %d", p.SeqNumber)
	}
}

func TestJitterBuffer_ConcurrentInsert_NoRace(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jb := NewJitterBuffer(ctx, 20*time.Millisecond)
	expired := uint64(time.Now().Add(-time.Second).UnixMilli())

	done := make(chan struct{})
	go func() {
		for i := uint64(0); i < 50; i++ {
			jb.Insert(i, &Packet{SeqNumber: i, Timestamp: expired})
		}
		close(done)
	}()

	<-done

	received := 0
	timeout := time.After(2 * time.Second)
	for received < 50 {
		select {
		case <-jb.out:
			received++
		case <-timeout:
			t.Fatalf("recebeu só %d de 50 pacotes esperados", received)
		}
	}
}
