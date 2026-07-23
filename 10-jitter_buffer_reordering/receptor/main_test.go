package main

import (
	"context"
	"encoding/binary"
	"jitter_buffer/protocol"
	"net"
	"testing"
	"time"
)

// buildFrame monta um frame completo (prefixo de 4 bytes de tamanho + payload
// do datagrama), no mesmo formato que o sender real produz.
func buildFrame(seq, payload, timestamp uint64) []byte {
	msg := protocol.RTPDatagram{
		SeqNumber: seq,
		Payload:   payload,
		Timestamp: timestamp,
	}
	body := msg.ToDatagram()

	sizeBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBuf, uint32(len(body)))

	frame := make([]byte, 0, len(sizeBuf)+len(body))
	frame = append(frame, sizeBuf...)
	frame = append(frame, body...)
	return frame
}

// newTestReceptor cria um Receptor real ligado a um net.Pipe, e devolve o
// lado da conexão que o teste usa pra "enviar" frames como se fosse o
// sender de verdade sem precisar de um socket UDP real.
func newTestReceptor(t *testing.T) (r *Receptor, testSide net.Conn, cancel context.CancelFunc) {
	t.Helper()

	serverSide, testSide := net.Pipe()
	ctx, cancel := context.WithCancel(context.Background())

	r = NewReceptor(ctx, serverSide)
	go r.handleRead()

	t.Cleanup(func() {
		cancel()
		testSide.Close()
	})

	return r, testSide, cancel
}

func TestReceptor_InOrderPackets_NoGapsNoDelays(t *testing.T) {
	r, conn, _ := newTestReceptor(t)

	for seq := uint64(0); seq < 5; seq++ {
		conn.Write(buildFrame(seq, seq, seq*20))
	}
	time.Sleep(50 * time.Millisecond)

	snap := r.Snapshot()

	if len(snap.SeqGaps) != 0 {
		t.Errorf("esperava 0 gaps em sequência sem perda, teve %d", len(snap.SeqGaps))
	}

	if len(snap.Delayed) != 0 {
		t.Errorf("esperava 0 pacotes atrasados em sequência sem perda, teve %d", len(snap.Delayed))
	}

	if snap.LastSeqNum != 4 {
		t.Errorf("LastSeqNum = %d, esperado 4 (último seq recebido)", snap.LastSeqNum)
	}
}

func TestReceptor_DetectsGap(t *testing.T) {
	r, conn, _ := newTestReceptor(t)
	conn.Write(buildFrame(0, 0, 0))
	conn.Write(buildFrame(1, 0, 20))

	// pula 2 e 3 de propósito simula 2 pacotes perdidos
	conn.Write(buildFrame(4, 0, 80))
	time.Sleep(50 * time.Millisecond)

	snap := r.Snapshot()

	if len(snap.SeqGaps) != 1 {
		t.Fatalf("esperava 1 evento de gap detectado, teve %d", len(snap.SeqGaps))
	}

	if snap.SeqGaps[0].SeqNumber != 4 {
		t.Errorf("gap deveria estar associado ao seq 4 (o que revelou o buraco), veio seq %d", snap.SeqGaps[0].SeqNumber)
	}

	if snap.LastSeqNum != 4 {
		t.Errorf("LastSeqNum deveria avançar para 4 mesmo com gap, veio %d", snap.LastSeqNum)
	}
}

func TestReceptor_DetectsOutOfOrderPacket(t *testing.T) {
	r, conn, _ := newTestReceptor(t)

	conn.Write(buildFrame(0, 0, 0))
	conn.Write(buildFrame(1, 0, 20))
	conn.Write(buildFrame(2, 0, 40))
	// chega um pacote atrasado, com seq menor que o maior já visto
	conn.Write(buildFrame(1, 0, 20))
	time.Sleep(50 * time.Millisecond)

	snap := r.Snapshot()

	if len(snap.Delayed) != 1 {
		t.Fatalf("esperava 1 pacote atrasado detectado, teve %d", len(snap.Delayed))
	}

	if snap.Delayed[0].SeqNumber != 1 {
		t.Errorf("pacote atrasado deveria ser o seq 1, veio seq %d", snap.Delayed[0].SeqNumber)
	}

	if snap.LastSeqNum != 2 {
		t.Errorf("LastSeqNum não deveria retroceder por causa de um pacote atrasado, veio %d", snap.LastSeqNum)
	}
}

func TestReceptor_GapThenRecovery_OnlyOneGapEvent(t *testing.T) {
	r, conn, _ := newTestReceptor(t)

	conn.Write(buildFrame(0, 0, 0))
	// pula 1 um gap
	conn.Write(buildFrame(2, 0, 40))
	conn.Write(buildFrame(3, 0, 60))
	conn.Write(buildFrame(4, 0, 80))
	time.Sleep(50 * time.Millisecond)

	snap := r.Snapshot()

	if len(snap.SeqGaps) != 1 {
		t.Fatalf("esperava exatamente 1 evento de gap (não um por pacote subsequente), teve %d", len(r.SeqGaps))
	}

	if snap.LastSeqNum != 4 {
		t.Errorf("LastSeqNum = %d, esperado 4 após recuperar a sequência", r.LastSeqNum)
	}
}
