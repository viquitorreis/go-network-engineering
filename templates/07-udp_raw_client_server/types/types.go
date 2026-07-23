package types

import (
	"bytes"
	"fmt"
	"strconv"
)

type Message struct {
	Cmd     Command
	Content int // can be string, depends on impl
}

type Command string

const (
	MSGCmd Command = "MSG"
)

func (c *Command) ToString() string {
	return string(*c)
}

func (m Message) ToDatagram() []byte {
	var b []byte
	return fmt.Appendf(b, "%s %d", m.Cmd.ToString(), m.Content)
}

func ParseMsg(b []byte) *Message {
	split := bytes.Split(b, []byte(" "))
	if len(split) == 0 {
		return nil
	}

	content, _ := strconv.Atoi(string(split[1]))

	return &Message{
		Cmd:     Command(split[0]),
		Content: content,
	}
}
