# Health Check Poller with Circuit Breaker

🇧🇷 [Versão em Português](README.pt-br.md)

**Category:** Concurrency / Networking
**Estimated time:** ~1.5 hours

## What it is

A distributed-style health checker that polls multiple HTTP endpoints concurrently at configurable intervals, aggregates their status, and implements a circuit breaker so a persistently failing endpoint stops being hammered with requests. Comparable to what a Kubernetes kubelet or a load balancer does to decide if a backend is healthy.

## What you'll learn

- Running independent polling loops per endpoint concurrently without one slow endpoint affecting the others.
- Implementing the circuit breaker pattern: opening the circuit after N consecutive failures and pausing checks for a cooldown period before retrying.
- Aggregating concurrently-updated status into a single queryable view, with a callback fired on status transitions.

## What's implemented

- `NewHealthPoller() *HealthPoller`, `AddEndpoint(config EndpointConfig)`, `Start()`, `Stop()`.
- `pollEndpoint` and `checkEndpoint` running one polling loop per registered endpoint.
- `aggregateResults()` collecting per-endpoint status into shared state.
- `GetStatus(endpoint string) (HealthStatus, bool)` and `GetAllStatuses() map[string]*HealthStatus`.
- An `onStatusChange` callback fired when an endpoint transitions between healthy and unhealthy.
- Tests cover a healthy endpoint, an unhealthy endpoint, the circuit breaker opening, circuit recovery, the status-change callback, multiple endpoints, graceful shutdown, per-request timeout, and `-race`.

## Design decisions

- Each endpoint gets its own independent polling goroutine and timer, so a slow or hanging endpoint can't delay checks against the others.
- The circuit breaker state (failure count, open/closed) is tracked per endpoint, not globally, since different endpoints can be healthy or degraded independently.

## How to run

```bash
go run .
go test ./...
```
