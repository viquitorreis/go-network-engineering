package main

import (
	"fmt"
	"time"
)

func main() {
	// Criar rate limiter: 10 tokens no máximo, adiciona 5 tokens a cada 100ms
	// Isso dá ~50 requests/segundo no steady state
	limiter := NewTokenBucket(10, 5, 100*time.Millisecond)

	fmt.Println("Starting rate limiter test...")
	fmt.Printf("Bucket: capacity=%d, refill rate=%d tokens per 100ms\n", 10, 5)

	// TODO: Simular múltiplas goroutines fazendo requests
	// Dica: use WaitGroup, lance ~50 goroutines, cada uma tenta Allow()
	// Conte quantas foram aceitas vs rejeitadas
	// Print os resultados

	fmt.Println("Test completed!")
}

// TokenBucket implementa rate limiting usando o algoritmo Token Bucket
type TokenBucket struct {
	// TODO: adicionar campos necessários
	// Dica: você precisa rastrear:
	// - quantos tokens existem agora
	// - capacidade máxima
	// - taxa de reabastecimento (tokens por intervalo)
	// - intervalo de reabastecimento
	// - timestamp do último refill
	// - mutex para proteger acesso concorrente
}

// NewTokenBucket cria um novo rate limiter
// capacity: número máximo de tokens no bucket
// refillRate: quantos tokens adicionar por intervalo
// refillInterval: com que frequência adicionar tokens (ex: 100ms)
func NewTokenBucket(capacity int, refillRate int, refillInterval time.Duration) *TokenBucket {
	// TODO: inicializar bucket começando cheio (todos os tokens disponíveis)
	return nil
}

// Allow tenta consumir um token
// Retorna true se conseguiu (requisição permitida)
// Retorna false se não tem tokens disponíveis (requisição rejeitada)
func (tb *TokenBucket) Allow() bool {
	// TODO: implementar lógica
	// Passos:
	// 1. Lock do mutex
	// 2. Calcular quantos tokens deveriam ter sido adicionados desde lastRefill
	// 3. Adicionar esses tokens (respeitando capacidade máxima)
	// 4. Atualizar lastRefill para agora
	// 5. Tentar consumir 1 token
	// 6. Unlock e retornar resultado
	return false
}

// refill é um método helper para calcular e adicionar tokens
// Retorna quantos tokens foram adicionados
func (tb *TokenBucket) refill() int {
	// TODO: implementar cálculo de refill
	// Quanto tempo passou desde lastRefill?
	// Quantos "intervalos" completos aconteceram nesse tempo?
	// Quantos tokens isso representa?
	// Não esqueça de respeitar a capacidade máxima!
	return 0
}

// Stats retorna estatísticas atuais do bucket (útil para debugging)
func (tb *TokenBucket) Stats() (available int, capacity int) {
	// TODO: retornar tokens disponíveis e capacidade
	// Precisa de lock? Por quê?
	return 0, 0
}
