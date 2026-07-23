# Mining Pool with Stratum Protocol

🇧🇷 [Versão em Português](README.pt-br.md)

**Category:** Networking / Protocols
**Estimated time:** ~2 hours

## What it is

A simplified implementation of the Stratum protocol (JSON-RPC over TCP, one JSON message per line) used by Bitcoin miners to talk to mining pools, plus two different pricing models for distributing hashrate value: a marketplace and an order-book exchange.

## What you'll learn

- Implementing a line-delimited JSON-RPC protocol over raw TCP: parsing, dispatch, and responding to distinct message types.
- The three core Stratum messages: `mining.subscribe` (handshake), `mining.notify` (server-push work assignment), and `mining.submit` (miner reporting a share).
- Modeling a pool as a server that both distributes work and aggregates results from many concurrent miners.

## What's implemented

- `pool/server.go`, `pool/dispatcher.go`, `pool/miner.go` implementing the pool side; `protocol/stratum.go` implementing message (de)serialization.
- `simulateMiner(addr, userAgent string) string` and JSON helpers (`mustMarshal`, `sendJSON`) for simulating miners against the pool.
- Both a marketplace pricing model and an order-book pricing model for hashrate, as two distinct code paths in the pool logic.
- Tests cover concurrent miners, share validation, a marketplace pricing tick, broadcast without deadlock, and exchange-style matching.

## Design decisions

- Messages are dispatched by type through a central `pool` package rather than parsed ad hoc per connection handler, keeping protocol parsing separate from pool business logic.
- The pool intentionally supports two different pricing models (marketplace vs. order book) as a way to compare trade-offs between a simpler flat-rate model and a matching-engine-style model, rather than picking one upfront.

## How to run

```bash
make execute
# or
go run .
go test ./...
```
