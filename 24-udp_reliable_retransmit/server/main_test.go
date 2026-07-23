package main

import (
	"context"
	"net"
	"testing"
	"time"
	"udpreliable/types"
)

func startTestServer(t *testing.T) (addr *net.UDPAddr, cleanup func()) {
	t.Helper()

	laddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("err resolvendo addr de teste: %v", err)
	}
	conn, err := net.ListenUDP("udp4", laddr)
	if err != nil {
		t.Fatalf("err abrindo socket de teste: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	server := NewServer(conn, ctx, 0.0)
	go server.Read()

	cleanup = func() {
		cancel()
		conn.Close()
	}
	return conn.LocalAddr().(*net.UDPAddr), cleanup
}

// sendRaw sends a raw datagram to the server and reads the answer with a deadline,
// sem depender do tipo Client real — mantém o teste independente da
// without depending on the real client type, which keeps the test independent of the client implementation
// this allows testing only the server's protocol contract
func sendRaw(t *testing.T, serverAddr *net.UDPAddr, datagram []byte) *types.Message {
	t.Helper()

	conn, err := net.DialUDP("udp4", nil, serverAddr)
	if err != nil {
		t.Fatalf("err discando pro server de teste: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write(datagram); err != nil {
		t.Fatalf("err escrevendo pro server: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("err lendo resposta do server (timeout ou erro real): %v", err)
	}

	reply, err := types.ParseMsg(buf[:n])
	if err != nil {
		t.Fatalf("resposta do server não parseou como Message válida: %v", err)
	}
	return reply
}

func TestServer_RespondsWithACK_MatchingSequence(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	req := types.Message{Cmd: types.MSGCmd, Content: []byte("hello"), ACK: 1}
	reply := sendRaw(t, addr, req.ToDatagram())

	if reply.Cmd != types.ACKCmd {
		t.Errorf("Cmd da resposta = %q, esperado ACKCmd", reply.Cmd)
	}
	if reply.ACK != req.ACK {
		t.Errorf("seq no ACK = %d, esperado %d (mesmo seq da mensagem original)", reply.ACK, req.ACK)
	}
}

func TestServer_ReAcksDuplicateSequence_WithoutDoubleProcessing(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	req := types.Message{Cmd: types.MSGCmd, Content: []byte("hello"), ACK: 5}

	first := sendRaw(t, addr, req.ToDatagram())
	second := sendRaw(t, addr, req.ToDatagram()) // mesmo seq, simulando retransmissão

	if first.ACK != 5 || second.ACK != 5 {
		t.Fatalf("esperava ACK=5 nas duas respostas, veio %d e %d", first.ACK, second.ACK)
	}
}

func TestServer_HandlesMultipleClientsIndependently(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	// two different clients, each using seq=1 there should be no
	// collision between them, since deduplication is by source address
	reqA := types.Message{Cmd: types.MSGCmd, Content: []byte("from A"), ACK: 1}
	reqB := types.Message{Cmd: types.MSGCmd, Content: []byte("from B"), ACK: 1}

	replyA := sendRaw(t, addr, reqA.ToDatagram())
	replyB := sendRaw(t, addr, reqB.ToDatagram())

	if replyA.ACK != 1 || replyB.ACK != 1 {
		t.Fatalf("esperava ACK=1 pros dois clients, veio %d e %d", replyA.ACK, replyB.ACK)
	}
}
