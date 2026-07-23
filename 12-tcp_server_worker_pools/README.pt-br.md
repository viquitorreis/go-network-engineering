# TCP Server com Worker Pool e Backpressure

🇺🇸 [English version](README.md)

**Categoria:** Networking
**Tempo estimado:** ~1h30

## O que é

Um servidor TCP que desacopla aceitar conexões de processá-las: um pool fixo de goroutines worker puxa conexões de uma fila limitada, pra que o servidor consiga descartar carga de forma explícita em vez de criar uma goroutine ilimitada por conexão.

## O que você aprende

- A diferença entre o modelo ingênuo de "goroutine por conexão" e um worker pool com fila limitada, e por que o segundo dá controle real sobre o consumo de recursos.
- Implementar backpressure com `select` + `default`: rejeitar uma conexão imediatamente quando a fila está cheia em vez de bloquear o loop de accept.
- Graceful shutdown de um servidor TCP: parar o loop de accept, drenar conexões em andamento, e retornar de forma limpa.

## O que foi implementado

- `NewTCPServer(config ServerConfig, handler Handler) *TCPServer`.
- `Start() error` rodando o loop de accept e despachando conexões pra um pool fixo de workers via um channel limitado.
- `EchoHandler(conn *Connection) error` como handler de exemplo.
- `Addr() string` e `Shutdown() error`.
- Os testes cobrem o comportamento do echo server, backpressure (rejeição com fila cheia), e graceful shutdown.

## Decisões de design

- O `Accept()` nunca bloqueia esperando processamento: ele entrega a conexão pra um channel bufferizado e volta imediatamente pra aceitar a próxima; se esse channel está cheio, a conexão é rejeitada via `select`/`default` em vez de bloquear o loop de accept.
- O número de workers e o tamanho da fila são configuráveis via `ServerConfig`, tornando o teto de recursos explícito em vez de implícito.

## Como rodar

```bash
go run .
go test ./...
```
