package types

// for this challenge we need a simple reliable delivery method
//  1. generate next seq number when sending new msg
//  2. Store the message sent, associated with the sequence number, to re-send when it timeouts
//  3. A way to confirm it received when the ACK arrives
type Session struct {
	// seq -> original msg
	Msgs    map[uint64]Message
	nextSeq uint64
}

func NewSession() *Session {
	return &Session{
		Msgs: make(map[uint64]Message),
	}
}

func (s *Session) NextSeq() uint64 {
	s.nextSeq++
	return s.nextSeq
}

func (s *Session) MarkPending(seq uint64, m Message) {
	s.Msgs[seq] = m
}

func (s *Session) Confirm(seq uint64) {
	delete(s.Msgs, seq)
}

func (s *Session) IsPending(seq uint64) bool {
	_, ok := s.Msgs[seq]
	return ok
}
