# TCP Multiplexed Stream Broker

🇧🇷 [Versão em Português](README.pt-br.md)

**Category:** Networking
**Estimated time:** ~2 hours

## What it is

A raw TCP broker that multiplexes multiple topics over a single connection using a manually implemented length-prefixed framing protocol (no ready-made library for message boundaries), with a client able to subscribe to specific topics and receive only broadcasts for those.

## What you'll learn

- Implementing length-prefixed framing by hand over raw TCP: writing a length header before the payload so the reader knows exactly where one message ends and the next begins.
- Topic multiplexing over a single connection: routing messages to the right subscribers without opening a connection per topic.
- Structuring a broker as a single centralized goroutine reachable only through channels (register/unregister/broadcast), the same shape as the WebSocket hub in challenge 16, applied here to raw TCP.
- Graceful shutdown driven by `context` and OS signals (`SIGTERM`/`SIGINT`).

## What's implemented

- `server/`: `NewServer(ctx context.Context, port uint16) *Server`, `Boostrap(ctx context.Context)`, `handleConn`, `handleRead`, `routeCommand`, `AddToBroker`, `CloseClient`.
- `NewBroker(ctx context.Context) *Broker` with `Bootstrap`, `routeBroadcast`, `routeMessage` running as the single owner of subscriber state.
- `Client`: `NewClient`, `AddTopic(t Topic) bool`, `WriteFrame(data []byte) error` implementing the length-prefixed write side.
- `Topic.IsValid()` and `GetTopics()` for topic validation.
- `client/`: a separate `main.go` implementing `writeFrame`/`readFrame` against the broker as a standalone client binary.
- `main.go` wires graceful shutdown: a signal handler cancels the shared `context`, which the server's `Boostrap` loop respects.
- Tests cover frame writing using the length prefix, register-then-broadcast delivery, broadcast isolation across topics, multiple subscribers on the same topic both receiving, unregister stopping delivery, and topic validation.

## Design decisions

- The broker is a **single goroutine with channels** (register/unregister/broadcast), avoiding a mutex-protected subscriber map, the same pattern used in the WebSocket challenge, here applied to raw TCP framing instead of an existing WebSocket library.
- Framing is manual and length-prefixed rather than newline-delimited (as in the Stratum mining pool challenge), which allows binary-safe payloads instead of requiring line-based text.
- Server and client are separate Go binaries under `server/` and `client/`, each with their own `main.go`, rather than one binary with a mode flag.

## How to run

```bash
make run-server
# in another terminal:
make run-client

# or, without make:
go run ./server
go run ./client

go test -v ./server/...
```
