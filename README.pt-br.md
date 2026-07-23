# Go Network Engineering

🇺🇸 [English version](README.md)

![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)

Uma coleção de challenges de Go autoproduzidos, em nível sênior, focados em network programming: sockets crus, design de protocolo, framing, confiabilidade, e os modos de falha que só aparecem quando você para de usar uma lib e passa a escrever a camada de transporte você mesmo.

Esses são um subconjunto de um repo de prática maior [Go Senior Challenges](https://github.com/viquitorreis/go-challenges). Separei os que são especificamente sobre rede porque juntos contam uma história mais focada: evoluindo de um servidor TCP básico até questões de nível de protocolo, como retransmissão idempotente e timing estilo RTP, sem misturar exercícios de concorrência ou estrutura de dados no meio.

Cada pasta de challenge tem seu próprio README com o contexto do problema, o que de fato foi construído, e as decisões de design por trás disso. As notas estendidas (enunciado do problema e esqueleto inicial, majoritariamente em português) ficam em [`README_FULL_NOTES.md`](./README_FULL_NOTES.md).

## Por que isso existe

A maioria dos materiais de network programming em Go para no "aqui está o `net.Listen`, aqui está o `net.Dial`". Não tem muito material que entre no que acontece quando você precisa desenhar seu próprio framing, lidar com leituras parciais, decidir o que "confiável" realmente significa pro seu protocolo, ou entender por que duas otimizações de TCP podem silenciosamente adicionar 40ms de latência em toda requisição. Esse repo é onde resolvi esses problemas diretamente, um protocolo de cada vez.

## Challenges

| # | Challenge | Categoria | O que é / o que você aprende |
|---|-----------|-----------|-------------------------------|
| 01 | [TCP Chat Server](./01-tcp_chat_server) | Networking | Servidor de chat em TCP puro: um lobby baseado em `sync.Cond` que espera um número mínimo de jogadores, e channels de broadcast por cliente pra que um cliente lento nunca trave os outros. |
| 02 | [Health Check Poller com Circuit Breaker](./02-health_check_poller_with_circuit_breaker) | Networking | Faz polling concorrente de múltiplos endpoints HTTP, agrega o status de saúde e abre um circuit breaker depois de N falhas seguidas por endpoint. |
| 03 | [TCP Server com Worker Pool e Backpressure](./03-tcp_server_worker_pools) | Networking | Desacopla aceitar conexões TCP de processá-las via uma fila limitada, rejeitando ou descartando carga em vez de criar uma goroutine ilimitada por conexão. |
| 04 | [Mining Pool com Protocolo Stratum](./04-mining_pool_with_stratum_protocol) | Networking / Protocolos | Implementação do protocolo Stratum (JSON-RPC sobre TCP) pra uma mining pool, incluindo dois modelos de precificação de hashrate: marketplace e order book. |
| 05 | [WebSocket Server: Multi-Room Broadcast](./05-websocket_server_multi_room_broadcast) | Streaming / Networking | Backend de chat WebSocket com read/write pump por conexão e um hub de goroutine única (sem mutex) que roteia broadcast por room via channels. |
| 06 | [TCP Multiplexed Stream Broker](./06-tcp_multiplexed_stream_broker) | Networking | Broker em TCP puro com framing manual length-prefixed e multiplexação de tópicos numa única conexão; broker centralizado de goroutine única (register/unregister/broadcast via channels) mais read/write pump por conexão. |
| 07 | [UDP Raw Client/Server com Perda Simulada](./07-udp_raw_client_server) | Networking | Servidor UDP sem conexão, rastreando clients por endereço, com simulação configurável de perda de pacote; o client dispara uma rajada de mensagens numeradas sem confirmação por mensagem, depois lê os ecos de volta dentro de uma única janela de prazo e reporta quais números de sequência nunca voltaram. |
| 08 | [Reliable UDP: Seq Number + ACK + Retransmit](./08-udp_reliable_retransmit) | Networking | Camada de confiabilidade stop-and-wait sobre UDP puro: cada mensagem carrega um número de sequência incremental, e o client retransmite o mesmo datagrama no timeout se nenhum ACK chegar; o servidor sempre reconfirma, mas rastreia os números de sequência já vistos por endereço de client, então uma retransmissão duplicada nunca é reprocessada, só reconfirmada. Inclui benchmarks de latência p50/p99 sob perda de pacote simulada. |
| 09 | [RTP-Style Packetizer: Seq + Timestamp](./09-rtp_style_packetizer_seq_timeout) | Streaming / Networking | Implementa timeliness sobre UDP em vez de confiabilidade: cada datagrama carrega um número de sequência e timestamp, sem ACK e sem retransmissão, já que um frame de áudio atrasado não vale nada depois que o momento de tocá-lo já passou. O receptor detecta gaps e chegadas fora de ordem só pelo número de sequência, evitando de propósito o padrão de ACK/retry do challenge de UDP confiável, já que isso contradiria o propósito do RTP. |

## Como rodar

A maioria dos challenges é um módulo Go independente. O padrão comum:

```bash
cd <pasta-do-challenge>
go run .              # ou `go run ./server` / `go run ./client` onde o código está separado
go test -race ./...   # nos challenges que têm suíte de testes
```

Alguns challenges expõem um `Makefile` com targets de build/run em vez disso — confira o README de cada challenge pro comando exato.