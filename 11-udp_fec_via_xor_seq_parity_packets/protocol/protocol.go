package protocol

import (
	"bytes"
	"fmt"
	"strconv"
)

// datagram represents a packet of data,
// each packet will represent a frame to be played on a specific time frame
// the seq number solves de sequence detection problem
type Datagram struct {
	GroupID    uint64
	Payload    uint64 // random data..
	SeqNumber  uint64
	Timestamp  uint64 // unix time
	PacketType PacketType
}

type PacketType uint8

const (
	DataPacket PacketType = iota
	ParityPacket
)

func Parse(b []byte) *Datagram {
	split := bytes.Split(b, []byte(" "))
	if len(split) < 5 {
		return nil
	}

	groupID, err := strconv.ParseUint(string(split[0]), 10, 64)
	if err != nil {
		return nil
	}

	seq, err := strconv.ParseUint(string(split[1]), 10, 64)
	if err != nil {
		return nil
	}

	payload, err := strconv.ParseUint(string(split[2]), 10, 64)
	if err != nil {
		return nil
	}

	timestamp, err := strconv.ParseUint(string(split[3]), 10, 64)
	if err != nil {
		return nil
	}

	pakType, err := strconv.ParseUint(string(split[4]), 10, 8)
	if err != nil {
		return nil
	}

	return &Datagram{
		GroupID:    groupID,
		SeqNumber:  seq,
		Payload:    payload,
		Timestamp:  timestamp,
		PacketType: PacketType(pakType),
	}
}

func (r Datagram) ToDatagram() []byte {
	var b []byte
	return fmt.Appendf(b, "%d %d %d %d %d", r.GroupID, r.SeqNumber, r.Payload, r.Timestamp, r.PacketType)
}
