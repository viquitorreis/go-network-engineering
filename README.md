# Go Systems Challenges

🇧🇷 [Versão em Português](README.pt-br.md)

![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)

A collection of self designed senior level Go challenges focused on network programming: raw sockets, protocol design, framing, reliability, and the failure modes that only show up once you stop using a library and start writing the transport layer yourself.

These are a subset of a larger practice repo [Go Senior Challenges](https://github.com/viquitorreis/go-challenges). I pulled the network specific ones out here because they tell a more focused story: building up from a basic TCP server to specific protocol level concerns like idempotent retransmission and RTP style timing, without concurrency exercises or data structure drills mixed in.

Each challenge folder has its own README with the problem context, what was actually built, and the design decisions behind it.

## Why this exists

Most resources on network programming in Go stop at "here's `net.Listen`, here's `net.Dial`." There isn't much that goes into what happens once you have to design your own framing, handle partial reads, decide what "reliable" actually means for your protocol, or figure out why two TCP optimizations can silently add 40ms of latency to every request. This repo is where I worked through those problems directly, one protocol at a time.

## Challenges

| # | Challenge | Category | What it is / what you'll learn |
|---|-----------|----------|----------------------------------|
| 01 | [TCP Chat Server](./01-tcp_chat_server) | Networking | Raw TCP chat server: a `sync.Cond`-based lobby that waits for a minimum number of players, and per-client broadcast channels so one slow client never blocks the others. |
| 02 | [Health Check Poller with Circuit Breaker](./02-health_check_poller_with_circuit_breaker) | Networking | Polls multiple HTTP endpoints concurrently, aggregates health status, and opens a circuit breaker after N consecutive failures per endpoint. |
| 03 | [TCP Server with Worker Pool & Backpressure](./03-tcp_server_worker_pools) | Networking | Decouples accepting TCP connections from processing them via a bounded queue, rejecting or shedding load instead of spawning an unbounded goroutine per connection. |
| 04 | [Mining Pool with Stratum Protocol](./04-mining_pool_with_stratum_protocol) | Networking / Protocols | JSON-RPC-over-TCP protocol (Stratum) implementation for a mining pool, including both a marketplace and an order-book pricing model for hashrate. |
| 05 | [WebSocket Server: Multi-Room Broadcast](./05-websocket_server_multi_room_broadcast) | Streaming / Networking | WebSocket chat backend with per-connection read/write pumps and a single-goroutine hub (no mutex) that routes broadcasts per room via channels. |
| 06 | [TCP Multiplexed Stream Broker](./06-tcp_multiplexed_stream_broker) | Networking | Raw TCP broker with manual length-prefixed framing and topic multiplexing over a single connection; a centralized single-goroutine broker (register/unregister/broadcast via channels) plus per-connection read/write pumps. |
| 07 | [UDP Raw Client/Server with Simulated Loss](./07-udp_raw_client_server) | Networking | Connectionless UDP server tracking clients by address string, with configurable packet-loss simulation; client fires a burst of numbered messages without per-message acknowledgment, then reads back echoes within a single deadline window and reports which sequence numbers never returned. |
| 08 | [Reliable UDP: Seq Number + ACK + Retransmit](./08-udp_reliable_retransmit) | Networking | Stop-and-wait reliability layer on top of raw UDP: every message carries an incrementing sequence number, and the client retransmits the identical datagram on timeout if no ACK arrives; the server always re-acknowledges, but tracks seen sequence numbers per client address so a retransmitted duplicate is never reprocessed, only re-acked. Includes p50/p99 latency benchmarks under simulated packet loss. |
| 09 | [RTP-Style Packetizer: Seq + Timestamp](./09-rtp_style_packetizer_seq_timeout) | Streaming / Networking | Implements timeliness over UDP instead of reliability: each datagram carries a sequence number and timestamp with no ACK and no retransmission, since a late audio frame is worthless once its playback moment has passed. The receiver detects gaps and out-of-order arrivals from the sequence number alone, deliberately avoiding the ACK/retry pattern from the reliable UDP challenge since it would defeat the point of RTP. |

## How to run

Most challenges are self-contained Go modules. The common pattern:

```bash
cd <challenge-folder>
go run .              # or `go run ./server` / `go run ./client` where the code is split
go test -race ./...   # for challenges that include a test suite
```

Some challenges expose a Makefile with build/run targets instead check the challenge's own README for the exact command.
