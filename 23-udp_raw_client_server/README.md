# 23, UDP Raw Client/Server with Simulated Loss

**Category:** Networking (raw, connectionless)
**Time:** ~2h

## Problem

Prototype the transport layer for a telemetry-style system (e.g. IoT sensors, real-time
game metrics) where throughput matters more than guaranteed delivery, hence UDP instead
of TCP. Before adding any reliability layer on top (sequence numbers, ACKs, retransmission),
first build the raw baseline and actually measure what gets lost.

UDP has no connection abstraction: the server listens on a single socket
(`net.ListenUDP`) and receives datagrams from any client on that same socket. There's no
`Accept()`, no per-connection goroutine, each read returns both the payload and the
sender's `*net.UDPAddr`, and identifying "who's talking to me" has to be done by address,
not by a `net.Conn`.

## What's implemented

**Server**
- Listens on a single UDP socket, no per-client connection state
- Tracks known clients in a map keyed by `addr.String()` (IP:port as string, comparing
  `*net.UDPAddr` pointers directly doesn't work, since `ReadFromUDP` returns a new pointer
  on every call even for the same physical sender)
- Simulates packet loss with a configurable probability (`rand.Float64() < lossRate`):
  a "dropped" packet is read off the socket but never processed or echoed back
- Echoes back every message that wasn't dropped, and counts successfully processed
  messages separately from total datagrams received

**Client**
- Sends a fixed burst of numbered messages (`numMsg = 100`) back-to-back, with no
  per-message wait, true fire-and-forget, matching how UDP is actually meant to be used
- After the burst, switches to a single read window with one absolute deadline
  (`SetReadDeadline`), reading back as many echoes as arrive until that deadline expires,
  not one deadline per message, which would serialize 100 sequential waits and badly
  distort the loss measurement
- Tracks which message numbers came back in a `map[int]bool`, then diffs against the
  full `0..99` range to report which ones never returned
- Handles out-of-order arrival correctly by design: responses are matched by the message
  number embedded in the payload, not by arrival order

## Design decisions worth noting

- **No `net.Conn` per client.** Since UDP has no connection, client identity is
  address-based (`net.UDPAddr.String()`), and that's the only viable map key, pointer
  identity on `*net.UDPAddr` is not stable across reads.
- **Burst-then-drain client loop, not request-response.** An earlier version of the
  client waited for a reply after every single send, which works but hides the real shape
  of UDP loss: with independent per-message timeouts, N consecutive drops serialize into
  N full timeout waits before the client even reaches the messages that succeeded. Splitting
  send and receive into two phases (or two goroutines) avoids that distortion.
- **A single "not received" ≠ a specific failure mode.** The client can't distinguish
  between the request never arriving, the response never arriving, or the server still
  processing, it only knows "no response within the deadline." That ambiguity is
  inherent to UDP and is exactly what the reliable-delivery version (seq numbers + ACK +
  retransmit) is meant to resolve later.

## What's *not* implemented (by design, for this challenge)

- No retransmission or acknowledgment protocol, that's the next step in the roadmap
- No round-trip latency measurement per message (listed as a stretch goal, not done here)
- No duplicate-packet simulation

## How to run

Terminal 1:
```bash
make run-server
```

Terminal 2:
```bash
make run-client
```

The client prints total messages sent and how many never received a response within the
read window, giving an approximate measurement of the server's configured loss rate.

## Tests

```bash
make test
```

Current test coverage: client-deduplication logic (`Server.addClient`) via unit tests.
The read/echo/loss loop itself lives inline in `main()` and isn't currently exported for
testing.