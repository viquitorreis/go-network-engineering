# TCP MULTIPLEXED STREAM BROKER

**Categoria**: Streaming + Concorrência (raw)
**Tempo**: 2h (15min teoria + ~1h45 challenge)

Resumo do que irá aprender nesse challenge:

- TCP raw com framing length-prefixed manual (sem lib pronta)
- Broker centralizado com goroutine única + channels (register/unregister/broadcast)
- Multiplexação por topic
- Read/write pumps separados por conexão
- Slow consumer handling ficou de bônus se quiser voltar depois

## Estudo antes (15 min)

Foco: **length-prefixed framing** sobre TCP.

**Length-prefixed framing**: É uma técnica que faz split/divisões de stream de bytes continuos em um mensagems distintas, de tamanho variável, ao prefixar cada body de mensagem com seu tamanho exato em bytes.

O TCP, usa stream de bytes sem noção da mensagem, então quando não temos uma lib pronta fazendo o framing, nós mesmos precisamos decidir onde uma mensagem começa e termina. O padrão mais comum é prefixar cada mensagem com N bytes indicando o tamanho do payload que vem a seguir (`binary.Write` com um uint32 de tamanho, depois os bytes).

**Revisar**: da lib `encoding/binary`, os métodos (`binary.BigEndian.PutUint32 / Uint32`) e como usar `io.ReadFull` para garantir que lemos exatamanete N bytes antes de tentar decodificar (um `conn.Read()` sozinho pode retornar menos bytes do que pedimos).  Pensa nisso como um Stratum Protocol, ou qualquer outro protocolo como uma nova linha de delimitador (\n), aqui vai ser um binário de tamanho pré-fixado, que é só uma variação.

- Por quê **io.ReadFull()**?

`conn.Read(buf)` não garante que vai preencher o buffer inteiro em uma chamada. Numa rede real, um pacote TCP pode chegar fragmentado, se pediu 512 bytes, mas o SO só tinha 40 disponíveis no buffer do Kernel naquele instante e `Read` retorna só 40, sem erro. Se você tratar isso como "a mensagem inteira chegou", vai processar dado incompleto / lixo.

O `io.ReadFull(reader, buf)` resolve isso: ele fica chamando `Read` internamente em loop até o buffer `buf` estar **totalmente preenchido**, ou até dar erro/EOF de verdade. É exatamanete o que queremos para ler os "4 bytes do tamanho", ou os "N bytes do payload" com garantia de que veio tudo de fato.

## Contexto

Imagine que está construindo uma camada de transporte de um sistema de market data interno: múltiplos serviços internos (não browser, serviços que não precisam do overhead do HTTP) se conectam via TCP puro num broker central, se inscrevem em "topicos" (ex: "prices.BTC", "prices.ETC"), e o broker distribui mensagens publicadas num tópico para todos os subscribers daquele tópico, multiplesxando várias conversas lógicas numa arquitetura de conexões TCP diretas, sem overhead de HTTP/WebSocket.

## O que construir

Um servidor TCP que aceita conexões raw, onde cada conexão pode:

- Mandar um comando `SUB <topic>` pra se inscrever (subscribe)
- Mandar um comando `PUB <topic> <payload>` para publicar (publish)
- Receber, de forma assíncrona, qualquer mensagem publicada nos topics que assinou

Procolo simples que você define: cada mensagem é `[4 bytes tamanho][N bytes payload]`, onde o payload é uma linha de texto tipo `SUB prices.BTC` ou `PUB prices.BTC 67000.50`.

### Sobre o Frame

Protocolo de 2 etapas, sempre nessa ordem:

- **Etapa 1: ler o tamanho**. Já sabemos que os primeiros 4 bytes de qualquer mensagem são um inteiro (uint32) dizendo quantos bytes vêm a seguir. Fazemos `io.ReadFull` num buffer de exatamente 4 bytes. Sempre 4, fixo, pois é fixo por definição NESSE PROTOCOLO.
- **Decodificar o tamanho**: esses 4 bytes brutos viram um número usando `binary.BigEndian.Uint32(...)`. Agora sabemos, "a mensagem que vem tem exatamente N bytes".
- **Etapa 2: ler o payload**. Alocamos um buffer de exatamente N bytes (o tamanho que acabou de descobrir) e faz outro `io.ReadFull` nesse buffer. Agora você tem a mensagem completa, nem um byte a mais nem a menos.
- **Pra escrever (enviar) um frame**: o inverso, pega o payload, calcula `len(payload)`, converte esse número para 4 bytes usando `binary.BigEndian.Uint32(...)`, escreve esses 4 bytes na conexão, depois escreve o payload inteiro logo em seguida. Sempre nessa ordem: tamanho primeiro e dado depois.

Esse é o delimitador que resolve o problema de "TCP é só um stream de bytes sem noção da mensagem", cada leitura sabe exatamente quanto deve ler, pois a mensagem anterior te disse o tamanho da próxima etapa.

**Requisitos obrigatório**:

- Servidor TCP puro (`net.Listen("tcp", ...)`), NÃO use nenhuma lib de framing pronta
- Framing manual length-prefixed (você deve implementar `readFrame`/`writeFrame`)
- Broker central (mesmo padrão de hub, assim como no challenge do WebSocket) roteando por **topico**, goroutine única + channels
- Múltiplos subscribers por tópico, múltiplos tópicos simultâneos
- Cada conexão TCP com read loop e write loop separado (mesmo motivo do Websocket: não pode ter duas goroutines escrevendo no mesmo `net.Conn` ao mesmo tempo)
- Graceful shutdown: `context` + sinal do SO fechando todas as conexões de forma limpa
- Se um subscriber for lento (buffer cheio), broker não pode travar, desconecta ele (mesmo lógica de "slow consumer" presente em outros challenges)

**Bonus (se sobrar tempo)**:

- Suporte a wildcard no subscribe (`prices.*`)
- Métrica simples: quantas mensagens / segundo por tópico

**O que seria observado nesse desafio**

Como você separa a responsabilidade de "parsing do procolo" (framing + comando) da lógica de "roteamento" (broker), e como lida com conexão lenta sem travar o broadcast pros outros subscribers

---

Por onde começar: implementa `readFrame(conn net.Conn) ([]byte, error)` e `writeFrame(conn net.Conn, data []byte) error`, nada de broker primeiro, só confirma que cosnegue mandar e ler frames de tamanho pré determinados corretamente. Testa com `nc` mesmo não vai funcionar (é binário), então escreve um client Go mínimo também pra testar.