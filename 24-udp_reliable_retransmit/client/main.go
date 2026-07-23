package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"udpreliable/types"
)

func main() {
	addr, err := net.ResolveUDPAddr("udp4", ":8080")
	if err != nil {
		log.Fatalf("err resolving udp addr: %v", err)
	}

	udpDial, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		log.Fatalf("err resolving udp conn: %v", err)
	}
	defer udpDial.Close()

	log.Println("client alive")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := NewClient(udpDial, ctx)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	client.wg.Go(func() {
		client.readLoop()
	})

	go func() {
		<-sigCh
		log.Println("closing client")
		cancel()
		udpDial.Close()
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()

		client.wg.Go(func() {
			client.SendReliable(text)
		})
	}

	client.wg.Wait()
	log.Println("bye")
}

type Client struct {
	Conn    *net.UDPConn
	Session *types.Session

	// each msg will have a message chan of 1
	pendingMu sync.Mutex
	pending   map[uint64]chan *types.Message

	maxRetries  uint8
	readTimeout time.Duration

	ctx context.Context
	wg  sync.WaitGroup
}

func NewClient(conn *net.UDPConn, ctx context.Context) *Client {
	return &Client{
		Conn:        conn,
		Session:     types.NewSession(),
		pending:     make(map[uint64]chan *types.Message),
		maxRetries:  5,
		readTimeout: 500 * time.Millisecond,
		ctx:         ctx,
	}
}

func (c *Client) readLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		buf := make([]byte, 512)
		n, err := c.Conn.Read(buf)
		if err != nil {
			continue // shutdown or real error
		}

		reply, err := types.ParseMsg(buf[:n])
		if err != nil || reply.Cmd != types.ACKCmd {
			continue
		}

		c.pendingMu.Lock()
		ch, ok := c.pending[reply.ACK]
		c.pendingMu.Unlock()

		if ok {
			ch <- reply
		}
	}
}

func (c *Client) SendReliable(text string) error {
	// stop-and-wait: send msg and wait for response.... (sync)
	seq := c.Session.NextSeq()
	msg := types.Message{
		Cmd:     types.MSGCmd,
		Content: []byte(text),
		ACK:     seq,
	}
	datagram := msg.ToDatagram()

	replyCh := make(chan *types.Message, 1)

	c.pendingMu.Lock()
	c.pending[seq] = replyCh
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, seq)
		c.pendingMu.Unlock()
	}()

	for range c.maxRetries {
		if _, err := c.Conn.Write(datagram); err != nil {
			return err
		}

		select {
		case reply := <-replyCh:
			log.Println("reply: ", reply)
			return nil
		case <-time.After(c.readTimeout):
			continue
		case <-c.ctx.Done():
			return c.ctx.Err()
		}

	}

	return fmt.Errorf("failed to received ACK after %d retries", c.maxRetries)
}
