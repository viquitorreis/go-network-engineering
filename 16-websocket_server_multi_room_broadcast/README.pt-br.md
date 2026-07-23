# WebSocket Server - Multi-Room Broadcast

🇺🇸 [English version](README.md)

**Categoria:** Streaming / Networking
**Tempo estimado:** ~1 hora

## O que é

Um backend de chat WebSocket com múltiplas rooms independentes: clientes conectam via `/ws?room=<nome>`, e uma mensagem enviada numa room é distribuída (broadcast) só pros outros clientes daquela mesma room.

## O que você aprende

- Por que um `*websocket.Conn` precisa de uma goroutine de read pump e outra de write pump dedicadas por conexão: leitura e escrita concorrentes na mesma conexão são seguras, mas duas escritas concorrentes não são (os frames podem se intercalar e corromper o stream, e o `-race` não detecta isso).
- Estruturar um hub como uma única goroutine acessada só via channels, pra que o estado compartilhado de rooms/clientes nunca precise de mutex.
- Limpar uma conexão na desconexão sem vazar goroutines ou channels.

## O que foi implementado

- `NewHub() *Hub` rodando seu próprio loop de eventos (`Bootstrap()`) que é dono de `clients map[string]map[*Client]bool`, acessível só via channels `register`, `unregister` e `broadcast`.
- `NewClient(...)`, `readPump()`, `writePump()` por conexão.
- `ServeWs(hub *Hub, room string, w http.ResponseWriter, r *http.Request)` fazendo o upgrade da conexão HTTP e conectando um cliente numa room.
- Mensagens são distribuídas só dentro da room onde foram enviadas, nunca entre rooms diferentes.

## O que não foi implementado

O enunciado original (mantido em [`PROMPT.md`](./PROMPT.md)) listava dois itens bônus que **não** foram construídos: um heartbeat de ping/pong pra detectar conexões mortas, e um endpoint listando a contagem de clientes ativos por room. O servidor também não implementa graceful shutdown (roda um `http.ListenAndServe` bloqueante, sem tratamento de sinal). Registrando isso aqui em vez de fingir que está pronto.

## Decisões de design

- O hub é uma **goroutine única com channels**, não um map protegido por mutex: `clients` só é tocado pela goroutine rodando `Bootstrap()`, então não tem disputa de lock nem risco de leitura inconsistente no map de rooms.
- Leitura e escrita são separadas em duas goroutines por cliente justamente pra que um broadcast pro cliente A nunca fique bloqueado por A estar no meio de uma leitura de uma mensagem não relacionada.

## Como rodar

```bash
go run .
# abra o test.html no devtools do navegador e chame ws.send("msg")
```
