package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"net"
	"time"

	"miningpool/market"
	"miningpool/pool"
	"miningpool/protocol"
)

func main() {
	mp := market.NewMarketplace(0.05)
	server := pool.NewServer(":8080", mp)

	go func() {
		if err := server.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	// dá tempo pro listener subir antes de conectar
	time.Sleep(100 * time.Millisecond)

	minerID := simulateMiner(":8080", "cgminer/4.10.0")
	time.Sleep(2 * time.Second)
	mp.Tick(1.0) // 1 h
	fmt.Printf("Total hashrate: %.20f\n", mp.TotalHashrate())
	fmt.Printf("Revenue %s: $%.64f\n", minerID, mp.Revenue(minerID))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

// simulateMiner conecta via TCP real e envia as 3 mensagens do protocolo.
// Isso testa o protocolo de ponta a ponta sem precisar de um minerador real.
func simulateMiner(addr, userAgent string) string {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("miner connect error: %v", err)
		return ""
	}
	defer conn.Close()

	// 1. Subscribe
	sendJSON(conn, protocol.Message{
		ID: func() *int {
			num := rand.IntN(math.MaxInt)
			return &num
		}(),
		Method: protocol.Subscribe,
		Params: mustMarshal(protocol.SubscribeParams{
			UserAgent: userAgent,
		}),
	})

	// 2. le a response do servidor
	scanner := bufio.NewScanner(conn)
	scanner.Scan()

	var resp protocol.Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		log.Printf("miner response unmarshal error: %v", err)
		return ""
	}

	log.Printf("miner received response: %+v", resp)

	var result map[string]string
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		log.Printf("miner response result unmarshal error: %v", err)
		return ""
	}

	// id real no mktplc
	serverMinerID := result["session_id"]

	// 3. Submit uma share (simulada — hash pode ser inválido para este exemplo)
	sendJSON(conn, protocol.Message{
		ID: func() *int {
			num := rand.IntN(math.MaxInt)
			return &num
		}(),
		Method: protocol.Submit,
		Params: mustMarshal(protocol.SubmitParams{
			MinerID: serverMinerID,
			JobID:   "job-001",
			Nonce:   42,
			Hash:    "0000abcd1234", // leading zeros simulam dificuldade baixa
		}),
	})

	// dá tempo pro servidor processar a share antes de fechar a conexão
	time.Sleep(500 * time.Millisecond)

	return serverMinerID
}

func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func sendJSON(conn net.Conn, v any) {
	b, _ := json.Marshal(v)
	b = append(b, '\n') // Stratum usa newline como delimitador
	conn.Write(b)
}
