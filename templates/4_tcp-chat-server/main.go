package main

import (
	"context"
	"fmt"
	"net"
	"time"
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

	// Simular alguns clients para teste (remova isso depois)
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
	Content string
	Time    time.Time
}

// Client representa um cliente conectado
type Client struct {
	// TODO: adicione campos necessários:
	// - ID único do cliente
	// - Conexão net.Conn
	// - Channel para receber mensagens
	// - Reader/Writer para ler/escrever na conexão
}

// ChatServer gerencia o servidor de chat
type ChatServer struct {
	// TODO: adicione campos necessários:
	// - Porta do servidor
	// - Map de clientes ativos
	// - Mutex para proteger o map
	// - sync.Cond para o lobby
	// - Contador de clientes
	// - Mínimo de players para começar
	// - WaitGroup para coordenação
}

type IChatServer interface {
	Start(ctx context.Context) error
	Stop() error
}

// NewChatServer cria um novo servidor de chat
func NewChatServer(port string, minPlayers int) IChatServer {
	return &ChatServer{
		// TODO: inicialize os campos
	}
}

// Start inicia o servidor TCP
func (s *ChatServer) Start(ctx context.Context) error {
	// TODO:
	// 1. Criar listener TCP com net.Listen("tcp", ":"+port)
	// 2. Loop infinito aceitando conexões com listener.Accept()
	// 3. Para cada conexão aceita, criar um Client e chamar handleClient em goroutine
	// 4. Respeitar ctx.Done() para shutdown

	// Hint:
	// listener, err := net.Listen("tcp", ":6969")
	// for { conn, err := listener.Accept() }

	return nil
}

// handleClient gerencia um cliente conectado
func (s *ChatServer) handleClient(ctx context.Context, conn net.Conn) {
	// TODO:
	// 1. Criar um Client com ID único e conexão
	// 2. Adicionar cliente ao map (thread-safe com mutex)
	// 3. LOBBY: Verificar se atingiu mínimo de players
	//    - Se sim, fazer Broadcast() para liberar todos
	//    - Se não, fazer Wait() para esperar
	// 4. Iniciar duas goroutines:
	//    - readLoop: lê mensagens do cliente
	//    - writeLoop: envia mensagens para o cliente
	// 5. Quando cliente desconectar, remover do map e notificar outros

	defer conn.Close()

	// Hint: Use sync.Cond para o lobby
	// s.cond.L.Lock()
	// s.playerCount++
	// if s.playerCount >= s.minPlayers {
	//     s.cond.Broadcast()
	// } else {
	//     s.cond.Wait()
	// }
	// s.cond.L.Unlock()
}

// readLoop lê mensagens de um cliente
func (s *ChatServer) readLoop(ctx context.Context, client *Client) {
	// TODO:
	// 1. Criar um bufio.Scanner ou bufio.Reader na conexão
	// 2. Loop lendo linhas da conexão
	// 3. Para cada mensagem recebida, fazer broadcast para todos outros clientes
	// 4. Se erro de leitura (cliente desconectou), sair do loop

	// Hint:
	// scanner := bufio.NewScanner(client.conn)
	// for scanner.Scan() {
	//     msg := scanner.Text()
	//     s.broadcast(client.ID, msg)
	// }
}

// writeLoop envia mensagens para um cliente
func (s *ChatServer) writeLoop(ctx context.Context, client *Client) {
	// TODO:
	// 1. Loop lendo do channel de mensagens do cliente
	// 2. Para cada mensagem, escrever na conexão
	// 3. Se ctx cancelar ou channel fechar, sair

	// Hint:
	// for msg := range client.messages {
	//     fmt.Fprintf(client.conn, "%s: %s\n", msg.From, msg.Content)
	// }
}

// broadcast envia uma mensagem para todos os clientes exceto o remetente
func (s *ChatServer) broadcast(fromID string, content string) {
	// TODO:
	// 1. Criar uma mensagem
	// 2. Iterar pelo map de clientes (thread-safe com mutex)
	// 3. Para cada cliente diferente do remetente:
	//    - Enviar mensagem para o channel do cliente
	//    - Usar select com default para não bloquear se channel cheio

	// Hint: Use select com default para não bloquear
	// select {
	// case client.messages <- msg:
	// default:
	//     // Cliente está lento/desconectado, pular
	// }
}

// removeClient remove um cliente e notifica outros
func (s *ChatServer) removeClient(clientID string) {
	// TODO:
	// 1. Lock no mutex
	// 2. Remover cliente do map
	// 3. Fechar o channel de mensagens do cliente
	// 4. Unlock no mutex
	// 5. Fazer broadcast que o cliente saiu
	// 6. Decrementar playerCount
}

// Stop para o servidor gracefully
func (s *ChatServer) Stop() error {
	// TODO:
	// 1. Fechar listener
	// 2. Fechar todas as conexões ativas
	// 3. Esperar WaitGroup terminar

	return nil
}