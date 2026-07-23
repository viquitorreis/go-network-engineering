# RELIABLE UDP - SEQ NUMBER + ACK + RETRANSMIT

**Categoria**: Network programming raw
**Tempo**: 2h (15min teoria + ~1h45 challenge)
Builda em cima de: challenge 23 (UDP raw client/server com perda simulada)

## Visão Geral

O modelo mais simples de reliability sobre UDP é **stop-and-wait com timeout**, manda uma mensagem, espera ACK, se não vier dentro de um prazo, reenvia a **mesma** mensagem (mesmo seq number).

Nesse caso, isso vai introduzir um problema que em client/server UDP comum não teriamos: **duplicação**. Se o ACK (não a mensagem) do servidor for perdido no meio dessa comunicação, o client vai reenviar algo que o servidor **já processou**, o servidor precisa distinguir que é uma mensagem nova, de "reenvio de algo que já vi", se não processa duas vezes. O nome disso é **idempotência via deduplicação por seq number**: o servidor guarda quais seq numbers já viu e, se receber um repetido, só reenvia o ACK de novo, sem reprocessar.

Só para revisar: TCP resolve isso com número de sequência de 32 bits monotônico + janela deslizante, o que deve ser feito aqui é algo bem mais ismples (stop-and-wait é janela de tamanho 1, só uma mensagem por vez, sem paralelismo ainda)

## Contexto

1. **Protocolo com seq number**: cada mensagem enviada carrega um número sequencial (`uint32` incremental). Reaproveita o `types.Message` (o tipo da mensagem) que já existe no challenge 23 do udp raw, e adiciona esse campo (se ainda não tiver).
2. **Servidor manda ACK, não eco genérico**: ao receber uma mensagem, o servidor responde com um ACK carregando o **mesmo seq number**, e não a mensagem inteira de volta, só a confirmação de que recebeu.
3. **Servidor deduplica**: mantém um `map[uint32]bool` (ou por client, `map[string]map[uint32]bool`) de seq numbers já processados. Se receber um seq perdido, reenvia o ACK mas **não reprocessa** a lógica de negócio (nesse challenge, "processar" pode ser só incrementar um contador, o importante é não incrementar duas vezes)
4. **Client com timeout e retransmit**: manda mensagem, espera ACK por um prazo curto (ex: 500ms). Se não chegar, reenvia a **mesma** mensagem (mesmo seq, mesmo payload). Repete até um limite máximo de tentativas (ex: 5), depois desiste e reporta falha definitiva para aquele seq.
5. **Continua simulando perda no servidor** (herda do challenge 23): assim você consegue observar o retransmit acontecendo de verdade, não só na teoria.

### Requisitos obrigatórios

- Stop-and-wait: uma mensagem em voo por vez (não precisa de sliding window nesse challenge, isso complica demais pelo tempo disponível, mas pode ser bonus se tiver tempo)
- Timeout configurável, retransmissão até um número máximo de tentativas.
- Servidor deduplicando por seq number (idempotência)
- Client reporta, ao final do envio de todas as N mensagens: quantas foram confirmadas na primeira tentativa, quantas precisaram de retransmit (e quantas vezes), quantas falharam definitivamente
- Trata separadamente: "não recebi o ACK" (pode ser mensagem OU ack perdido, não sabemos qual e não precisa saber) vs "recebi ACK" (sucesso, para de tentar aquele seq)

### Bonus (se sobrar tempo)

- Exponential backoff no timeout entre tentativas (em vez de esperar tempo fixo)
- Métrica de "seq numbers que precisaram de retransmit" vs "só a mensagem se perdeu" vs "só o ACK se perdeu", isso precisaria do servidor para logar separadamente cada tipo de perda simulada (loss na request vs loss no ack)

## O que vai ser observado em um challenge desses

1. Como desenha o estado do lado do client (o que precisa saber sobre "mensagem N ainda não confirmada" enquanto espera) sem travar o programa inteiro num único `Read` bloqueante pra sempre
2. Como o servidor garante que reprocessar um seq repetido nunca causa efeito colateral duplicado (dedup é fácil de esquecer de aplicar em algum ponto do fluxo)

---

Primeiro passo: pensa como o **formato do ACK** vai ser distinguível do formato de uma mensagem normal, o servidor agora manda dois tipos de coisas diferentes (`Cmd` no `types.Message` tipo `MSGCmd` vs um novo `ACKCmd`) 

## Benchmarks

Esse protocolo é confiável e funcional com (seq + ACK + retransmit). 3 métricas vão importar nesse benchmark:

### 1. Taxa de retransmissão efetiva vs taxa de perda configurada

Se configurar 10% de perda no servidor, quantas mensagens de fato precisam de retry no client?

Isso não é 1:1 óbvio assim, pois a perda pode acontecer tanto na ida (request) quanto na volta (ACK), e as duas contam como "precisou retransmitir" do ponto de vista do client, mesmo que só uma tenha sido perdida de verdade. isso significa que a taxa de retry observada tende a ser **maior** que a taxa de perda configurada (perda de 10% na ida E 10% na volta não soma 10%, a chance de algo perder durante o round-trip é maior que a perda de um lado só).

### 2. Latência (tempo até confirmação), com e sem retry

Uma mensagem que precisa de 1 retry paga o custo do `readTimeout` inteiro (ex: 500ms) só esperando, antes de tentar de novo. Isso é o tipo de número que interessa em discussões "qual impacto de X% de perda na latência p50 / p99 do protocolo?"

### 3. Taxa de falha definitiva

Isso é quando esgota os `maxRetries` sem confirmar.

Isso te diz parâmetros atuais (5 tentativas, 500ms de timeout) são adequados para que nível de perda, ou se quebram cedo demais.

Esse protocolo é confiável e funcional com seq + ACK + retransmit implementados.
Três métricas importam nesse benchmark:

### 1. Taxa de retransmissão efetiva vs. taxa de perda configurada

Se você configurar 10% de perda no servidor, quantas mensagens de fato precisaram
de retry no client?

Isso não é uma relação direta de 1:1, já que a perda pode acontecer tanto na ida
(request) quanto na volta (ACK), e as duas contam como "precisou de retry" do
ponto de vista do client, mesmo que só um lado tenha perdido o pacote de verdade.
Isso significa que a taxa de retry observada tende a ser **maior** que a taxa de
perda configurada (10% de perda na ida e 10% na volta não somam simplesmente 10%;
a chance de algo se perder em algum ponto do round-trip é maior que a perda de
qualquer um dos lados sozinho).

### 2. Latência (tempo até confirmação), com e sem retry

Uma mensagem que precisa de um retry paga o custo total do `readTimeout` (ex:
500ms) só esperando, antes de tentar de novo. Esse é exatamente o tipo de número
que importa numa discussão do tipo "qual o impacto de X% de perda na latência p50
/ p99 do protocolo?"

### 3. Taxa de falha definitiva

Isso é quando o `maxRetries` se esgota sem confirmação.

Isso te diz se os parâmetros atuais (5 tentativas, 500ms de timeout) são
adequados pra um determinado nível de perda, ou se eles quebram cedo demais.

## Benchmark de Perda e Retransmissão

Rodado com `go test -bench=BenchmarkReliability -benchtime=200x`, simulando perda
de pacote tanto no request quanto no ACK, de forma independente (ver
`Server.LossRate`), 200 mensagens por taxa de perda.

| Taxa de perda (cada direção) | Taxa de retry | Taxa de falha | Latência p50 | Latência p99 |
|---|---|---|---|---|
| 0% | 0.0% | 0.0% | ~0ms | ~0ms |
| 5% | 11.0% | 0.0% | ~0ms | 201ms |
| 10% | 21.5% | 0.0% | ~0ms | 303ms |
| 20% | 38.0% | 1.0% | ~0ms | 303ms |
| 40% | 60.0% | 6.5% | 101ms | 505ms |

### Por que a taxa de retry é aproximadamente o dobro da taxa de perda configurada

A perda é simulada de forma independente no request e no ACK, então uma única
tentativa só tem sucesso se as duas direções passarem. Com uma taxa de perda `L`
por direção, a chance de uma única tentativa ter sucesso é `(1-L)²`, não `1-L`.
Com 40% de perda por direção, isso resulta numa taxa de sucesso de 36% por
tentativa, o que significa que aproximadamente 64% das mensagens precisam de pelo
menos um retry. A taxa de retry observada de 60% na linha de 40% de perda bate de
perto com essa previsão.

A taxa de falha segue o mesmo efeito composto, só que elevado ao número de
tentativas: `(1 - (1-L)²)^maxRetries`. Com 40% de perda e 5 tentativas máximas,
isso prevê cerca de 10,7% das mensagens esgotando todas as tentativas, próximo
dos 6,5% observados (o tamanho da amostra é pequeno, 200 mensagens por taxa,
então alguma variância é esperada).

### Por que o p99 se distancia do p50 conforme a perda aumenta

O p50 fica perto de 0ms até 20% de perda, porque a maioria das mensagens ainda
tem sucesso na primeira tentativa. O p99 cresce bem mais rápido: ele é dominado
pela minoria de mensagens que precisam de um ou mais retries, e cada retry paga o
`readTimeout` completo (100ms nesse benchmark) antes de tentar de novo. Com 40%
de perda, o p99 (505ms) é aproximadamente 5x o p50 (101ms), uma medição direta da
cauda de retries, e não do caso típico.

### Notas

- O `readTimeout` nesse benchmark é de 100ms, mais curto que os 500ms usados pelo
  client real em operação normal, mantido curto aqui só pra deixar o benchmark
  rápido de rodar
- Os testes funcionais de corretude (correspondência de ACK, deduplicação,
  isolamento por client) rodam contra um servidor com 0% de perda simulada, já
  que introduzir aleatoriedade ali deixaria eles instáveis (flaky) sem testar
  nada além do que esse benchmark já cobre de propósito