package main

import (
	"encoding/binary"
	"net"
	"sync"
)

type Client struct {
	conn   *net.Conn
	topics map[Topic]bool
	send   chan []byte

	mu sync.RWMutex
}

func NewClient(conn *net.Conn) *Client {
	return &Client{
		conn:   conn,
		topics: make(map[Topic]bool),
		send:   make(chan []byte, 1),
	}
}

func (c *Client) AddTopic(t Topic) bool {
	if !t.IsValid() {
		return false
	}

	c.mu.Lock()
	if _, ok := c.topics[t]; ok {
		c.mu.Unlock()
		return true
	}

	c.topics[t] = true
	c.mu.Unlock()

	return true
}

func (c *Client) WriteFrame(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	sizeBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(sizeBuf, uint32(len(data)))

	if _, err := (*c.conn).Write(sizeBuf); err != nil {
		return err
	}
	_, err := (*c.conn).Write(data)
	return err
}
