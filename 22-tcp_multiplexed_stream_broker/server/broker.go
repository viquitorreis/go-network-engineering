package main

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"time"
)

type Broker struct {
	topics map[Topic]map[*Client]bool

	broadcast chan *Message

	register   chan Register
	unregister chan Register
}

type Register struct {
	client *Client
	topic  Topic
}

type Message struct {
	topic Topic
	data  []byte
}

func NewBroker(ctx context.Context) *Broker {
	broker := &Broker{
		topics:     make(map[Topic]map[*Client]bool),
		broadcast:  make(chan *Message, 20),
		register:   make(chan Register),
		unregister: make(chan Register),
	}

	go broker.Bootstrap(ctx)

	return broker
}

func (b *Broker) Bootstrap(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.routeBroadcast()
		case reg := <-b.register:
			if b.topics[reg.topic] == nil {
				b.topics[reg.topic] = make(map[*Client]bool)
			}

			b.topics[reg.topic][reg.client] = true

			reg.client.AddTopic(reg.topic)
			log.Printf("REGISTERED: client on topic %s, total: %d", reg.topic, len(b.topics[reg.topic])) // temporário
		case msg := <-b.broadcast:
			b.routeMessage(msg)
		case unreg := <-b.unregister:
			delete(b.topics[unreg.topic], unreg.client)
		}
	}
}

func (b *Broker) routeBroadcast() {
	// rand values generated
	// case BTCTopic:
	btcVal := rand.IntN(65000-60000) + 60000

	// case ETHTopic:
	ethVal := rand.IntN(9000-8500) + 8500

	for topic, clientMap := range b.topics {
		switch topic {
		case BTCTopic:
			for client, ok := range clientMap {
				if ok {
					client.WriteFrame([]byte(fmt.Sprintf("%d", btcVal)))
				}
			}
		case ETHTopic:
			for client, ok := range clientMap {
				if ok {
					client.WriteFrame([]byte(fmt.Sprintf("%d", ethVal)))
				}
			}
		}
	}
}

func (b *Broker) routeMessage(msg *Message) {
	for client := range b.topics[msg.topic] {
		client.WriteFrame(msg.data)
	}
}
