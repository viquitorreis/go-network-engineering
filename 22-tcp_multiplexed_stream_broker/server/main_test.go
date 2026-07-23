package main

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

// readFrameTest reads a length-prefixed frame (4 bytes size + payload)
// the same protocol used by the real client. Used only in tests.
func readFrameTest(t *testing.T, conn net.Conn) []byte {
	t.Helper()

	sizeBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, sizeBuf); err != nil {
		t.Fatalf("err reading frame size: %v", err)
	}
	size := binary.BigEndian.Uint32(sizeBuf)

	payload := make([]byte, size)
	if _, err := io.ReadFull(conn, payload); err != nil {
		t.Fatalf("err reading frame payload: %v", err)
	}
	return payload
}

// newTestClient creates a pair of connections via net.Pipe and returns
// the *Client (server side) already ready to be registered in the broker,
// and the side of the connection that the "test client" uses to read what arrives.
func newTestClient() (*Client, net.Conn) {
	serverSide, testSide := net.Pipe()
	client := NewClient(&serverSide)
	return client, testSide
}

func TestClient_WriteFrame_UsesLengthPrefix(t *testing.T) {
	client, testSide := newTestClient()
	defer testSide.Close()

	payload := []byte("62453")

	done := make(chan error, 1)
	go func() {
		done <- client.WriteFrame(payload)
	}()

	got := readFrameTest(t, testSide)

	if err := <-done; err != nil {
		t.Fatalf("WriteFrame retornou erro: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("payload recebido = %q, esperado %q", got, payload)
	}
}

func TestBroker_RegisterThenBroadcast_DeliversToSubscriber(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	broker := NewBroker(ctx)
	client, testSide := newTestClient()
	defer testSide.Close()

	broker.register <- Register{client: client, topic: BTCTopic}

	// small delay to allow the broker to process the register before broadcasting
	time.Sleep(50 * time.Millisecond)

	msg := &Message{topic: BTCTopic, data: []byte("62453")}

	done := make(chan struct{})
	go func() {
		got := readFrameTest(t, testSide)
		if string(got) != "62453" {
			t.Errorf("payload recebido = %q, esperado %q", got, "62453")
		}
		close(done)
	}()

	broker.broadcast <- msg

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout esperando mensagem chegar no subscriber")
	}
}

func TestBroker_Broadcast_DoesNotLeakAcrossTopics(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	broker := NewBroker(ctx)

	btcClient, btcSide := newTestClient()
	defer btcSide.Close()
	ethClient, ethSide := newTestClient()
	defer ethSide.Close()

	broker.register <- Register{client: btcClient, topic: BTCTopic}
	broker.register <- Register{client: ethClient, topic: ETHTopic}
	time.Sleep(50 * time.Millisecond)

	// only publishes to BTC the ETH subscriber should not receive anything
	broker.broadcast <- &Message{topic: BTCTopic, data: []byte("100")}

	btcDone := make(chan struct{})
	go func() {
		got := readFrameTest(t, btcSide)
		if string(got) != "100" {
			t.Errorf("BTC subscriber recebeu %q, esperado %q", got, "100")
		}
		close(btcDone)
	}()

	select {
	case <-btcDone:
	case <-time.After(time.Second):
		t.Fatal("timeout esperando mensagem no subscriber de BTC")
	}

	// confirms that the ETH subscriber did NOT receive anything
	ethSide.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	buf := make([]byte, 4)
	_, err := ethSide.Read(buf)
	if err == nil {
		t.Fatal("subscriber de ETH recebeu dado, mas não deveria (topic diferente)")
	}
}

type frameResult struct {
	payload []byte
	err     error
}

func readFrameAsync(conn net.Conn) <-chan frameResult {
	ch := make(chan frameResult, 1)
	go func() {
		sizeBuf := make([]byte, 4)
		if _, err := io.ReadFull(conn, sizeBuf); err != nil {
			ch <- frameResult{err: err}
			return
		}
		size := binary.BigEndian.Uint32(sizeBuf)

		payload := make([]byte, size)
		if _, err := io.ReadFull(conn, payload); err != nil {
			ch <- frameResult{err: err}
			return
		}

		ch <- frameResult{payload: payload}
	}()
	return ch
}

func TestBroker_MultipleSubscribersSameTopic_BothReceive(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	broker := NewBroker(ctx)

	clientA, sideA := newTestClient()
	defer sideA.Close()
	clientB, sideB := newTestClient()
	defer sideB.Close()

	broker.register <- Register{client: clientA, topic: BTCTopic}
	broker.register <- Register{client: clientB, topic: BTCTopic}
	time.Sleep(50 * time.Millisecond)

	// IMPORTANT: as net.Pipe is synchronous (unbuffered),
	// the broker's Write will block until the other side reads.
	// So we need to start reading from both sides before broadcasting,
	// otherwise the broker may block writing to the second client while we're only reading from the first.
	chA := readFrameAsync(sideA)
	chB := readFrameAsync(sideB)

	broker.broadcast <- &Message{topic: BTCTopic, data: []byte("777")}

	timeout := time.After(time.Second)
	for i, ch := range []<-chan frameResult{chA, chB} {
		select {
		case res := <-ch:
			if res.err != nil {
				t.Fatalf("erro lendo subscriber %d: %v", i, res.err)
			}
			if string(res.payload) != "777" {
				t.Errorf("subscriber %d recebeu %q, esperado %q", i, res.payload, "777")
			}
		case <-timeout:
			t.Fatal("timeout esperando mensagem chegar em um dos dois subscribers")
		}
	}
}

func TestBroker_Unregister_StopsDelivery(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	broker := NewBroker(ctx)
	client, testSide := newTestClient()
	defer testSide.Close()

	broker.register <- Register{client: client, topic: BTCTopic}
	time.Sleep(50 * time.Millisecond)

	broker.unregister <- Register{client: client, topic: BTCTopic}
	time.Sleep(50 * time.Millisecond)

	broker.broadcast <- &Message{topic: BTCTopic, data: []byte("111")}

	testSide.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	buf := make([]byte, 4)
	_, err := testSide.Read(buf)
	if err == nil {
		t.Fatal("client desregistrado recebeu mensagem, mas não deveria")
	}
}

func TestTopic_IsValid(t *testing.T) {
	cases := []struct {
		topic Topic
		valid bool
	}{
		{BTCTopic, true},
		{ETHTopic, true},
		{Topic("DOGE"), false},
		{Topic(""), false},
	}

	for _, c := range cases {
		if got := c.topic.IsValid(); got != c.valid {
			t.Errorf("Topic(%q).IsValid() = %v, esperado %v", c.topic, got, c.valid)
		}
	}
}
