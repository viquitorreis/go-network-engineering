package protocol

import (
	"bytes"
	"fmt"
	"strconv"
)

// datagram represents a packet of data,
// each packet will represent a frame to be played on a specific time frame
// the seq number solves de sequence detection problem
type RTPDatagram struct {
	Payload   uint16 // random data..
	SeqNumber uint32
	Timestamp uint64 // unix time
}

func Parse(b []byte) *RTPDatagram {
	split := bytes.Split(b, []byte(" "))
	if len(split) < 3 {
		return nil
	}

	seq, err := strconv.ParseUint(string(split[0]), 10, 32)
	if err != nil {
		return nil
	}
	payload, err := strconv.ParseUint(string(split[1]), 10, 16)
	if err != nil {
		return nil
	}
	timestamp, err := strconv.ParseUint(string(split[2]), 10, 64)
	if err != nil {
		return nil
	}

	return &RTPDatagram{
		SeqNumber: uint32(seq),
		Payload:   uint16(payload),
		Timestamp: timestamp,
	}
}

func (r RTPDatagram) ToDatagram() []byte {
	var b []byte
	return fmt.Appendf(b, "%d %d %d", r.SeqNumber, r.Payload, r.Timestamp)
}
