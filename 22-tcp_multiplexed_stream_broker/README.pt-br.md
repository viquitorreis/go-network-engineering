# TCP Multiplexed Stream Broker

🇺🇸 [English version](README.md)

**Categoria:** Networking
**Tempo estimado:** ~2 horas

## O que é

Um broker em TCP puro que multiplexa vários tópicos numa única conexão usando um protocolo de framing length-prefixed implementado manualmente (sem lib pronta pra delimitar mensagens), com um cliente podendo se inscrever em tópicos específicos e receber só os broadcasts daqueles tópicos.

## O que você aprende

- Implementar framing length-prefixed na mão sobre TCP puro: escrever um header de tamanho antes do payload pra que o leitor saiba exatamente onde uma mensagem termina e a próxima começa.
- Multiplexação de tópicos numa única conexão: rotear mensagens pros subscribers certos sem abrir uma conexão por tópico.
- Estruturar um broker como uma única goroutine centralizada acessada só via channels (register/unregister/broadcast), o mesmo formato do hub WebSocket do challenge 16, aplicado aqui em TCP puro.
- Graceful shutdown controlado por `context` e sinais do SO (`SIGTERM`/`SIGINT`).

## O que foi implementado

- `server/`: `NewServer(ctx context.Context, port uint16) *Server`, `Boostrap(ctx context.Context)`, `handleConn`, `handleRead`, `routeCommand`, `AddToBroker`, `CloseClient`.
- `NewBroker(ctx context.Context) *Broker` com `Bootstrap`, `routeBroadcast`, `routeMessage` rodando como o único dono do estado de subscribers.
- `Client`: `NewClient`, `AddTopic(t Topic) bool`, `WriteFrame(data []byte) error` implementando o lado de escrita com length-prefix.
- `Topic.IsValid()` e `GetTopics()` pra validação de tópico.
- `client/`: um `main.go` separado implementando `writeFrame`/`readFrame` contra o broker, como um binário de cliente independente.
- `main.go` conecta o graceful shutdown: um handler de sinal cancela o `context` compartilhado, que o loop `Boostrap` do servidor respeita.
- Os testes cobrem escrita de frame usando o length prefix, entrega depois de register-then-broadcast, isolamento de broadcast entre tópicos, múltiplos subscribers no mesmo tópico recebendo, unregister parando a entrega, e validação de tópico.

## Decisões de design

- O broker é uma **goroutine única com channels** (register/unregister/broadcast), evitando um map de subscribers protegido por mutex, o mesmo padrão usado no challenge do WebSocket, aqui aplicado a framing TCP puro em vez de uma lib de WebSocket pronta.
- O framing é manual e length-prefixed em vez de delimitado por linha (como no challenge da mining pool com Stratum), o que permite payloads binary-safe em vez de exigir texto baseado em linha.
- Servidor e cliente são binários Go separados dentro de `server/` e `client/`, cada um com seu próprio `main.go`, em vez de um binário só com uma flag de modo.

## Como rodar

```bash
make run-server
# em outro terminal:
make run-client

# ou, sem make:
go run ./server
go run ./client

go test -v ./server/...
```
