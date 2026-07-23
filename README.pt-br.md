# Go Systems Challenges

🇺🇸 [English version](README.md)

![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)

Uma coleção de 22 challenges de Go autoproduzidos, em nível sênior, construídos pra aprofundar conhecimento prático de concorrência, network programming e padrões de sistemas distribuídos o tipo de problema que aparece em backends reais e em entrevistas técnicas de empresas focadas em infraestrutura.

Construí isso pra mim mesmo porque não encontrei material que fosse além de "assim funcionam goroutines e channels" e entrasse nos trade-offs que realmente importam em produção: granularidade de lock, backpressure, graceful shutdown, idempotência sob retry, consenso. Compartilhando caso seja útil pra outros engenheiros se preparando pro mesmo tipo de problemas e/ou entrevistas.

Cada pasta de challenge tem seu próprio README com o contexto do problema, o que de fato foi implementado, e as decisões de design relevantes. Os textos estendidos (enunciado do problema + esqueleto inicial, majoritariamente em português) ficam em [`README_FULL_NOTES.md`](./README_FULL_NOTES.md).

## Por que isso existe

A maioria dos materiais de concorrência em Go para no "assim funcionam goroutines e channels". Não tinha muito material que entrasse nos trade-offs que importam em produção: granularidade de lock, ordem de graceful shutdown, idempotência sob retry concorrente, consenso. Esse repo é o currículo que montei pra mim mesmo pra fechar essa lacuna - um problema de cada vez, cada um com aprendizados claros e, quando existe, uma suíte de testes que garante correção com `-race`.

## Challenges

| # | Challenge | Categoria | O que é / o que você aprende |
|---|-----------|-----------|-------------------------------|
| 1 | [Event Bus (Fan-out)](./1_event-bus-fan-out) | Concorrência | Event bus pub/sub onde um evento é distribuído pra N subscribers independentes, cada um com seu próprio channel bufferizado. Cobre `sync.RWMutex` em maps compartilhados e o clássico bug de closure sobre variável de loop. |
| 2 | [Log Aggregator (Fan-in)](./2_log-aggregator-fan-in) | Concorrência | N produtores escrevendo logs num agregador único (fan-in). Cobre coordenação com `sync.WaitGroup` e graceful shutdown em camadas (produtores fecham → bridge fecha → agregador retorna). |
| 3 | [Image Processing Pipeline](./3_image-processor-pipeline) | Concorrência / Pipeline | Pipeline concorrente de 4 estágios (listar → carregar → converter pra grayscale → salvar) conectados por channels, cada estágio fechando seu próprio channel de saída e respeitando cancelamento por `context`. |
| 4 | [TCP Chat Server](./4_tcp-chat-server) | Networking | Servidor de chat em TCP puro: um lobby baseado em `sync.Cond` que espera um número mínimo de jogadores, e channels de broadcast por cliente pra que um cliente lento nunca trave os outros. |
| 5 | [Rate Limiter (Token Bucket)](./5_rate_limiter_token_bucket) | Concorrência | Rate limiter thread-safe com o algoritmo token bucket, com refill preguiçoso (sem goroutine de timer batendo a cada request) e `Allow()` seguro sob concorrência. |
| 6 | [Worker Pool com Priority Queue](./6_worker_pool_priority_queue) | Concorrência / Estruturas de Dados | Priority queue baseada em `container/heap`, envolvida com `sync.Cond` pra uso concorrente, alimentando um pool fixo de workers com backpressure via fila limitada. |
| 7 | [LRU Cache com TTL](./7_lru_cache_thread_safe_with_ttl) | Estruturas de Dados / Concorrência | Cache no estilo produção: hashmap + lista duplamente encadeada pra eviction O(1), mais limpeza ativa de TTL rodando em background. |
| 8 | [Health Check Poller com Circuit Breaker](./8_health_check_poller_with_circuit_breaker) | Concorrência / Networking | Faz polling concorrente de múltiplos endpoints HTTP, agrega o status de saúde e abre um circuit breaker depois de N falhas seguidas por endpoint. |
| 9 | [Trie Thread-Safe](./9_trie_thread_safe) | Estruturas de Dados / Concorrência | Trie de autocomplete com locking granular por node ("hand-over-hand") em vez de um mutex global, pra que buscas de prefixos diferentes não disputem o mesmo lock. |
| 10 | [Skip List](./10_skip_list_thread_safe) | Estruturas de Dados / Concorrência | Skip list probabilística (a estrutura por trás dos sorted sets do Redis), usando o padrão de update array pra reconectar múltiplos níveis com segurança no insert/delete. |
| 11 | [Exchange Order Book](./11_exchange_order_book) | Estruturas de Dados | Motor de matching com price-time priority, com heaps separadas de bid/ask, casando ordens quando `max(bids) >= min(asks)`. |
| 12 | [TCP Server com Worker Pool e Backpressure](./12_tcp_server_worker_pools) | Networking | Desacopla aceitar conexões TCP de processá-las via uma fila limitada - rejeita/descarta carga em vez de criar uma goroutine ilimitada por conexão. |
| 13 | [Idempotent Payment Processing (PostgreSQL)](./13_idempotent_payment_processing_postgresql) | Sistemas Distribuídos | Requisições de pagamento shardeadas por chave de idempotência, cada shard processado sequencialmente por um único worker, com `ON CONFLICT DO NOTHING` no banco pra garantir que retry nunca cobra duas vezes. |
| 14 | [Mining Pool com Protocolo Stratum](./14_mining_pool_with_stratum_protocol) | Networking / Protocolos | Implementação do protocolo Stratum (JSON-RPC sobre TCP) pra uma mining pool, incluindo dois modelos de precificação de hashrate: marketplace e order book. |
| 15 | [Raft Leader Election](./15_raft_leader_election) | Sistemas Distribuídos | Eleição de líder do consenso Raft: terms, timeouts de eleição randomizados, heartbeats e step-down ao encontrar mensagens com term maior. |
| 16 | [WebSocket Server - Multi-Room Broadcast](./16-websocket_server_multi_room_broadcast) | Streaming / Networking | Backend de chat WebSocket com read/write pump por conexão e um hub de goroutine única (sem mutex) que roteia broadcast por room via channels. |
| 17 | [Worker Pool + Pipeline (Log Processing)](./17-worker-pool-pipeline) | Concorrência / Pipeline | Pipeline de 3 estágios (gerar → filtrar → sink com rate limit) que exercita fan-in de múltiplos workers de filtro pra um único channel, fechado só depois que todos terminam. |
| 18 | [Worker Pool Health Checker](./18-worker-pool-health-checker) | Concorrência | Pool fixo de workers checando uma lista de URLs concorrentemente, aplicando timeout por URL de fora (`select` + `time.After`), já que a checagem mockada não recebe `context.Context`. |
| 19 | [Graceful Shutdown Worker Pool](./19-graceful_shutdown_worker_pool) | Concorrência | Worker pool que drena jobs em andamento manualmente ao receber `SIGTERM`/`SIGINT` - sem `http.Server.Shutdown()` fazendo o trabalho por você. |
| 20 | [Fan-out/Fan-in Rate Limiter](./20-fan-out-fan-in-rate-limiter) | Concorrência | N workers (fan-out) compartilhando um rate limit global via `time.Ticker` compartilhado, com todos os resultados convergindo de volta num único channel (fan-in). |
| 21 | [Concurrent Log File Analyzer](./21-concurrent-log-file-analyzer) | Concorrência / I/O | Um leitor sequencial (`bufio.Scanner`) alimentando N workers de parsing, com uma goroutine agregadora coletando as contagens - modelo pra processamento de log linha-a-linha sem carregar o arquivo inteiro na memória. |
| 22 | [TCP Multiplexed Stream Broker](./22-tcp_multiplexed_stream_broker) | Networking | Broker em TCP puro com framing manual length-prefixed e multiplexação de tópicos numa única conexão; broker centralizado de goroutine única (register/unregister/broadcast via channels) mais read/write pump por conexão. |

## Como rodar

A maioria dos challenges é um módulo Go independente. O padrão comum:

```bash
cd <pasta-do-challenge>
go run .              # ou `go run ./server` / `go run ./client` onde o código está separado
go test -race ./...   # nos challenges que têm suíte de testes
```

Alguns challenges expõem um `Makefile` com os targets `build`/`execute` em vez disso - confira o README de cada challenge pro comando exato. O `13_idempotent_payment_processing_postgresql` também precisa de uma instância PostgreSQL rodando.
