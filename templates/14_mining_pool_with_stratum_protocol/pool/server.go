package pool

import (
	"bufio"
	"context"
	"log"
	"net"
	"sync"
)

// Server é o TCP server da mining pool.
// Aceita conexões de mineradores, faz parsing das mensagens Stratum,
// e despacha para os handlers corretos.
type Server struct {
	addr     string
	listener net.Listener

	mu     sync.RWMutex
	miners map[string]*MinerState // minerID -> estado

	dispatcher *Dispatcher
	market     HashrateMarket // interface — pode ser Marketplace ou Exchange

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// HashrateMarket é a interface comum entre Marketplace e Exchange.
// Isso permite que o Server não saiba qual modelo está usando.
type HashrateMarket interface {
	RegisterMiner(minerID string, hashrateTHs float64) error
	UnregisterMiner(minerID string)
	TotalHashrate() float64
	Revenue(minerID string) float64 // quanto esse minerador ganhou
}

func NewServer(addr string, market HashrateMarket) *Server {
	// TODO
	return nil
}

func (s *Server) Start() error {
	// TODO: net.Listen, loop de Accept, goroutine por conexão
	return nil
}

func (s *Server) handleConn(conn net.Conn) {
	// TODO: criar MinerState, bufio.Scanner com SplitFunc de newline,
	// loop de leitura, dispatch para handleMessage
}

func (s *Server) handleMessage(miner *MinerState, data []byte) {
	// TODO: protocol.Parse, switch em msg.Method:
	//   Subscribe -> gera SessionID, registra miner, envia response
	//   Submit    -> valida share, atualiza hashrate, responde
	//   Notify    -> mineradores não mandam notify, retornar erro
}

func (s *Server) Shutdown(ctx context.Context) error {
	// TODO: fechar listener, cancelar context, aguardar wg com timeout
	return nil
}

// BroadcastJob envia um novo job para todos os mineradores conectados.
// Chamado pelo Dispatcher quando um novo bloco chega.
func (s *Server) BroadcastJob(job *protocol.NotifyParams) {
	// TODO: RLock miners, iterar, SendMessage em goroutines separadas
	// por que goroutines separadas aqui? pensa em um minerador lento...
}

func (s *Server) Addr() string {
	// TODO: retornar listener addr
}
