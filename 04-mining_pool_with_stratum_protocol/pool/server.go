package pool

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"miningpool/protocol"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
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
	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		addr:   addr,
		miners: make(map[string]*MinerState),
		market: market,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *Server) Start() error {
	// TODO: net.Listen, loop de Accept, goroutine por conexão
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		slog.Error("error starting tcp listener", "error", err)
		return err
	}

	s.mu.Lock()
	s.listener = listener
	s.mu.Unlock()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}

			slog.Error("error accepting connection", "error", err)
			continue
		}

		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	// TODO: criar MinerState, bufio.Scanner com SplitFunc de newline,
	// loop de leitura, dispatch para handleMessage
	defer s.wg.Done()
	defer conn.Close()

	miner := &MinerState{
		Conn:       conn,
		Writer:     bufio.NewWriter(conn),
		Difficulty: 4,
	}

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		// o MinerState é criado aqui
		s.handleMessage(miner, scanner.Bytes())
	}

	if miner.ID != "" {
		s.mu.Lock()
		delete(s.miners, miner.ID)
		s.mu.Unlock()
		// s.market.UnregisterMiner(miner.ID)
	}
}

func (s *Server) handleMessage(miner *MinerState, data []byte) {
	// TODO: protocol.Parse, switch em msg.Method:
	//   Subscribe -> gera SessionID, registra miner, envia response
	//   Submit    -> valida share, atualiza hashrate, responde
	//   Notify    -> mineradores não mandam notify, retornar erro
	msg, err := protocol.Parse(data)
	if err != nil {
		slog.Error("error parsing message", "error", err)
		return
	}

	switch msg.Method {
	case protocol.Subscribe:
		var params protocol.SubscribeParams
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			slog.Error("error unmarshaling subscripe params", "error", err)
			return
		}

		id := uuid.New().String()

		miner.ID = id
		miner.SessionID = id
		miner.ConnectedAt = time.Now()

		s.mu.Lock()
		s.miners[miner.ID] = miner
		s.mu.Unlock()

		result, _ := json.Marshal(map[string]string{"session_id": miner.SessionID})
		miner.SendMessage(&protocol.Response{
			ID:     *msg.ID,
			Result: result,
		})

	case protocol.Submit:
		var params protocol.SubmitParams
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			slog.Error("error unmarshaling submit params", "error", err)
			return
		}

		slog.Info("submit received", "hash", params.Hash, "difficulty", miner.Difficulty)
		valid := protocol.ValidateShare(params.Hash, miner.Difficulty)
		slog.Info("share validation", "valid", valid)

		// valid := protocol.ValidateShare(params.Hash, miner.Difficulty)
		if valid {
			entry := shareEntry{
				At:         time.Now(),
				Difficulty: miner.Difficulty,
			}
			miner.mu.Lock()
			miner.ShareWindow = append(miner.ShareWindow, entry)
			miner.LastShareAt = time.Now()
			miner.mu.Unlock()

			hashrate := miner.HashrateTHs(60)
			slog.Info("share accepted", "minerID", miner.ID, "hashrate", hashrate, "shares", len(miner.ShareWindow))
			s.market.RegisterMiner(miner.ID, hashrate)

			// s.market.RegisterMiner(miner.ID, miner.HashrateTHs(60))

			res, _ := json.Marshal(valid)
			miner.SendMessage(&protocol.Response{
				ID:     *msg.ID,
				Result: res,
			})
		}
	case protocol.Notify:
		slog.Warn("received notify from miner, ignoring", "minerID", miner.ID)
	default:
		slog.Warn("unknown method", "method", msg.Method)
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.cancel()
	s.listener.Close()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// BroadcastJob envia um novo job para todos os mineradores conectados.
// Chamado pelo Dispatcher quando um novo bloco chega.
func (s *Server) BroadcastJob(job *protocol.NotifyParams) {
	// TODO: RLock miners, iterar, SendMessage em goroutines separadas
	// por que goroutines separadas aqui? pensa em um minerador lento...
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, miner := range s.miners {
		m := miner
		go m.SendMessage(job)
	}
}

func (s *Server) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}
