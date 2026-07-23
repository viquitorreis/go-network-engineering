# RELIABLE UDP: SEQ NUMBER + ACK + RETRANSMIT

**Category:** Network programming, raw
**Time:** 2h (15min theory + ~1h45 challenge)
**Builds on:** challenge 23 (raw UDP client/server with simulated loss)

## Overview

The simplest reliability model on top of UDP is **stop-and-wait with timeout**: send
a message, wait for an ACK, and if it doesn't arrive within a deadline, retransmit
the **same** message (same sequence number).

This introduces a problem that a plain UDP client/server wouldn't have to deal with:
**duplication**. If the server's ACK (not the message itself) gets lost somewhere in
that exchange, the client will retransmit something the server **already
processed**. The server needs to distinguish a genuinely new message from "a
retransmission of something I've already seen," or it ends up processing it twice.
This is called **idempotency via sequence number deduplication**: the server tracks
which sequence numbers it has already seen, and if it receives a repeat, it just
re-sends the ACK without reprocessing.

For reference: TCP solves this with a monotonic 32-bit sequence number plus a
sliding window. What you're building here is much simpler (stop-and-wait is a
window of size 1, one message in flight at a time, no parallelism yet).

## What to build

1. **Protocol with sequence numbers.** Every message sent carries an incrementing
   sequence number (`uint32`). Reuse the `types.Message` type already built in
   challenge 23, and add this field if it isn't there yet.
2. **The server sends an ACK, not a generic echo.** On receiving a message, the
   server responds with an ACK carrying the **same sequence number**, not the whole
   message back, just the confirmation that it was received.
3. **The server deduplicates.** It keeps a `map[uint32]bool` (or, per client,
   `map[string]map[uint32]bool`) of sequence numbers already processed. If it
   receives a repeated sequence number, it re-sends the ACK but **does not
   reprocess** the business logic (in this challenge, "processing" can be as simple
   as incrementing a counter; what matters is not incrementing it twice).
4. **The client has timeout and retransmit.** It sends a message, waits for an ACK
   for a short window (e.g. 500ms). If it doesn't arrive, it retransmits the
   **same** message (same sequence number, same payload). It repeats this up to a
   maximum number of attempts (e.g. 5), then gives up and reports a definitive
   failure for that sequence number.
5. **Keeps simulating loss on the server** (inherited from challenge 23), so you
   can actually observe retransmission happening, not just reason about it in
   theory.

### Required work

- Stop-and-wait: one message in flight at a time (no sliding window needed for this
  challenge, that would add too much complexity for the time available, but it's a
  valid bonus if time allows)
- Configurable timeout, retransmission up to a maximum number of attempts
- Server deduplicating by sequence number (idempotency)
- The client reports, after sending all N messages: how many were confirmed on the
  first attempt, how many needed retransmission (and how many times), how many
  failed definitively
- Handles these two cases separately: "didn't receive the ACK" (could be the
  message or the ack that was lost, you don't know which and don't need to) versus
  "received the ACK" (success, stop retrying that sequence number)

### Bonus (if time allows)

- Exponential backoff on the timeout between attempts (instead of a fixed wait)
- A metric distinguishing "sequence numbers that needed retransmission" from "only
  the message was lost" from "only the ACK was lost". This would require the
  server to log each type of simulated loss separately (request loss vs. ack loss)

## What will be evaluated

1. How you design the client-side state (what it needs to know about "message N is
   still unconfirmed" while waiting) without blocking the entire program on a
   single indefinitely-blocking `Read`
2. How the server guarantees that reprocessing a repeated sequence number never
   causes a duplicated side effect (deduplication is easy to forget to apply at
   some point in the flow)

---

First step: think about how the **ACK format** will be distinguishable from a
normal message's format. The server now sends two different kinds of things
(`Cmd` in `types.Message`, e.g. `MSGCmd` versus a new `ACKCmd`).

## Benchmarks

This protocol is reliable and functional with seq + ACK + retransmit in place.
Three metrics matter for this benchmark:

### 1. Effective retransmission rate vs. configured loss rate

If you configure 10% loss on the server, how many messages actually needed a retry
on the client?

This isn't a straightforward 1:1 relationship, since loss can happen either on the
way out (request) or on the way back (ACK), and both count as "needed a retry"
from the client's point of view, even though only one side actually lost a packet.
This means the observed retry rate tends to be **higher** than the configured loss
rate (10% loss outbound and 10% loss inbound don't simply add up to 10%; the
chance of something getting lost somewhere in the round trip is higher than the
loss on either side alone).

### 2. Latency (time to confirmation), with and without retry

A message that needs one retry pays the full cost of the `readTimeout` (e.g.
500ms) just waiting, before trying again. This is exactly the kind of number that
matters in a discussion like "what's the impact of X% loss on the protocol's p50 /
p99 latency?"

### 3. Definitive failure rate

This is when `maxRetries` is exhausted without confirmation.

It tells you whether the current parameters (5 attempts, 500ms timeout) are
adequate for a given loss level, or whether they break down too early.

## Loss and Retransmission Benchmark

Ran with `go test -bench=BenchmarkReliability -benchtime=200x`, simulating packet
loss on both the request and the ACK independently (see `Server.LossRate`), 200
messages per loss rate.

| Loss rate (each direction) | Retry rate | Failure rate | p50 latency | p99 latency |
|---|---|---|---|---|
| 0% | 0.0% | 0.0% | ~0ms | ~0ms |
| 5% | 11.0% | 0.0% | ~0ms | 201ms |
| 10% | 21.5% | 0.0% | ~0ms | 303ms |
| 20% | 38.0% | 1.0% | ~0ms | 303ms |
| 40% | 60.0% | 6.5% | 101ms | 505ms |

### Why retry rate is roughly double the configured loss rate

Loss is simulated independently on the request and the ACK, so a single attempt
only succeeds if both directions get through. With a per-direction loss rate `L`,
the chance of a single attempt succeeding is `(1-L)^2`, not `1-L`. At 40% loss per
direction, that works out to a 36% success rate per attempt, meaning roughly 64%
of messages need at least one retry. The observed 60% retry rate at the 40% loss
row lines up closely with that prediction.

Failure rate follows the same compounding effect, just raised to the number of
retries: `(1 - (1-L)^2)^maxRetries`. At 40% loss with 5 max retries, that predicts
about 10.7% of messages exhausting every retry, close to the observed 6.5%
(sample size is small at 200 messages per rate, so some variance is expected).

### Why p99 pulls away from p50 as loss increases

p50 stays near 0ms up through 20% loss because most messages still succeed on the
first attempt. p99 grows much faster: it's dominated by the minority of messages
that need one or more retries, each retry paying the full `readTimeout` (100ms in
this benchmark) before trying again. At 40% loss, p99 (505ms) is roughly 5x p50
(101ms), a direct measurement of the retry tail rather than the typical case.

### Notes

- `readTimeout` in this benchmark is 100ms, shorter than the 500ms used by the
  real client in normal operation, kept short here only to keep the benchmark
  fast to run
- Functional correctness tests (ACK matching, deduplication, per-client isolation)
  run against a server with 0% simulated loss, since introducing randomness there
  would make them flaky without testing anything additional beyond what this
  benchmark already covers deliberately