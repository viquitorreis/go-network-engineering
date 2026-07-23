# Mining Pool com Protocolo Stratum

🇺🇸 [English version](README.md)

**Categoria:** Networking / Protocolos
**Tempo estimado:** ~2 horas

## O que é

Uma implementação simplificada do protocolo Stratum (JSON-RPC sobre TCP, uma mensagem JSON por linha) usado por mineradores de Bitcoin pra se comunicar com mining pools, mais dois modelos de precificação diferentes pra distribuir valor de hashrate: um marketplace e uma exchange no estilo order book.

## O que você aprende

- Implementar um protocolo JSON-RPC delimitado por linha sobre TCP puro: parsing, dispatch e resposta pra tipos de mensagem distintos.
- As três mensagens principais do Stratum: `mining.subscribe` (handshake), `mining.notify` (trabalho enviado pelo servidor, server-push) e `mining.submit` (minerador reportando uma share).
- Modelar uma pool como um servidor que distribui trabalho e agrega resultados de muitos mineradores concorrentes ao mesmo tempo.

## O que foi implementado

- `pool/server.go`, `pool/dispatcher.go`, `pool/miner.go` implementando o lado da pool; `protocol/stratum.go` implementando a (de)serialização das mensagens.
- `simulateMiner(addr, userAgent string) string` e helpers de JSON (`mustMarshal`, `sendJSON`) pra simular mineradores contra a pool.
- Um modelo de precificação de marketplace e um modelo de order book pra hashrate, como dois caminhos de código distintos na lógica da pool.
- Os testes cobrem mineradores concorrentes, validação de shares, um tick de precificação do marketplace, broadcast sem deadlock, e matching no estilo exchange.

## Decisões de design

- As mensagens são despachadas por tipo através de um pacote `pool` central em vez de parseadas manualmente em cada handler de conexão, mantendo o parsing do protocolo separado da lógica de negócio da pool.
- A pool suporta dois modelos de precificação diferentes de propósito (marketplace vs order book), como forma de comparar os trade-offs entre um modelo mais simples de taxa fixa e um modelo no estilo motor de matching, em vez de escolher um só de cara.

## Como rodar

```bash
make execute
# ou
go run .
go test ./...
```
