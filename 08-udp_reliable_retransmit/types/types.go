package types

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

type Message struct {
	Cmd     Command
	Content []byte
	ACK     uint64
}

type Command string

const (
	MSGCmd Command = "MSG"
	ACKCmd Command = "ACK"
)

func (c *Command) ToString() string {
	return string(*c)
}

func (m Message) ToDatagram() []byte {
	var b []byte
	return fmt.Appendf(b, "%s %s %d", m.Cmd.ToString(), m.Content, m.ACK)
}

func ParseMsg(b []byte) (*Message, error) {
	split := bytes.Split(b, []byte(" "))
	if len(split) < 3 {
		return nil, errors.New("invalid message length")
	}

	cmd := split[0]
	ackRaw := split[len(split)-1]
	content := bytes.Join(split[1:len(split)-1], []byte(" "))

	// written as text must be decoded as text too
	ackNum, err := strconv.ParseUint(string(ackRaw), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid ack number: %w", err)
	}

	return &Message{
		Cmd:     Command(cmd),
		Content: content,
		ACK:     ackNum,
	}, nil
}
