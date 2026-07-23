# Health Check Poller with Circuit Breaker

🇺🇸 [English version](README.md)

**Categoria:** Concorrência / Networking
**Tempo estimado:** ~1h30

## O que é

Um health checker no estilo distribuído que faz polling de múltiplos endpoints HTTP concorrentemente em intervalos configuráveis, agrega o status de cada um, e implementa circuit breaker pra que um endpoint falhando persistentemente pare de receber requests. Comparável ao que um kubelet do Kubernetes ou um load balancer fazem pra decidir se um backend está saudável.

## O que você aprende

- Rodar loops de polling independentes por endpoint concorrentemente, sem que um endpoint lento afete os outros.
- Implementar o padrão circuit breaker: abrir o circuito depois de N falhas seguidas e pausar as checagens por um período de cooldown antes de tentar de novo.
- Agregar status atualizado concorrentemente numa única visão consultável, com um callback disparado nas transições de status.

## O que foi implementado

- `NewHealthPoller() *HealthPoller`, `AddEndpoint(config EndpointConfig)`, `Start()`, `Stop()`.
- `pollEndpoint` e `checkEndpoint` rodando um loop de polling por endpoint registrado.
- `aggregateResults()` coletando o status de cada endpoint num estado compartilhado.
- `GetStatus(endpoint string) (HealthStatus, bool)` e `GetAllStatuses() map[string]*HealthStatus`.
- Um callback `onStatusChange` disparado quando um endpoint transiciona entre saudável e não saudável.
- Os testes cobrem endpoint saudável, endpoint não saudável, abertura do circuit breaker, recuperação do circuito, o callback de mudança de status, múltiplos endpoints, graceful shutdown, timeout por request, e `-race`.

## Decisões de design

- Cada endpoint tem sua própria goroutine e timer de polling independentes, então um endpoint lento ou travado não consegue atrasar as checagens dos outros.
- O estado do circuit breaker (contagem de falhas, aberto/fechado) é rastreado por endpoint, não globalmente, já que endpoints diferentes podem estar saudáveis ou degradados de forma independente.

## Como rodar

```bash
go run .
go test ./...
```
