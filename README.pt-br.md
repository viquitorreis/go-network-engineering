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
| 10 | [Jitter Buffer: Reordenação](./10-jitter_buffer_reordering) | Streaming / Networking | Reaproveita a skip list construída para o order book (mesma inserção O(log n) / front O(1), agora indexada por número de sequência em vez de preço) para segurar pacotes estilo RTP fora de ordem até que o horário programado de liberação chegue ou um deadline por pacote expire; um ticker drena a frente da lista para um channel de reprodução, então um pacote que nunca aparece é pulado em vez de travar a reprodução, e o player reconstrói os gaps de sequência esperados para reportar perda definitiva e latência de reprodução. |
| 11 | [UDP FEC (XOR): Pacotes de Sequência + Paridade + Reconstrução](./11-udp_fec_via_xor_seq_parity_packets) | Streaming / Networking | Adiciona correção de erro proativa (forward error correction) por cima do packetizer estilo RTP: o emissor faz XOR de cada grupo de tamanho fixo de pacotes de dados num pacote de paridade enviado junto, e o receptor reconstrói um único pacote faltante por grupo a partir dos sobreviventes mais a paridade, reportando perda definitiva só quando 2 ou mais pacotes do mesmo grupo estão faltando, já que paridade XOR simples não recupera mais que 1 perda por grupo. |
| 12 | [XDP: Programa Conta Pacotes por Porta](./12-xdp_program_counter_by_port) | Networking / Kernel bypass | Primeiro challenge de kernel bypass: um programa em C restrito anexado no hook XDP faz o parsing manual dos cabeçalhos Ethernet/IP/UDP e incrementa um contador por porta de destino num eBPF map, tudo antes do kernel sequer alocar um sk_buff; um loader Go em userspace (cilium/ebpf + bpf2go) carrega o bytecode já verificado, anexa ele numa interface, e consulta o mesmo map periodicamente pra contagens ao vivo, com os dois lados se comunicando só através de memória compartilhada gerenciada pelo kernel. |
| 13 | [XDP: Filtra/Dropa por Critério](./13-xdp_filters_drop_by_criteria) | Networking / Kernel bypass | Estende o contador de pacotes com um segundo eBPF map guardando uma blocklist controlada pelo Go, de forma que o programa do lado kernel dropa pacotes correspondentes via XDP_DROP antes de custarem qualquer processamento adicional do kernel, enquanto o loader em userspace atualiza a política em runtime sem recompilar; a separação espelha o padrão real de produção (Cilium, Katran) de um plano de controle em userspace decidindo política que um plano de dados em kernel só executa, além de um bug capturado onde pacotes eram lidos como IPv4 sem checar eth->h_proto antes, corrompendo silenciosamente o parsing de qualquer tráfego IPv6. |
| 14 | [Socket AF_XDP: Primeiro Attach — Redirect XDP Customizado via bpf2go](./14-af_xdp_socket) | Networking / Kernel Bypass | Anexa um programa XDP em C customizado (compilado via bpf2go, mesmo padrão dos challenges 12/13) que redireciona pacotes pra um socket AF_XDP registrado num XSKMAP; o lado userspace é Go escrito à mão usando syscall cru contra golang.org/x/sys/unix (registro de UMEM, setup de fill/rx ring via mmap, bind), em vez de um wrapper de terceiros pra AF_XDP, trocando o programa fixo hardcoded de libs como asavie/xdp por controle total sobre a lógica de redirect. |

## Como rodar

A maioria dos challenges é um módulo Go independente. O padrão comum:

```bash
cd <pasta-do-challenge>
go run .              # ou `go run ./server` / `go run ./client` onde o código está separado
go test -race ./...   # nos challenges que têm suíte de testes
```

Alguns challenges expõem um `Makefile` com targets de build/run em vez disso — confira o README de cada challenge pro comando exato.