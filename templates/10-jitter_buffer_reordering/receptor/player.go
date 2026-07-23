package main

import (
	"log"
	"time"
)

type Player struct {
	expectedSeq  uint64
	lastPlayTime time.Time
	stats        *PlayBackStats
}

type PlayBackStats struct {
	TotalPlayed  uint64
	TotalLost    uint64
	TotalLatency uint64
}

func NewPlayer() *Player {
	return &Player{
		expectedSeq:  0,
		lastPlayTime: time.Time{},
		stats:        new(PlayBackStats),
	}
}

func (p *Player) Bootstrap(jitterBuffer *JitterBuffer) {
	for packet := range jitterBuffer.out {

	}
}

// metrics report
func (p *Player) Report() {
	avgLatency := float64(0)
	if p.stats.TotalPlayed > 0 {
		avgLatency = float64(p.stats.TotalLatency) / float64(p.stats.TotalPlayed)
	}

	log.Printf(
		"--- session report ---\nplayed: %d\nlost: %d\navg latency: %.2fms",
		p.stats.TotalPlayed, p.stats.TotalLost, avgLatency,
	)
}
