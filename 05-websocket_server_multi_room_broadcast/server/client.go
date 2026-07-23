package server

import (
	"context"
	"log"
	"net/http"

	"github.com/coder/websocket"
)

type Client struct {
	hub  *Hub
	room string
	conn *websocket.Conn
	send chan []byte
}

func NewClient(h *Hub, conn *websocket.Conn, r string, sendCh chan []byte) *Client {
	return &Client{
		hub:  h,
		room: r,
		conn: conn,
		send: sendCh,
	}
}

// readPump envia mensagens do webscoket para o hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close(websocket.StatusNormalClosure, "closed")
	}()

	c.conn.SetReadLimit(1024)

	for {
		_, msg, err := c.conn.Read(context.Background())
		if err != nil {
			log.Printf("err is: %v", err)
			break
		}

		c.hub.broadcast <- Message{
			room: c.room,
			data: msg,
		}
	}
}

// writePump envia mensagem do hub para a conexão websocket
func (c *Client) writePump() {
	defer c.conn.Close(websocket.StatusGoingAway, "closed")

	ctx := context.Background()

	for msg := range c.send {
		if err := c.conn.Write(ctx, websocket.MessageBinary, msg); err != nil {
			log.Println("err writing", err)
			return
		}
	}
}

func ServeWs(hub *Hub, room string, w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // só pra dev/teste local
	})
	if err != nil {
		log.Println("err serving ws", err)
		return
	}

	client := NewClient(hub, conn, room, make(chan []byte, 256))

	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}
