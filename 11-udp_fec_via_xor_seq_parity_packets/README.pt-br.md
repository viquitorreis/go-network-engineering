# FEC (XOR) - SEQ DE PACOTES + PARIDADE + RECONSTRUÇÃO

**Categoria**: Streaming / Network programming raw
**Tempo**: 2h (15min teoria + ~1h45 challenge)
**Builda em cima de**: challenge 09 (RTP-style packetizer com seq + timestamp). Reaproveita o mesmo formato de pacote com seq+timestamp

## Estudo

### O que é FEC (Forward Error Correction) via XOR

Vale lembrar: diferença entre **reliability** (ACK+retransmit, challenge 08) e **timeliness** (RTP, challenge 09), FEC é uma **terceira estratégia de **controle de erro no transporte**.

O FEC resolve um problema que nenhuma outra resolve bem: se não pode esperar o tempo de um retransmit (algo enviado em tempo real), mas também não pode simplesmente aceitar a perda.

A ideia central: em vez de reagir à perda depois que ela acontece (retransmitir), você manda redundância junto com os dados, de forma que o receptor consiga reconstruir um pacote perdido sem precisar pedir de novo. Isso evita o custo de um round-trip inteiro (que o retransmit sempre paga) a correção já está embutida no que chegou.

### Por quê usar XOR?

XOR tem uma paridade que vem da matemática muito útil: se você tem:

```
A XOR B XOR C = P
```

Sendo P = pacote de paridade, e perde qualquer **um** dos quatro (digamos, `B`), conseguimos recuperar fazendo `B = A XOR C XOR P`. Isso é literalmente um RAID5 aplicado a pacotes de rede ao invés de discos, mesmo princípio mas domínio diferente.

**Trade-off**: manda 1 pacote de paridade a cada N pacotes de dados (ex: 1 a cada 4). Isso é overhead de banda **constante e previsível**, que se paga SEMPRE mesmo quando não tem perda alguma na comunicação.

Comparado com o retransmit: lá só pagamos o custo **quando** perde (mas paga em latência, não banda). FEC (proativo, Forward Error Correction) troca banda extra por proteção contra perda **sem** esperar retransmit. Ótimo para tempo real onde 1 pacote perdido sem correção significa artefato visível / auditável, mas retransmit chegaria tarde demais.

## Contexto:

Hoje o sistema de streaming de áudio (challenges 9 e 10) hoje só detecta e reordena, quando um pacote se perde de vez, ele fica perdido. Hoje adicionamos uma camada que **recupera** parte dessas perdas sem pedir permissão, mandando um pacote de paridade a cada grupo de N pacotes de dados.

## O que construir:

1. **Agrupamento em blocos FEC:**: o sender agrupa pacotes de dados em grupos fixos (ex: 4 pacotes de dados), calcula o XOR de todos eles, e manda esse resultado como **5º pacote** (o de paridade), identificado por um campo indicando "isso é paridade do grupo X"

2. **Simulação de perda continua**: (reaproveita rand.Float64() < lossRate de challenges anteriores)

3. **Reconstrução no receptoro**: se exatamente 1 pacote do grupo de 5 (4 dados + 1 paridade) se perder, o receptor reconstrói ele fazendo XOR de todos os outros que recebeu se 2 ou mais se perderem no mesmo grupo, XOR simples não recupera nada (limitação real, precisa reportar como perda definitiva)

4. **Relatório**: quantos pacotes foram recuperados via FEC vs quantos foram perda definitiva (grupo com 2+ perdas) — e o overhead de banda real (quantos pacotes de paridade foram enviados no total)

## Requisitos obrigatórios

- Tamanho de grupo configurável (ex: 4 dados + 1 paridade)
- XOR calculado corretamente sobre o payload dos pacotes do grupo
- Reconstrução funcionando quando exatamente 1 pacote do grupo falta
- Detecção honesta de "não recuperável" quando 2+ faltam no mesmo grupo
- Benchmark: taxa de recuperação efetiva vs taxa de perda configurada, em diferentes tamanhos de grupo (grupo maior = menos overhead de banda, mas mais vulnerável a perda dupla no mesmo grupo)

## Bonus (se sobrar tempo):

Comparar overhead de banda real (% de pacotes extra enviados) entre FEC e o custo de retransmit do challenge 08, no mesmo cenário de perda qual estratégia "paga mais" em cada regime de perda.

**O que seria observado**:

se você entende o limite estrutural do XOR simples (recupera 1 perda por grupo, não mais) e desenha o tamanho de grupo como um trade-off explícito (banda vs robustez), não como número arbitrário.

---
