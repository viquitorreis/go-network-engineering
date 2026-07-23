package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

/*
═══════════════════════════════════════════════════════════════════════════
TODO - PASSOS PARA IMPLEMENTAR O TCP CHAT SERVER
═══════════════════════════════════════════════════════════════════════════

1. CRIAR O SERVER
   - Usar net.Listen para escutar em uma porta (ex: :6969)
   - Aceitar conexões de clientes com Accept() em um loop
   - Para cada cliente que conecta, criar uma goroutine

2. LOBBY (SALA DE ESPERA)
   - Usar sync.Cond para bloquear clientes até ter mínimo de 2 players
   - Quando cliente conecta, incrementar contador
   - Se atingir mínimo, fazer Broadcast() para liberar todos
   - Clientes ficam esperando em Wait() até o Broadcast

3. GERENCIAR CLIENTES
   - Guardar cada cliente em um map (clientID -> Client)
   - Cada cliente precisa de um channel para receber mensagens
   - Quando cliente envia mensagem, fazer broadcast para TODOS os outros

4. BROADCAST DE MENSAGENS
   - Ler mensagem do cliente A
   - Enviar para os channels de todos os outros clientes (B, C, D...)
   - Usar goroutines separadas: uma lê, outra escreve

5. HANDLE DISCONNECT
   - Quando cliente desconecta, remover do map
   - Notificar outros clientes
   - Fechar o channel do cliente que saiu

6. GRACEFUL SHUTDOWN
   - Context para cancelar tudo quando server parar
   - Fechar todas as conexões ativas
   - Esperar goroutines terminarem com WaitGroup

═══════════════════════════════════════════════════════════════════════════
*/

func main() {
	fmt.Println("=== TCP Chat Server com Lobby ===")
	fmt.Println("Network Programming: TCP + sync.Cond + Broadcast")

	// Criar servidor que espera mínimo de 2 players
	server := NewChatServer("6969", 2)

	// Context com timeout de 5 minutos
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Iniciar servidor em goroutine
	go func() {
		fmt.Println("👽 Server listening on :6969")
		fmt.Println("Connect with: telnet localhost 6969")
		fmt.Println("Waiting for at least 2 players to start...")

		if err := server.Start(ctx); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	// Simular alguns clients para teste
	time.Sleep(1 * time.Second)

	fmt.Println("To test:")
	fmt.Println("   Terminal 1: telnet localhost 6969")
	fmt.Println("   Terminal 2: telnet localhost 6969")
	fmt.Println("   Type messages and press Enter")
	fmt.Println("   Press Ctrl+C to stop server")

	// Manter servidor rodando
	<-ctx.Done()
	fmt.Println("Timeout reached, stopping server...")

	server.Stop()
	fmt.Println("Server stopped")
}

// Message representa uma mensagem no chat
type Message struct {
	From    string
	Time    string
	Content string
}

// Client representa um cliente conectado
type Client struct {
	id       uuid.UUID
	conn     net.Conn
	messages chan Message
}

// ChatServer gerencia o servidor de chat
type ChatServer struct {
	port          string
	clients       map[uuid.UUID]*Client
	clientCounter uint32
	minClients    uint32

	cond *sync.Cond
	wg   sync.WaitGroup
	mu   sync.Mutex
}

type IChatServer interface {
	Start(ctx context.Context) error
	Stop() error
}

// NewChatServer cria um novo servidor de chat
func NewChatServer(port string, minPlayers int) IChatServer {
	server := &ChatServer{
		port:       port,
		clients:    make(map[uuid.UUID]*Client),
		minClients: uint32(minPlayers),
		mu:         sync.Mutex{},
	}
	server.cond = sync.NewCond(&server.mu)
	return server
}

// Start inicia o servidor TCP
func (s *ChatServer) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		fmt.Println("err starting tcp server: ", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("err accepting connection: ", err)
			continue
		}

		select {
		case <-ctx.Done():
			log.Println("Couldnt create client. Context expired")
			continue
		default:
			go s.handleClient(ctx, conn)
		}
	}
}

// handleClient gerencia um cliente conectado
func (s *ChatServer) handleClient(ctx context.Context, conn net.Conn) {
	s.mu.Lock()

	client := &Client{
		id:       uuid.New(),
		conn:     conn,
		messages: make(chan Message, 100),
	}

	s.clients[client.id] = client
	s.clientCounter++

	if s.clientCounter >= s.minClients {
		log.Println("NEW LOBBY STABLISHED!")
		s.cond.Broadcast()
	} else {
		s.cond.Wait()
	}
	s.mu.Unlock()

	s.wg.Add(2)
	go func() {
		defer s.wg.Done()
		s.readLoop(ctx, client)
	}()

	go func() {
		defer s.wg.Done()
		s.writeLoop(ctx, client)
	}()

	s.wg.Wait()
	conn.Close()
}

// readLoop lê mensagens de um cliente
func (s *ChatServer) readLoop(ctx context.Context, client *Client) {
	// TODO:
	// 1. Criar um bufio.Scanner ou bufio.Reader na conexão
	// 2. Loop lendo linhas da conexão
	// 3. Para cada mensagem recebida, fazer broadcast para todos outros clientes
	// 4. Se erro de leitura (cliente desconectou), sair do loop

	reader := bufio.NewReader(client.conn)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("context expired on readLoop")
			return
		default:
			msg, err := reader.ReadBytes('\n')
			// if err == io.EOF {
			// 	// connection closed
			// 	log.Printf("client: %s disconnected from lobby", client.id)
			// 	return
			// }

			if err != nil {
				fmt.Println("err is: ", err)
				return
			}

			// fmt.Println("msg is: ", string(msg))

			s.broadcast(client.id, Message{
				From:    client.id.String(),
				Time:    time.Now().UTC().Format("2006-01-02 15:04:05"),
				Content: fmt.Sprintf("content: %s", msg),
			})
		}
	}
}

// writeLoop envia mensagens para um cliente
func (s *ChatServer) writeLoop(ctx context.Context, client *Client) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("ctx expired on writeLoop")
			return
		case msg, ok := <-client.messages:
			if !ok { // channel closed
				return
			}

			_, err := fmt.Fprintf(client.conn, "%s\n", msg)
			if err != nil {
				fmt.Println("err while writing")
				return
			}
		}
	}
}

// broadcast envia uma mensagem para todos os clientes exceto o remetente
func (s *ChatServer) broadcast(fromID uuid.UUID, content Message) {
	s.mu.Lock()
	for k, v := range s.clients {
		if k != fromID {
			select {
			case v.messages <- content:
			default:
				fmt.Println("skipping broadcast on client: ", v.id)
				continue
			}
		}
	}
	s.mu.Unlock()
}

// removeClient remove um cliente e notifica outros
func (s *ChatServer) removeClient(clientID uuid.UUID) {
	s.mu.Lock()
	client, exists := s.clients[clientID]
	if !exists {
		s.mu.Unlock()
		return
	}

	delete(s.clients, clientID)
	close(client.messages)
	s.clientCounter--
	s.mu.Unlock()

	s.broadcast(clientID, Message{
		From:    clientID.String(),
		Time:    time.Now().UTC().Format("2006-01-02 15:04:05"),
		Content: fmt.Sprintf("Client %s exited chat.", clientID),
	})
}

// Stop para o servidor gracefully
func (s *ChatServer) Stop() error {
	s.mu.Lock()

	clientIDs := make([]uuid.UUID, 0, len(s.clients))
	for id := range s.clients {
		clientIDs = append(clientIDs, id)
	}
	s.mu.Unlock()

	for _, id := range clientIDs {
		s.removeClient(id)
	}

	s.wg.Wait()

	return nil
}
