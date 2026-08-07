# AF_XDP SOCKET: PRIMEIRO ATTACH
**Categoria**: Networking / Kernel Bypass
**Tempo**: 3h
**Builda em cima de**: 13-xdp_filters_drop_by_criteria (mesmo XDP program hook, agora redirecionando pra um socket em vez de contar/dropar)

## Estudo antes (10-15min)
A diferença entre XDP puro (roda dentro do kernel, decide DROP/PASS/etc **antes do pacote entrar na stack de rede**) e AF_XDP (socket especial que recebe pacotes crus direto do driver da NIC, via um buffer compartilhado entre kernel e userspace chamado **UMEM**, sem passar pela stack de rede tradicional). AF_XDP ainda precisa de um XDP program anexado pra decidir *quais* pacotes redirecionar pro socket (via XDP_REDIRECT) — os dois trabalham juntos, XDP program decide, AF_XDP entrega. Hoje é só montar o socket básico e o attach, sem redirect ainda (isso é o próximo challenge).

## Contexto
Os challenges 12 e 13 processaram pacote inteiramente dentro do kernel (contar, dropar). Isso é rápido mas limitado — não dá pra inspecionar payload complexo nem processar em userspace sem pagar custo de cópia pela stack normal. AF_XDP existe pra isso: throughput de kernel bypass, com a flexibilidade de processar em Go no userspace.

## O que construir
1. UMEM: região de memória compartilhada registrada com o kernel, dividida em frames de tamanho fixo, mais os 4 rings de controle (fill, completion, rx, tx)
2. Socket AF_XDP criado e vinculado (bind) a uma interface de rede + fila (queue id) específica
3. Attach do socket ao UMEM via **setsockopt**
4. Loop básico: popula o fill ring com frames disponíveis, faz poll no socket, lê o rx ring quando pacote chega, imprime tamanho e primeiros bytes do pacote recebido

O ring responsável por dizer ao kernel "aqui está um frame livre pra você preencher" é o **fill ring**. Você "enche" o ring buffer com endereços de buffer vazios, o kernel consome desse ring, escreve o pacote recebido lá dentro, e devolve a posição preenchida via **rx ring** (esse sim é o que lemos pra pegar o pacote). O completion e o tx ring são o par simétrico do lado de *enviar* (sem uso hoje, foco é só receber).

**Duas diferenças em relação aos challenges 12/13**:
- Sem parsing de Ethernet/IP/UDP aqui. No challenge 12 era preciso inspecionar o pacote pra decidir contar por porta. Aqui não — a decisão é só "esse queue id tem socket registrado?", então nem precisa tocar em `ctx->data`/`data_end`. Isso também significa: redireciona todo tipo de pacote que chegar nessa queue, não só UDP.
- Sem `qidconf_map` separado (que o `asavie/xdp` usa no repo dele) — um único `xsks_map` já resolve, porque a própria presença de uma entrada na queue já significa "tem socket aqui, redireciona".

## Requisitos obrigatórios
- Foco só no attach e no primeiro pacote chegando pelo socket, sem lógica de redirect condicional via XDP program ainda (testa com tráfego que já bateria no host de qualquer forma, tipo loopback via veth pair, se não tiver interface física disponível pra testar com segurança)
- Documentar os 4 rings (fill, completion, rx, tx) e o papel de cada um — isso é a base conceitual pro challenge de zero-copy que vem depois
- Vai precisar rodar como root ou com capability CAP_NET_RAW/CAP_BPF — documenta isso no README, já que é diferente dos challenges anteriores

**Bonus (se sobrar tempo)**
- Medir e documentar quantos pacotes por segundo você consegue ler do rx ring num teste local simples, como baseline pra comparar com o benchmark de zero-copy de um challenge futuro

O que será observado: se os 4 rings foram entendidos na função de cada um (não só copiados de um exemplo), e se o socket realmente recebe pacote cru, não simulado

**Observações importantes**
- O **XDP program** (o que roda dentro do kernel, decidindo o redirect) precisa ser escrito em C/C++ (ou Rust, etc.) e compilado pra bytecode eBPF via `clang`/LLVM. Não dá pra escrever isso em Go puro, já que a linguagem precisa de um SO pra rodar.
- O **socket AF_XDP** em si (userspace, lendo do rx ring) é Go puro, usando uma lib como `github.com/cilium/ebpf` ou `github.com/asavie/xdp` — não precisa de C pra essa parte.
- Ou seja: precisa de pouco C pro programa XDP que faz o redirect, mas a maior parte do trabalho (setup do socket, UMEM, rings, processamento de pacote) é Go — a parte de C segue o mesmo padrão já usado nos challenges 12/13.

## Referências
- Lib Go pura que encapsula socket AF_XDP, UMEM e os 4 rings. Tem `examples/` com `sendudp.go` e `senddnsqueries.go` prontos pra rodar e modificar:
  https://github.com/asavie/xdp
- Explica os 4 rings, copy mode vs zero-copy, e por que precisa de um programa XDP mínimo pro redirect:
  https://github.com/xdp-project/xdp-tutorial/blob/main/advanced03-AF_XDP/README.org
- Exemplo XDP + Go + bpf2go (sem C manual além do próprio `.c`, compilado via `go generate`, o resto é Go):
  https://github.com/cilium/ebpf/blob/main/examples/xdp/main.go
- Doc oficial do kernel sobre AF_XDP, com referência sobre os 4 rings (fill, completion, rx, tx) e o fluxo `XSKMAP + XDP_REDIRECT`:
  https://docs.kernel.org/networking/af_xdp.html

## O que foi construído de fato
Em vez do programa fixo do `asavie/xdp`, o programa XDP aqui foi escrito em C e compilado via `bpf2go` (mesmo padrão dos challenges 12/13), dando controle total sobre a lógica de redirect em vez de depender de bytecode hardcoded. O socket AF_XDP (registro de UMEM, setup dos rings, loop de fill/receive) foi escrito à mão em Go usando syscall cru (`setsockopt`/`getsockopt`/`mmap`/`bind`) contra os tipos de `golang.org/x/sys/unix`, em vez de depender de um wrapper de socket de terceiros — mais perto de como isso seria feito num sistema de produção que precisa de lógica de filtro customizada.

---
Primeiro passo: antes de qualquer código de socket, qual dos 4 rings (fill, completion, rx, tx) é responsável por dizer ao kernel "aqui está um frame de buffer livre pra você preencher com o próximo pacote que chegar"? Pensa nisso primeiro — é a peça que normalmente confunde na primeira vez com AF_XDP.