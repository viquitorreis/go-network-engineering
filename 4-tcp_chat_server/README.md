# TCP Chat Server

🇧🇷 [Versão em Português](README.pt-br.md)

**Category:** Networking
**Estimated time:** ~1.5 hours

## What it is

A raw TCP chat server: clients connect directly to a socket (e.g. via `telnet`), wait in a lobby until a minimum number of players has joined, and then have their messages broadcast to everyone else connected.

## What you'll learn

- Working with raw `net.Conn` sockets instead of an HTTP framework: reading and interpreting bytes as messages, writing bytes back.
- Using `sync.Cond` to wake up multiple waiting goroutines at once when a condition changes (the lobby waiting for enough players).
- The "never write to the same connection from two goroutines" rule, solved with a dedicated per-client broadcast channel plus a dedicated write goroutine per client.
- Why a slow client should never block broadcasts to fast clients.

## What's implemented

- `NewChatServer(port string, minPlayers int) IChatServer` and `Start(ctx context.Context) error` to accept connections.
- A lobby that blocks new clients (via `sync.Cond`) until `minPlayers` have connected.
- `handleClient`, `readLoop`, and `writeLoop` per connection: one goroutine reads from the socket, another drains that client's dedicated message channel and writes to the socket.
- `broadcast` fans a message out to every connected client's channel.
- `removeClient` and `Stop()` for connection cleanup and server shutdown.
- Tests cover accepting connections, the lobby waiting for the minimum player count, broadcasting, multiple clients, client disconnects, empty messages, and rapid message bursts.

## Design decisions

- Each client has its own outbound channel and its own write-pump goroutine, so a slow reader on one connection never blocks broadcast delivery to the others.
- The lobby uses `sync.Cond` instead of polling or a channel-based semaphore, since it needs to wake multiple waiting goroutines simultaneously once the player threshold is met.

## How to run

```bash
go run .
# in another terminal:
telnet localhost 6969

go test ./...
```
