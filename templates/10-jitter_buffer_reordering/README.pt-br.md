# JITTER BUFFER REORDENAÇÃO

**Categoria**: Streaming / Network programming raw
**Tempo**: 2h (15min teoria + ~1h45 challenge)
**Builda em cima de**: challenge 26 (RTP-style packetizer com seq + timestamp)

## Estudo (15 min):

Foco disso: **jitter** é a variação no tempo de chegada dos pacotes, mesmo que o emissor mande em intervalos perfeitamente regulates (20ms, 20ms, 20ms...), a rede NÃO entrega nessa regularidade, pode tomar caminhos diferentes e chegar em tempo diferente no destino. Um pacote pode chegar em 18ms, outro em 35, outro em 15, etc... a rede "engasta e acelera" (congestionamento e descongestionamento, entre outras questões). Se o receptor tocasse cada pacote **assim que chegasse**, o áudio / vídeo, ficaria robótico e com cortes, porque o timing de reprodução ficaria refém da variação da rede, não do timing original de captura.

Solução: **jitter buffer**

Assim que tocar um pacote assim que ele chega, você **segura ele por um tempo fixo** (o "buffer delay", tipicamente de 20 - 200ms dependendo da aplicação) antes de liberar para reprodução. Isso cria uma margem de segurança se um pacote chegar um pouco atrasado, ainda há tempo de reordená-lo antes do momento em que ele **prtecisaria** ser tocado. O trade-off é: **buffer maior = mais tolerância a jitter, mas mais  latência total** (você sempre está "atrasando"  a reprodução antes de tocar, de propósito). É por isso que chamadas de voz em tempo real usam buffers pequenos (20-60ms, latência importa mais que suavidade perfeita) enquanto streaming de vídeo gravado usa buffers de segundos (suavidade importa mais que latência).

*Revisar*: o `Timestamp` que já tem no pacote RTP-style é o que permite calcular "quando esse pacote **deveria ser tocado**, o jitter buffer usa isso para decidir a ordem de liberação do play, não a ordem de chegada na rede.

## Contexto:

Já temos o receptor detectando gaps e pacotes fora de ordem (challenge 26), mas hoje ele só **loga** isso, não faz nada de útil com a informação, mas isso vai acabar ajudando. Agora deve construir o **componente que reordena o que chega**, liberando pacotes numa fila de reprodução na ordem certa (por seq/timestamp), mesmo que a rede entregue fora de ordem, com um limite de quanto tempo esperar antes de desistir de um pacote atrasado demais.

## O que construir:

1. **Buffer com capacidade / janela de tempo configurável**: uma estrutura que recebe pacotes chegando (possivelmente fora de ordem), e segura eles por um tempo antes de liderar.

2. **Lógica de liberação por timestamp, não por chegada**: o buffer precisa saber "qual é o próximo pacote que deveria tocar" baseado no `Timestamp/SeqNumber` esperado, não em quem chegou primeiro na rede

3. **Timeout de espera por pacote atrasado**: se um pacote esperado não chegar dentro da janela do buffer, o player não pode travar esperando para sempre, depois de um prazo, ele **pula** aquele pacote (marca que foi perdido) e libera o próximo que já tem disponível.

4. **Integrar com o receptor do challenge 26**: em vez de só logar gap/fora-de-ordem, o receptor agora alimenta esse buffer, e um "player" simulado (pode ser só um `log.Println` indicando "toca frame N") consome do buffer na ordem correta

## Requisitos obrigatórios

- Buffer com janela de tempo configurável (ex: 100ms), pacotes só são liberados depois de terem "esperado" esse tempo desde a captura original (baseado no `Timestamp`)
- Reordenação: se pacotes 5,7,6 chegarem nessa ordem nada rede, o buffer libera 5,6,7 (ordem correta), não a ordem de chegada
- Timeout por pacote: se o seq esperadonão aparecer dentro do prazo da janela, marca como perdido e segue em frente sem travar o player
- Relatório final: quantos pacotes foram reordenados com sucesso, quantos foram perdidos de fato (nunca chegaram dentro da janela), latencia efetiva de reprodução (tempo de cqaptura e "reprodução")

## Bonus (se sobrar tempo):

- Buffer adaptativo: em vez de janela fixa, ajusta o tamanho da janela dinamicamente baseado no jitter observado recentemente (se a rede está estável, encolhe a janela pra reduzir latência; se está instável, aumenta pra tolerar mais)
- Métrica de jitter de verdade, no formato que RTCP usa (RFC 3550): média móvel da diferença de intervalos de chegada consecutivos

**O que seria observado**:

Como você estrutura a fila de espera (heap por timestamp? lista ordenada? outra coisa?) de forma que inserir um pacote fora de ordem e descobrir "qual é o próximo a liberar" sejam os dois eficientes e como você decide de forma limpa, quando desistir de esperar um pacote sem travar o player indefinidamente.

---

Primeiro passo: sem código: pensa na estrutura de dados já foi feito (dos challenges anteriores), que já deixa os elementos ordenados e dá o "menor" de forma eficiente, para não ter que ficar escaneando.

## Skip list, por quê é ideal aqui

É um pouco diferente do order book, la era usado uma skip list ordenada por **preço**, e o valor "score" nunca mudava depois de inserido (uma ordem de R$100 continua sendo R$100 até ser cancelada / executada). Aqui a chave se ordenação é o `timestamp / seq number` de cada pacote, mesma mecânica extra: `Insert(seq, pacote)` te dá O(log n) de inserção **mantendo ordem**, e `Front()` sempre te dá "o próximo a liberar", em O(1), sem escanear nada.

Diferença de uso importante: no order book raramente removiamos do meio sem ser via `Cancel` explícito. Aqui, o padrão de acesso é mais parecido com uma fila de prioridade que estamos constantemente **drenando pela frente** (`Front()` + remove) conforme o tempo passa e pacotes "vencem" a janela de espera, e consequentemente **inserindo fora de ordem** no meio (quando um pacote atrasado finalmente chega). Isso é exatamente o padrão que skip list resolve bem: `Front()` barato, `Insert` em qualquer posição O(log n), sem o problema de remoção aleatoria do heap.

A skip list resolve um problema: qual o próximo da ordem. Mas ainda precisa de uma segunda lógica que vai **verificador de prazo**, que fica perguntando periodicamente "o pacote na frente da fila já esperou tempo demais?". Basicamente um ticker rodando em paralelo chegando `Front()` da skip list e comparando com o relógio, e decide se libera (ordem toda) ou desiste (por timeout) do que tá na frente.