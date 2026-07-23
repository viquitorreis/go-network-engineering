# RTP-STYLE PACKETIZER - SEQ + TIMESTAMP

**Categoria**: Streaming / Network programming raw
**Tempo**: 2h (15min teoria + ~1h45 challenge)
**Builda em cima de**: challenge 23/24 (UDP raw + reliable), mas atenção nesse: esse challenge não usa reliability. É uma mudança de modelo mental importante, ver abaixo.

## Estudo antes (15 min):

Foco central: **RTP (real-time transport protocol) não é confiável, e isso é proposital**. No challenge de reliable UDP, perder uma mensagem era inaceitável, você retransmitia até garantir a entrega. Só que em streaming de mídia em tempo real (áudio, vídeo, game state), o cenário é oposto: se um pacote de áudio se perder, **retransmitir ele não ajuda em nada**, quando a retransmissão chegasse, o momento de tocar aquele áudio já passou.. É melhor perder o pacote e seguir em frente do que travar esperando algo que não serve mais quando chegar a trasado. Isso é a diferençá fundamental entre **reliability** (garantir que tudo chega, não importa quando) e **timeliness** (garantir que as coisas cheguem a tempo, aceitando que algumas não cheguem).

RTP (Real time trasport protocol, usado por WebRTPC, VOIP, streaming ao vivo) é desenhado em cima dessa filosofia: cada pacote carrega um **sequence number** (para detectar perda e reordenar o que chegou fora de ordem) e um **timestamp** (pra saber quando aquele dado deveria ser "tocado", e não só quando ele chegou).

Revisa isso mentalmente: um pacote de áudio de 20ms captado no timestamp X precisa ser tocado 20ms depois do pacote anterior, mesmo que ele tenha chegado na rede fora de ordem ou com atraso variável (jitter), o timestamp é o que permite ao receptor saber "isso deveria tocar aqui", independente da ordem de chegada na rede.

## Contexto

Você está construindo a camada de transporte de um sistema de streaming de áudio simples (pense numa chamada de voz simplificada)

- O emissor captura "frames" de áudio periodicamente e envia via UDP puro.
- O receptor precisa reconstruir a sequência correta para tocar os frames na ordem certa, mesmo que a rede entregue fora de ordem ou com perda.

Isso é diferente do challenge de reliable UDP, aqui **não pode haver retransmissão**, o objetivo não é garantir que tudo chegue, é dar ao receptor a **informação necessária** (seq + timestamp) pra ele decidir o que fazer com o que chegou.

**O que construir**:

1. **Formato de pacote estilo RTP simplificado**:

- Cada datagrama carrega no mínimo `SequenceNumber` (uint16 ou uint32, incrementa a cada pacote enviado, independente de perda)
- `Timestamp` (quando aquele frame foi "capturado", em unidades consistentes, pode ser um contador incremental simulando amostras de áudio, não precisa ser wall-clock real)
- `Payload` (os dados em si, pode ser simulado, tipo um número identificando qual "frame" é, não precisa ser áudio de verdade)

2. **Emissor (sender)**:

- Gera frames em intervalos regulares (ex: a cada 20ms, simulando captura de áudio)
- Manda cada um via UDP puro, sem esperar confirmação nenhuma, puro fire-and-forget, diferente do client reliable que foi feito.

3. **Receptor (receiver)**:

- Recebe os pacotes conforme chegam (que pode ser fora de ordem, já que UDP não garante ordem), e precisa **detectar** duas coisas a partir do `SequenceNumber`: pacotes que chegaram fora de ordem (seq menor que o último já processado) e pacotes que nunca chegaram (gaps na sequência)

4. **Relatório de sessão**: 

Ao final (ou periodicamente), o receptor reporta: quantos pacotes recebeu, quantos gaps detectou (seq numbers que nunca chegaram), quando chegaram fora de ordem, e a variação de timing entre chegada e o que o timestamp esperava isso é uma prévia do conceito de **jitter**, que vamos ver melhor posteriormente.

## Requisitos obrigatórios

- Pacote `SequenceNumber`, `Timestamp` e o `Payload`, formato binário simples (pode reaproveitar o padrão de framing que já foi feito antes, ou simplificar para esse caso já que não precisa de comando tipo MSG/ACK).
- Sender manda em intervalos regulares (ticker), sem esperar resposta nenhuma
- Receiver detecta gaps na sequência (seq numbers pulados), não tenta recuperar, só detecta e reporta
- Receiver detecta pacotes fora de ordem (chegou um seq menor que o maior já visto)
- Nenhuma retransmissão, nenhum ACK, isso é intencional, não esquece de omitir
- Relatório final com: total enviado (conhecido pelo sender), total recebido, taxa de perda observada, quantos fora de ordem

## Bonus (se sobrar tempo)

- Simula perda de pacote no meio do caminho (reaproveita a lógica de `rand.Float64() < lossRate` que já foi feita antes), pra ver o gap detection funcionando de verdade, não só na teoria
- Calcula uma métrica simples de jitter: para cada pacote, compara o intervalo esperado entre timestamps consecutivos com o invervalo real de chegada, isso é o embrião do que vamos precisar para quarta feira.

**O que vai ser observado**

Como desenha detecção de gap/fora-de-ordem usando só `SequenceNumber` (sem nenhuma estrutura de estado pesada, isso deveria ser bem mais simples que o `pending map` que fizemos no realible UDP, já que aqui não tem retry esperando resposta), e se você resiste à tentação de  adicionar reliability por engano (esse é um erro comum: gente que acabou de implementar ACK/retry tende a querer reusar esse padrão aqui, mas isso contradiria o propósito do RTP)

---

Primeiro passo: pensa como o `SequenceNumber` sozinho já resolve o gap de detection de graça, se o receptor guarda só **um número** (o último seq visto), que operação simples nesse número já revela "quantos pacotes eu perdi entre o último e esse que acabou de chegar"? 