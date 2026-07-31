package main

import (
	"context"
	"encoding/binary"
	"feq_parity_packets/protocol"
	"net"
	"testing"
	"time"
)

// buildFrame monta um frame completo (prefixo de 4 bytes de tamanho + payload
// do datagrama), no mesmo formato que o sender real produz.
func buildFrame(groupID, seq, payload, timestamp uint64, pType protocol.PacketType) []byte {
	msg := protocol.Datagram{
		GroupID:    groupID,
		SeqNumber:  seq,
		Payload:    payload,
		Timestamp:  timestamp,
		PacketType: pType,
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

	r = NewReceptor(ctx, 4, serverSide)
	r.maxWait = 30 * time.Millisecond

	go r.handleRead()
	go r.monitor()

	t.Cleanup(func() {
		cancel()
		testSide.Close()
	})

	return r, testSide, cancel
}

func TestReceptor_CompleteGroup_NoRecoveryNeeded(t *testing.T) {
	r, conn, _ := newTestReceptor(t)

	// grupo 0 completo: 4 dados + paridade, nada faltando
	// group 0 complete: 4 data packets + parity, nothing missing
	var parity uint64
	payloads := []uint64{10, 20, 30, 40}
	for i, p := range payloads {
		conn.Write(buildFrame(0, uint64(i), p, 0, protocol.DataPacket))
		parity ^= p
	}
	conn.Write(buildFrame(0, 0, parity, 0, protocol.ParityPacket))

	time.Sleep(100 * time.Millisecond) // waits monitor process

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stats.Completed != 1 {
		t.Errorf("Completed = %d, expected 1", r.stats.Completed)
	}

	if r.stats.Recovered != 0 || r.stats.Lost != 0 {
		t.Errorf("expected only Completed, got Recovered=%d Lost=%d", r.stats.Recovered, r.stats.Lost)
	}
}

func TestReceptor_OneMissing_RecoveredViaFEC(t *testing.T) {
	r, conn, _ := newTestReceptor(t)

	// group 0: missing the packet of data and seq=2 by purpose
	payloads := map[uint64]uint64{0: 10, 1: 20, 3: 40}
	var parity uint64
	for i, p := range map[uint64]uint64{0: 10, 1: 20, 2: 30, 3: 40} {
		_ = i

		parity ^= p
	}

	for seq, p := range payloads {
		conn.Write(buildFrame(0, seq, p, 0, protocol.DataPacket))
	}

	conn.Write(buildFrame(0, 0, parity, 0, protocol.ParityPacket))

	time.Sleep(100 * time.Millisecond)

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stats.Recovered != 1 {
		t.Fatalf("Recovered = %d, expected 1 (only 1 packet missing, should reconstruct)", r.stats.Recovered)
	}

	if r.stats.Completed != 0 || r.stats.Lost != 0 {
		t.Errorf("expected only Recovered, got Completed=%d Lost=%d", r.stats.Completed, r.stats.Lost)
	}
}

func TestReceptor_TwoMissing_DefinitiveLoss(t *testing.T) {
	r, conn, _ := newTestReceptor(t)

	// group 0: only 2 of 4 data packets arrive, plus parity 2 missing,
	// not recoverable with simple XOR
	conn.Write(buildFrame(0, 0, 10, 0, protocol.DataPacket))
	conn.Write(buildFrame(0, 1, 20, 0, protocol.DataPacket))
	conn.Write(buildFrame(0, 0, 10^20^30^40, 0, protocol.ParityPacket))

	time.Sleep(100 * time.Millisecond)

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stats.Lost != 1 {
		t.Fatalf("Lost = %d, expected 1 (2 packets missing, not recoverable)", r.stats.Lost)
	}

	if r.stats.Completed != 0 || r.stats.Recovered != 0 {
		t.Errorf("expected only Lost, got Completed=%d Recovered=%d", r.stats.Completed, r.stats.Recovered)
	}
}

func TestReceptor_MultipleGroups_Independent(t *testing.T) {
	r, conn, _ := newTestReceptor(t)

	// group 0: complete
	var parity0 uint64
	for i, p := range []uint64{1, 2, 3, 4} {
		conn.Write(buildFrame(0, uint64(i), p, 0, protocol.DataPacket))
		parity0 ^= p
	}
	conn.Write(buildFrame(0, 0, parity0, 0, protocol.ParityPacket))

	// group 1: only parity arrives, all data packets lost
	conn.Write(buildFrame(1, 0, 999, 0, protocol.ParityPacket))

	time.Sleep(100 * time.Millisecond)

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stats.Completed != 1 {
		t.Errorf("Completed = %d, expected 1 (group 0)", r.stats.Completed)
	}

	if r.stats.Lost != 1 {
		t.Errorf("Lost = %d, expected 1 (group 1, only parity arrived)", r.stats.Lost)
	}
}
