# Go Systems Challenges

🇧🇷 [Versão em Português](README.pt-br.md)

![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)

A collection of 26 self-designed, senior-level Go challenges built to deepen practical knowledge of concurrency, network programming, and distributed systems patterns the kind of problems that show up in real backend systems and distributed systems and in interviews at infrastructure-focused companies.

I built these for myself because I couldn't find resources that went beyond "here's how goroutines and channels work" into the trade-offs that actually matter in production: lock granularity, backpressure, graceful shutdown, idempotency under retries, consensus. Sharing them in case they're useful for other engineers preparing for the same kind of interviews.

Each challenge folder has its own README with the problem context, what was actually implemented, and the relevant design decisions. The extended write-ups (problem statement + starting skeleton, mostly in Portuguese) live in [`README_FULL_NOTES.md`](./README_FULL_NOTES.md).

## Why this exists

Most Go concurrency resources stop at "here's how goroutines and channels work." There wasn't much available that pushed into the trade-offs that matter in production: lock granularity, backpressure, graceful shutdown ordering, idempotency under concurrent retries, consensus. This repo is the curriculum I built for myself to close that gap - one problem at a time, each with a clear set of learnings and, where present, a test suite that enforces correctness with `-race`.

## Challenges

| # | Challenge | Category | What it is / what you'll learn |
|---|-----------|----------|----------------------------------|
| 1 | [Event Bus (Fan-out)](./1_event-bus-fan-out) | Concurrency | Pub/sub event bus where one event fans out to N independent subscribers, each with its own buffered channel. Covers `sync.RWMutex` on shared subscriber maps and the classic goroutine-closure-over-loop-variable bug. |
| 2 | [Log Aggregator (Fan-in)](./2_log-aggregator-fan-in) | Concurrency | N producers writing logs into a single aggregator (fan-in). Covers `sync.WaitGroup` coordination and layered graceful shutdown (producers close → bridge closes → aggregator returns). |
| 3 | [Image Processing Pipeline](./3_image-processor-pipeline) | Concurrency / Pipeline | 4-stage concurrent pipeline (list → load → grayscale → save) connected by channels, each stage closing its own output channel and respecting `context` cancellation. |
| 4 | [TCP Chat Server](./4_tcp-chat-server) | Networking | Raw TCP chat server: a `sync.Cond`-based lobby that waits for a minimum number of players, and per-client broadcast channels so one slow client never blocks the others. |
| 5 | [Rate Limiter (Token Bucket)](./5_rate_limiter_token_bucket) | Concurrency | Thread-safe token bucket rate limiter with lazy refill (no background timer goroutine ticking per request) and safe concurrent `Allow()` calls. |
| 6 | [Worker Pool with Priority Queue](./6_worker_pool_priority_queue) | Concurrency / Data Structures | `container/heap`-based priority queue wrapped for concurrent use with `sync.Cond`, feeding a fixed pool of workers with bounded-queue backpressure. |
| 7 | [LRU Cache with TTL](./7_lru_cache_thread_safe_with_ttl) | Data Structures / Concurrency | Production-style cache: hashmap + doubly linked list for O(1) LRU eviction, plus active TTL cleanup running in the background. |
| 8 | [Health Check Poller with Circuit Breaker](./8_health_check_poller_with_circuit_breaker) | Concurrency / Networking | Polls multiple HTTP endpoints concurrently, aggregates health status, and opens a circuit breaker after N consecutive failures per endpoint. |
| 9 | [Thread-Safe Trie](./9_trie_thread_safe) | Data Structures / Concurrency | Autocomplete trie with fine-grained per-node ("hand-over-hand") locking instead of one global mutex, so unrelated prefix lookups don't contend. |
| 10 | [Skip List](./10_skip_list_thread_safe) | Data Structures / Concurrency | Probabilistic skip list (the structure behind Redis sorted sets), using the update-array pattern to safely rewire multiple levels on insert/delete. |
| 11 | [Exchange Order Book](./11_exchange_order_book) | Data Structures | Price-time priority matching engine with separate bid/ask heaps, matching orders when `max(bids) >= min(asks)`. |
| 12 | [TCP Server with Worker Pool & Backpressure](./12_tcp_server_worker_pools) | Networking | Decouples accepting TCP connections from processing them via a bounded queue, rejecting/shedding load instead of spawning an unbounded goroutine per connection. |
| 13 | [Idempotent Payment Processing (PostgreSQL)](./13_idempotent_payment_processing_postgresql) | Distributed Systems | Payment requests sharded by idempotency key, each shard processed sequentially by a single worker, with `ON CONFLICT DO NOTHING` at the DB layer to guarantee no double-charge on retry. |
| 14 | [Mining Pool with Stratum Protocol](./14_mining_pool_with_stratum_protocol) | Networking / Protocols | JSON-RPC-over-TCP protocol (Stratum) implementation for a mining pool, including both a marketplace and an order-book pricing model for hashrate. |
| 15 | [Raft Leader Election](./15_raft_leader_election) | Distributed Systems | Raft consensus leader election: terms, randomized election timeouts, heartbeats, and step-down on higher-term messages. |
| 16 | [WebSocket Server - Multi-Room Broadcast](./16-websocket_server_multi_room_broadcast) | Streaming / Networking | WebSocket chat backend with per-connection read/write pumps and a single-goroutine hub (no mutex) that routes broadcasts per room via channels. |
| 17 | [Worker Pool + Pipeline (Log Processing)](./17-worker-pool-pipeline) | Concurrency / Pipeline | 3-stage pipeline (generate → filter → rate-limited sink) exercising fan-in from multiple filter workers into one channel, closed only after all workers finish. |
| 18 | [Worker Pool Health Checker](./18-worker-pool-health-checker) | Concurrency | Fixed worker pool checking a list of URLs concurrently, applying a per-URL timeout from the outside (`select` + `time.After`) since the mocked check doesn't accept a `context.Context`. |
| 19 | [Graceful Shutdown Worker Pool](./19-graceful_shutdown_worker_pool) | Concurrency | Worker pool that drains in-flight jobs manually on `SIGTERM`/`SIGINT` - no `http.Server.Shutdown()` doing the work for you. |
| 20 | [Fan-out/Fan-in Rate Limiter](./20-fan-out-fan-in-rate-limiter) | Concurrency | N workers (fan-out) sharing one global rate limit via a shared `time.Ticker`, with all results converging back into a single channel (fan-in). |
| 21 | [Concurrent Log File Analyzer](./21-concurrent-log-file-analyzer) | Concurrency / I/O | Single sequential reader (`bufio.Scanner`) feeding N parsing workers, with one aggregator goroutine collecting counts - a template for CPU-bound-per-line log processing without loading the file into memory. |
| 22 | [TCP Multiplexed Stream Broker](./22-tcp_multiplexed_stream_broker) | Networking | Raw TCP broker with manual length-prefixed framing and topic multiplexing over a single connection; a centralized single-goroutine broker (register/unregister/broadcast via channels) plus per-connection read/write pumps. |
| 23 | [UDP Raw Client/Server with Simulated Loss](./23-udp_raw_client_server) | Networking | Connectionless UDP server tracking clients by address string (no `net.Conn` to key on), with configurable packet-loss simulation; client fires a burst of numbered messages without per-message acknowledgment, then reads back echoes within a single deadline window and reports which sequence numbers never returned. |
| 24 | [Reliable UDP: Seq Number + ACK + Retransmit](./24-udp_reliable_retransmit) | Networking | Stop-and-wait reliability layer on top of raw UDP: every message carries an incrementing sequence number, and the client retransmits the identical datagram on timeout if no ACK arrives; the server always re-acknowledges, but tracks seen sequence numbers per client address so a retransmitted duplicate is never reprocessed, only re-acked, keeping the protocol idempotent under packet loss on either the request or the response. p50 and p99 analysis and benchmarks |
| 25 | [Order Book Rewrite: Skip List + Doubly Linked Cancellation](./25-order_book_skip_list_doubly_linked) | Data Structures & Systems Design | Rewrites a heap-based matching engine's internals into a skip list ordered by price (replacing both the heap and the price-to-orders map with a single source of truth) plus a doubly linked list per price level (replacing a slice, so cancelling an order anywhere in the queue is O(1) instead of O(n)); documents the design tradeoffs, including why deleting a heap's arbitrary element forces an O(n log n) rebuild that a skip list avoids. |
| 26 | [RTP-Style Packetizer: Seq + Timestamp](./26-rtp_style_packetizer_seq_timeout) | Streaming / Networking | Implements timeliness over UDP instead of reliability: each datagram carries a sequence number and timestamp with no ACK and no retransmission, since a late audio frame is worthless once its playback moment has passed; the receiver detects gaps and out-of-order arrivals purely from the sequence number delta, deliberately avoiding the ACK/retry pattern from the reliable UDP challenge since it would defeat the point of RTP. |

## How to run

Most challenges are self-contained Go modules. The common pattern:

```bash
cd <challenge-folder>
go run .              # or `go run ./server` / `go run ./client` where the code is split
go test -race ./...   # for challenges that include a test suite
```

Some challenges expose a `Makefile` with `build`/`execute` targets instead - check the challenge's own README for the exact command. `13_idempotent_payment_processing_postgresql` additionally needs a running PostgreSQL instance.
