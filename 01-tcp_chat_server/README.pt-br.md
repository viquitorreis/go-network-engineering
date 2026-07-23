# TCP Chat Server

🇺🇸 [English version](README.md)

**Categoria:** Networking
**Tempo estimado:** ~1h30

## O que é

Um servidor de chat em TCP puro: clientes conectam direto num socket (ex: via `telnet`), esperam num lobby até um número mínimo de jogadores entrar, e depois têm suas mensagens distribuídas (broadcast) pra todo mundo mais que está conectado.

## O que você aprende

- Trabalhar com sockets `net.Conn` crus em vez de um framework HTTP: ler e interpretar bytes como mensagens, escrever bytes de volta.
- Usar `sync.Cond` pra acordar múltiplas goroutines esperando de uma vez quando uma condição muda (o lobby esperando jogadores suficientes).
- A regra de "nunca escrever na mesma conexão a partir de duas goroutines", resolvida com um channel de broadcast dedicado por cliente e uma goroutine de escrita dedicada por cliente.
- Por que um cliente lento nunca deve travar o broadcast pros clientes rápidos.

## O que foi implementado

- `NewChatServer(port string, minPlayers int) IChatServer` e `Start(ctx context.Context) error` pra aceitar conexões.
- Um lobby que bloqueia novos clientes (via `sync.Cond`) até `minPlayers` terem conectado.
- `handleClient`, `readLoop` e `writeLoop` por conexão: uma goroutine lê do socket, outra drena o channel de mensagens dedicado daquele cliente e escreve no socket.
- `broadcast` distribui uma mensagem pro channel de cada cliente conectado.
- `removeClient` e `Stop()` pra limpeza de conexão e desligamento do servidor.
- Os testes cobrem aceitar conexões, o lobby esperando o número mínimo de jogadores, broadcast, múltiplos clientes, desconexão de cliente, mensagens vazias, e rajadas rápidas de mensagens.

## Decisões de design

- Cada cliente tem seu próprio channel de saída e sua própria goroutine de escrita, então um leitor lento numa conexão nunca trava a entrega de broadcast pros outros.
- O lobby usa `sync.Cond` em vez de polling ou um semáforo baseado em channel, já que precisa acordar múltiplas goroutines esperando ao mesmo tempo assim que o número mínimo de jogadores é atingido.

## Como rodar

```bash
go run .
# em outro terminal:
telnet localhost 6969

go test ./...
```
