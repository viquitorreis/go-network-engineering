package main

import (
	"context"
	"math/rand"
	"time"
)

type JitterBuffer struct {
	sl       *SkipList
	deadline time.Duration
	out      chan *Packet
}

func NewJitterBuffer(ctx context.Context, deadline time.Duration) *JitterBuffer {
	jb := &JitterBuffer{
		sl:       NewSkipList(16, 0.5, rand.New(rand.NewSource(time.Now().UnixNano()))), // sem ctx/deadline aqui, skip list volta a ser genérica
		deadline: deadline,
		out:      make(chan *Packet),
	}

	go jb.Bootstrap(ctx)

	return jb
}

func (jb *JitterBuffer) Insert(seq uint64, p *Packet) {
	jb.sl.Insert(seq, p)
}

func (jb *JitterBuffer) Out() <-chan *Packet {
	return jb.out
}

func (jb *JitterBuffer) Bootstrap(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for {
				packet, ok := jb.sl.PopFrontWithCond(func(p *Packet) bool {
					releasedAt := time.UnixMilli(int64(p.Timestamp)).Add(jb.deadline)

					return !time.Now().Before(releasedAt)
				})
				if !ok {
					break
				}

				jb.out <- packet
			}
		}
	}
}
