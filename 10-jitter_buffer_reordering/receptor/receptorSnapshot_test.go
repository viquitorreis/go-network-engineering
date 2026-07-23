package main

import "jitter_buffer/protocol"

type ReceptorSnapshot struct {
	LastSeqNum uint64
	SeqGaps    []*protocol.RTPDatagram
	Delayed    []*protocol.RTPDatagram
}

func (r *Receptor) Snapshot() ReceptorSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()

	gaps := make([]*protocol.RTPDatagram, len(r.SeqGaps))
	copy(gaps, r.SeqGaps)
	delayed := make([]*protocol.RTPDatagram, len(r.Delayed))
	copy(delayed, r.Delayed)

	return ReceptorSnapshot{
		LastSeqNum: r.LastSeqNum,
		SeqGaps:    gaps,
		Delayed:    delayed,
	}
}
