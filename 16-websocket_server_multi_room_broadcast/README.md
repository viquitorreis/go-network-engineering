# WebSocket Server - Multi-Room Broadcast

🇧🇷 [Versão em Português](README.pt-br.md)

**Category:** Streaming / Networking
**Estimated time:** ~1 hour

## What it is

A WebSocket chat backend for multiple independent rooms: clients connect via `/ws?room=<name>`, and a message sent in a room is broadcast only to other clients in that same room.

## What you'll learn

- Why a `*websocket.Conn` needs a dedicated read pump and write pump goroutine per connection: concurrent reads and writes on the same connection are safe, but two concurrent writes are not (frames can interleave and corrupt the stream, and `-race` won't catch it).
- Structuring a hub as a single goroutine reached only through channels, so the shared room/client state never needs a mutex.
- Cleaning up a connection on disconnect without leaking goroutines or channels.

## What's implemented

- `NewHub() *Hub` running its own event loop (`Bootstrap()`) that owns `clients map[string]map[*Client]bool`, reachable only via `register`, `unregister`, and `broadcast` channels.
- `NewClient(...)`, `readPump()`, `writePump()` per connection.
- `ServeWs(hub *Hub, room string, w http.ResponseWriter, r *http.Request)` upgrading the HTTP connection and wiring a client into a room.
- Messages are broadcast only within the room they were sent in, never across rooms.

## What's not implemented

The original problem statement (kept in [`PROMPT.md`](./PROMPT.md)) listed two bonus items that were **not** built: a ping/pong heartbeat to detect dead connections, and an endpoint listing active client counts per room. The server also does not implement graceful shutdown (it runs a blocking `http.ListenAndServe` with no signal handling). Noting this here instead of silently claiming it works.

## Design decisions

- The hub is a **single goroutine with channels**, not a mutex-protected map: `clients` is only ever touched by the goroutine running `Bootstrap()`, so there's no lock contention and no risk of a torn read on the room map.
- Read and write are split into two goroutines per client specifically so a broadcast to client A is never blocked by client A being mid-read on an unrelated incoming message.

## How to run

```bash
go run .
# open test.html in a browser devtools console and call ws.send("msg")
```
