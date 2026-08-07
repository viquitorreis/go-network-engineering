# AF_XDP SOCKET: FIRST ATTACH

**Categoria**: Networking / Kernel Bypass
**Tempo**: 3h
**Builda em cima de**: 13-xdp_filters_drop_by_criteria (mesmo XDP program hook, agora redirecionando pra um socket em vez de contar/dropar)

## Estudo antes (10-15min)

A diferença entre XDP puro (roda dentro do kernel, decide DROP/PASS/etc **antes do pacote entrar na stack de rede**)
e AF_XDP (socket especial que recebe pacotes crus direto do driver da NIC, via um
buffer compartilhado entre kernel e userspace chamado **UMEM**, sem passar pela
stack de rede tradicional). AF_XDP ainda precisa de um XDP program anexado
pra decidir *quais* pacotes redirecionar pro socket (via XDP_REDIRECT), os
dois trabalham juntos, XDP program decide, AF_XDP entrega. Hoje estamos apenas montando
o socket básico e o attach, sem redirect ainda (isso é o próximo
challenge).

## Contexto

Os challenges 12 e 13 processaram pacote inteiramente dentro do
kernel (contar, dropar). Isso é rápido mas limitado, você não consegue
inspecionar payload complexo nem processar em userspace sem custo de cópia
pela stack normal. AF_XDP existe pra isso: throughput de kernel bypass, mas
com a flexibilidade de processar em Go no userspace.

## O que construir

1. UMEM: região de memória compartilhada registrada com o kernel, dividida
   em frames de tamanho fixo, mais os 4 rings de controle (fill, completion,
   rx, tx)
2. Socket AF_XDP criado e vinculado (bind) a uma interface de rede + fila
   (queue id) específica
3. Attach do socket ao UMEM via **setsockopt**
4. Loop básico: popula o fill ring com frames disponíveis, faz poll no
   socket, lê o rx ring quando pacote chega, imprime tamanho e primeiros
   bytes do pacote recebido

O ring responsável por dizer ao kernel "aqui está um frame livre pra você preencher" é o **fill ring**. Você "enche" o ring buffer com endereços de buffer vazios, o kernel consome desse ring, escreve o pacote recebido lá dentro, e devolve a posição preenchida via **rx ring** (esse sim é o que lemos para pegar o pacote). O completion e o rx ring são par simétrico do lado de *enviar* (sem uso agora, foco é só receber)

**Duas diferenças dos outros challenges 12/13**:

- Sem parsing de Ethernet/IP/UDP aqui. No challenge 12 que precisamos inspecionar o pacote pra decidir contar por porta. Aqui não, a decisão é só "esse queue id tem socket registrado?", então não precisa nem tocar em ctx->data/data_end. Isso também significa: redireciona todo tipo de pacote que chegar nessa queue, não só UDP.
- Sem qidconf_map separado (que o asavie/xdp usa no repo dele), um único xsks_map já resolve, porque a própria presença de uma entrada na queue já significa "tem socket aqui, redireciona".

## Requisitos obrigatórios

- Foco só no attach e no primeiro pacote chegando pelo socket sem
  redirect via XDP program ainda (pode testar com tráfego que já bateria
  no host de qualquer forma, tipo loopback com veth pair, se não tiver
  interface física disponível pra testar com segurança)
- Documentar os 4 rings (fill, completion, rx, tx) e o papel de cada um
  isso é a base conceitual pro challenge de zero-copy que vem depois
- Vai precisar rodar como root ou com capability CAP_NET_RAW/CAP_BPF
  documenta isso no README, já que é diferente dos challenges anteriores

**Bonus (se sobrar tempo)**

- Medir e documentar quantos pacotes por segundo
- Você consegue ler do rx ring num teste local simples, como baseline pra
comparar com o benchmark de zero-copy do challenge 

O que será observado: se os 4 rings foram entendidos na função de cada um
(não só copiados de um exemplo), e se o socket realmente recebe pacote
crus, não simulados

**Observações importantes**

- O **XDP program** (o que roda dentro do kernel, decidindo redirect) precisa ser escrito em C/C++ (ou rust etc) e compilado para bytecode eBPF via `clang/LLVM`. Não dá para escrever isso em Go puro pois a linguagem precisa de um OS para rodar.
- O **socket AF_XDP** em si (userspace, lendo do rx ring) é Go puro, usando uma lib como `github.com/cilium/ebpf` ou `github.com/asavie/xdp`, isso não precisa de C.
- Ou seja, precisa de pouco C pro programa XDP que faz o redirect, mas a maior parte do trabalho (setup do socket, UMEM, rings, processamento de pacote) é Go, essa parte de C foi feito nos challenges 12/13.

## Referencias

- Lib Go pura que usa um socket AF_XDP, UMEM e os 4 rings. Tem examples/ com sendudp.go e senddnsqueries.go prontos para rodar e modificar

https://github.com/asavie/xdp

- Explica os 4 rings, copy mode vs zero copy, e por que precisa de um programa XDP mínimo para o redirect:

https://github.com/xdp-project/xdp-tutorial/blob/main/advanced03-AF_XDP/README.org

- Exemplo XDP + Go + bpf2go (sem C manual). Tem um .c que pode ser compilado com `go generate`, resto é go

https://github.com/cilium/ebpf/blob/main/examples/xdp/main.go

- Doc oficial do kernel sobre AF_XDP com referência sobre os 4 rings (fill, completion, rx, tx) e o fluxo `KSKMAP + XDP_REDIRECT`

## O que fazer

1. Clona github.com/asavie/xdp, roda o example examples/sendudp (ou lê e
   entende o de recebimento, se tiver foco em receber pacote, não mandar)
2. Modifica pra rodar no seu ambiente: interface de loopback ou veth pair
   local, já que não precisa de tráfego real de produção pra esse
   challenge
3. Depois de rodar, escreve um resumo curto (README ou comentário) dos 4
   rings E de qual chamada da lib corresponde a qual ring (ex:
   "xsk.Fill(...) enche o fill ring, xsk.Receive(...) lê do rx ring")
4. Ajusta o exemplo pra imprimir tamanho + primeiros bytes de cada pacote
   recebido, em vez do que o exemplo original faz

---

Primeiro passo: antes de qualquer socket, qual dos 4 rings (fill,
completion, rx, tx) é responsável por dizer ao kernel "aqui está um frame
de buffer livre pra você preencher com o próximo pacote que chegar"? Pensa
nisso primeiro é a peça que normalmente confunde na primeira vez com
AF_XDP.