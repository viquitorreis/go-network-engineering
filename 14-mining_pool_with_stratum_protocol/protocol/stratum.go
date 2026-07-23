package protocol

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

// MessageType representa os 3 tipos de mensagem do Stratum simplificado.
// Em produção (SlushPool, F2Pool) existem mais, mas esses 3 cobrem o core.
type MessageType string

const (
	Subscribe MessageType = "mining.subscribe"
	Notify    MessageType = "mining.notify"
	Submit    MessageType = "mining.submit"
)

// Message é o envelope JSON que toda mensagem Stratum usa.
// O campo Params é raw porque cada tipo de mensagem tem params diferentes —
// você vai fazer unmarshal do params específico depois de conhecer o Method.
type Message struct {
	ID     *int            `json:"id"` // nil para server-push (notify)
	Method MessageType     `json:"method"`
	Params json.RawMessage `json:"params"`
}

// Response é o que o server manda de volta para requests com ID.
type Response struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *RPCError       `json:"error"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// SubscribeParams são os params do mining.subscribe.
type SubscribeParams struct {
	UserAgent string `json:"user_agent"` // ex: "cgminer/4.10.0"
	SessionID string `json:"session_id"` // vazio na primeira conexão
}

// NotifyParams são os params do mining.notify (server -> miner).
type NotifyParams struct {
	JobID      string `json:"job_id"`
	PrevHash   string `json:"prev_hash"`
	Difficulty uint64 `json:"difficulty"` // alvo simplificado: share válida se hash < target
	CleanJobs  bool   `json:"clean_jobs"` // true = descarta jobs anteriores
}

// SubmitParams são os params do mining.submit (miner -> server).
type SubmitParams struct {
	MinerID string `json:"miner_id"`
	JobID   string `json:"job_id"`
	Nonce   uint64 `json:"nonce"` // o minerador encontrou esse nonce
	Hash    string `json:"hash"`  // sha256 do bloco com esse nonce
}

// Parse faz o dispatch do JSON cru para o tipo correto.
// Retorna um dos tipos acima ou erro se o método for desconhecido.
func Parse(data []byte) (*Message, error) {
	// TODO: unmarshal em Message, validar Method
	var msg *Message
	if err := json.Unmarshal(data, &msg); err != nil {
		slog.Error("error unmarshaling json", "error", err)
		return nil, err
	}

	switch msg.Method {
	case Subscribe, Notify, Submit:
		break
	default:
		return nil, fmt.Errorf("received invalid message method: %v", msg.Method)
	}

	return msg, nil
}

// ValidateShare verifica se o hash submetido satisfaz a dificuldade do job.
// Simplificação: conta leading zeros no hash (hex string).
// Em Bitcoin real, compara com um target de 256 bits.
func ValidateShare(hash string, difficulty uint64) bool {
	var zeros uint64
	for _, char := range hash[:8] {
		if char != '0' {
			break
		}
		zeros++
	}
	return zeros >= difficulty
}
