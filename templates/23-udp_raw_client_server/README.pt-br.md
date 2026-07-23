# UDP RAW CLIENT/SERVER - COM PERDA SIMULADA

- Categoria: Network programming raw
- Tempo: 2h (15min teoria + ~1h45 challenge)

**Estudo (15m)**:

Foco específico: porque UDP não tem `Accept()` / `Dial()` no sentido de conexão. Em UDP, o servidor abre **um único socket** com **net.ListenUDP** e recebe datagramas de qualquer client nesse mesmo socket, não existe "uma goroutine por conexão", como no TCP, porque **não existe uma conexão, sessão, nem handshake**, ao contrário do TCP. Cada `ReadFromUDP` te devolve os bytes e o endereço (`*net.UDPAddr`) de quem mandou, é assim que sabe quem está falando, pacote a pacote.

Client: `net.DialUDP`, `WriteToUDP`, `ReadFromUDP`
Server: `net.ListenUDP`

Quando usamos `DialUDP` no client, cria uma "pseudo conexão", só do lado do client. Isso é como a arquitetura do Hub/Broker, bão tem mais "register de conexão", o estado de "quem é cliente", precisa ser rastreado por endereço (`UDPAddr`), e não pelo `net.Conn`.

## Contexto

Você está prototipando a camada de transporte de um sistema de telemetria (ex: sensores IoT, ou métricas de jogo em tempo real) onde throughput e latência importam mais que garantia de entrega por isso a escolha de UDP em vez de TCP. Mas você precisa entender e medir exatamente o que se perde ao abrir mão de TCP, antes de decidir se e como adicionar confiabilidade por cima (que faremos em outro challenge).

**O que construir**:

1. Um servidor UDP que escuta uma porta, recebe datagrams de múltiplos clients (identificados por endereço), e ecoa de volta (ou proecssa e responde) cada um
2. Um client UDP que manda uma sequencia de mensagens (ex MSG 1, MSG 2, ... MSG 100)
3. Um **simulador de perda de pacote** no lado do servidor, artificialmente descarta uma % dos datagramas recebidos.
4. O client precisa **detectar e reportar** quais números de mensagem nunca receberam resposta (ele sabe que mandou 100 por exemplo, então no fim deve comprar o que recebeu de volta vs o que esperava)

**Requisitos obrigatórios:**

- servidor UDP puro (`net.ListenUDP`), sem nenhuma lib de reliability
- Client manda N mensagens numeradas em sequencia rápida (sem esperar ACK, UDP puro é fire and forget)
- Server com % de perdas simuladas
- Client relata ao final: quantas mensagens mandou, quantas respostas recebeu, quais numeros faltaram
- Tratar o caso de mensagens chegarem **fora de ordem** no client (UDP não garante ordem, não assume que a resposta de MSG 5 chega antes da de MSG 8)

**Bonus**

- Medir round trip por mensagem (timestamp no envio e calcula delta ao receber)
- Simula Não só perda mas tambem duplicacao de pacote (processar o mesmo datagram 2x de forma proposital)

**O que será observado:**

Como você estrutura o estado de "quem enviou o quê" sem ter uma net.Conn por client e se seu client lida corretamente com o fato de que, em UDP, "não recebi resposta" pode significar 3 coisas diferentes (pacote de request perdido, pacote de response perdido, ou processamento ainda em andamento) e você não consegue distinguir essas 3 só olhando o timeout.