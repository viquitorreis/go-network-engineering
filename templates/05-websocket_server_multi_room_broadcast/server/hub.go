package server

type Message struct {
	room string
	data []byte
}

// broadcast é por room e não para todo mundo
type Hub struct {
	// string -> nome da room
	clients map[string]map[*Client]bool

	// mensagens chegando dos clients
	broadcast chan Message

	// registra requests dos clients
	register chan *Client

	// desregistra o client do hub
	unregister chan *Client
}

func NewHub() *Hub {
	hub := &Hub{
		clients:    make(map[string]map[*Client]bool),
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}

	go hub.Bootstrap()

	return hub
}

// o hub é uma goroutine única com channels
// só é acessado por UMA goroutine, que é a do próprio Hub, rodando um loop infinito com select,
// e todo mundo mais fala com ela por channels (broadcast, register, unregister)
func (h *Hub) Bootstrap() {
	for {
		select {
		case msg, ok := <-h.broadcast:
			if !ok {
				continue
			}

			for c := range h.clients[msg.room] {
				select {
				case c.send <- msg.data:
				default:
					close(c.send)
					delete(h.clients[msg.room], c)
				}
			}
		case client, ok := <-h.register:
			if !ok {
				continue
			}

			if h.clients[client.room] == nil {
				h.clients[client.room] = make(map[*Client]bool)
			}

			h.clients[client.room][client] = true
		case client, ok := <-h.unregister:
			if !ok {
				continue
			}

			if _, exists := h.clients[client.room][client]; exists {
				delete(h.clients[client.room], client)
				close(client.send)
			}
		}
	}
}
