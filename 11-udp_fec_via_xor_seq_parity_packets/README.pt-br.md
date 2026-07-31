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

## Benchmarks

Machine:

```
goos: linux
goarch: amd64
pkg: feq_parity_packets/receptor
cpu: 12th Gen Intel(R) Core(TM) i5-1235U
```

| Group | Loss | Overhead | Definitive Loss | Recovered |
|---|---|---|---|---|
| 4 | 5% | 25.00% | 3.00% | 14.67% |
| 4 | 10% | 25.00% | 6.67% | 30.33% |
| 4 | 20% | 25.00% | 21.33% | 40.33% |
| 8 | 5% | 12.50% | 9.33% | 28.67% |
| 8 | 10% | 12.50% | 27.00% | 31.67% |
| 8 | 20% | 12.50% | 57.00% | 28.67% |
| 16 | 5% | 6.25% | 26.33% | 33.67% |
| 16 | 10% | 6.25% | 57.67% | 26.00% |
| 16 | 20% | 6.25% | 90.00% | 7.33% |

### A intuição do porquê "recovered %" tem um pico

`Recovered` só acontece no caso muito específico de **exatamente um sumir**. Com um grupo pequeno (poucos pacotes enviados), a chance de exatamente 1 sumir é baixa simplesmente porque tem poucos membros envolvidos, mas conforme o grupo cresce, tem mais **chances** de exatamente uma pessoa sumir (mais "tentativas"), então a taxa de recuperação sobe. Só que ao mesmo tempo, o grupo maior também aumenta a chance de **duas ou mais** sumirem, esse efeito cresce mais rápido. Em algum ponto, a chance de 2+ ultrapassa a chance de "exatamente 1", e a partir dai crescer o grupo só piora as coisas: mais perda definitiva, menos recuperação.

Podemos ver isso nos números: 20% de perda, `recovered` sobe de 40,33% (grupo 4) para 28,67% (grupo 8) e desaba para 7,33% (grupo 16), o pico já passou entre 4 e 8, e depois disso só piora.

### Não existe apenas uma direção para trade-off

O trade-off, depende de vários fatores, como a taxa de perda esperada da rede.

- **Rede boa (5% de perda):** grupo 16 já perde 26,33% definitivamente, isso é surpreendentemente ruim mesmo numa rede "boa", pois o denominador (16 pacotes) é grande demais. o grupo 4 continua com só 3% de perda definitiva, e paga mais overhead (25%) que o grupo 16 (6,25%). Então mesmo em rede boa, o grupo 4 é estritamente mais seguro, mas é mais caro em banda (esperado pelo approach feito).

- **Rede ruim (20% de perda):** a diferença fica gritante, o grupo 4 perde 21,33% definitivamente, o grupo 16 perde 90%. Nessa condição grupo grande é quase inútil.

### Conclusão

Grupo pequeno é sempre mais robusto. Grupo grande é sempre mais econômico em banda. Não tem um grupo que vai vencer nos dois lados no approach escolhido ao mesmo tempo. A escolha certa depende de qual recurso é mais escasso no cenário real (banda ou tolerância a perda), e esse benchmark é a ferramenta para decidir isso.

### Possíveis Melhorias

- **Reed-Solomon em vez de XOR com paridade única.** XOR é, na prática, o caso especial de Reed-Solomon com só 1 pacote de paridade, e é por isso que ele só recupera exatamente 1 perda por grupo. Usar `k` pacotes de dados + `m` pacotes de paridade (Reed-Solomon) toleraria até `m` perdas por grupo, ao custo de aritmética de Galois Field, que é bem mais cara que um XOR simples.
- **Interleaving.** Esse benchmark assume perda de pacote independente e aleatória. Perda de rede real costuma vir em rajada (um roteador congestionado derruba vários pacotes consecutivos de uma vez). Se um grupo FEC é formado por pacotes consecutivos, uma única rajada pode derrubar mais de um pacote do mesmo grupo. Interleaving espalha pacotes consecutivos entre grupos diferentes, então uma rajada atinge vários grupos levemente em vez de um grupo só pesadamente.