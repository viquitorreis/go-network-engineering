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