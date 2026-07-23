# TCP Server with Worker Pool and Backpressure

🇧🇷 [Versão em Português](README.pt-br.md)

**Category:** Networking
**Estimated time:** ~1.5 hours

## What it is

A TCP server that decouples accepting connections from processing them: a fixed pool of worker goroutines pulls connections from a bounded queue, so the server can shed load explicitly instead of spawning an unbounded goroutine per connection.

## What you'll learn

- The difference between the naive "goroutine-per-connection" model and a worker pool with a bounded queue, and why the latter gives you actual control over resource consumption.
- Implementing backpressure with `select` + `default`: rejecting a connection immediately when the queue is full instead of blocking the accept loop.
- Graceful shutdown of a TCP server: stopping the accept loop, draining in-flight connections, and returning cleanly.

## What's implemented

- `NewTCPServer(config ServerConfig, handler Handler) *TCPServer`.
- `Start() error` running the accept loop and dispatching connections to a fixed worker pool via a bounded channel.
- `EchoHandler(conn *Connection) error` as the example handler.
- `Addr() string` and `Shutdown() error`.
- Tests cover the echo server behavior, backpressure (queue-full rejection), and graceful shutdown.

## Design decisions

- `Accept()` never blocks on processing: it hands the connection to a buffered channel and immediately loops back to accept the next one; if that channel is full, the connection is rejected via `select`/`default` instead of blocking the accept loop.
- The number of workers and the queue size are both configurable via `ServerConfig`, making the resource ceiling explicit rather than implicit.

## How to run

```bash
go run .
go test ./...
```
