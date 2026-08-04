# XDP - PROGRAMA CONTA PACOTES POR PORTA

**Categoria**: Network programming / Kernel bypass
**Tempo**: 2h (30min teoria + ~1h30 challenge hoje a teoria pesa mais que o normal, é conteúdo genuinamente novo)
**Builda em cima de**: conceitualmente nada dos challenges anteriores (é uma camada abaixo de tudo que você fez até agora) mas segunda-feira você já devia ter lido sobre o caminho do pacote no kernel + feito setup da toolchain, então parte disso já é familiar

## Estudo antes

### O que é eBPF

Bem, primeiro vale ler sobre o que é eBPF: https://ebpf.io/what-is-ebpf/

Resumindo eBPF em uma frase: é uma forma de **rodar programas dentro do kernel** de uma forma segura, sem precisar recompilar o kernel ou escrever um módulo de kernel tradicional (que é perigoso, bug ali pode derrubar a máquina inteira). O kernel roda um **verifier** que analisa seu programa antes de aceitar carregar ele, rejeita loops infinitos, acesso de memória fora de bounds, qualquer coisa que pode travar ou crashar o sistema. Isso é o que torna eBPF seguro o suficiente pra rodar código de terceiros dentro do kernel.

### O que é XDP especificamente

O XDP é um dos pontos de anexação possíveis para um programa eBPF especificamente, o ponto **mais cedo possível** no caminho de um pacote de rede: **antes** do kernel sequer alocar a estrutura `sk_buff` (a representação interna de um pacote que o kernel usa em todo o resto da pilha de rede). O XDP roda literalmente no **driver da placa de rede**, no momento em que o pacote acabou de chegar fisicamente.

**Por que isso importa pra performance**: processar um pacote em XDP custa **ordens de magnitude a menos** que deixar ele subir pela pilha de rede normal do kernel (que aloca `sk_buff`, passa por netlifer/iptables, roteamento, etc. Tudo isso **antes** de qualquer aplicação em userspace sequer saber que o pacote existe). É por isso que XDP é usado em produção para mitigação de DDoS, load balancing (o Facebook/Meta usa isso no Katran), e qualquer cenário que precisa decidir "processar, dropar ou redirecionar" um volume grande de pacotes com o mínimo de overhead possível.

O que precisa ser entendido e porque é diferente de tudo que foi feito até então: o programa que roda **dentro do kernel** é escrito em um subset restrito de C, compilado pelo `clang`/LLVM para bytecode eBPF não é C completo, sem heap, sem chamada de função arbitrária, loops precisam ter limite provável estaticamente. O programa que roda em **userspace** (que carrega esse bytecode no kernel, anexa ele numa interface de rede, e lê os resultados) é Go de verdade, usando a lib do cilium ebpf `github.com/cilium/ebpf`. Não vamos escrever Rust nem C++ pra isso, só um pedacinho pequeno de C restrito, e o resto continua sendo o Go.

**Como kernel e userspace conversam, a peça que fecha o quebra-cabeça**: eBPF **maps** são estruturas de dados (essencialmente hash maps ou arrays) que tanto o programa C no kernel quando o programa Go em userspace conseguem acessar. É assim que vai montar os pacotes: o programa C, rodando para cada pacote que chega, incrementa um contador dentro de um map, indexado pela porta de destino. O programa Go, rodando normalmente em userspace, **lê** esse mesmo map periodicamente sem nenhuma comunicação de rede entre os dois, é memória compartilhada gerenciado pelo kernel.

## Contexto

Você está construindo a peça mais básica de qualquer sistema de observabilidade/mitigação de rede de alta performance: contar quantos pacotes chegam por porta de destino, sem que esse processo de contagem em si vire um gargalo. Isso é literalmente o primeiro passo antes de qualquer coisa mais sofisticada (filtrar, dropar, redirecionar) que você vamos construir nos próximos dias desta semana.

## O que construir

1. **Programa XDP em C**: para cada pacote recebido, faz o parsing manual do cabeçalho Ethernet -> IP -> TCP/UDP (sem lib nenhuma é parsing de bytes cru, igual o que já fizemos com framing binário, só que agora em C e em cima de um formato de pacote real de rede) para extrair a porta de destino.
2. **Um eBPF map** (tipo `BPF_MAP_TYPE_HASH` ou `BPF_MAP_TYPE_ARRAY`) indexado por porta, guardando a contagem de pacotes vistos.
3. **Loader em Go** usando `cilium/ebpf` que compila/carrega o bytecode, anexa o programa numa interface de rede (pode ser a `lo`, loopback para testar local sem precisar de tráfego real de rede) e periodicamente lê o mapa e imprime as contagens por porta.

## Requisitos obrigatórios

- Programa XDP compila e carrega sem erro do verifier (isso sozinho já é vitória hoje o verifier rejeita muita coisa que pareceria válido em C normal)
- Contagem correta por porta, testável mandando tráfego real (ex: `curl` `localhost:PORTA` de outro terminal enquanto o XDP está anexado em `lo`)
- Loader Go limpo: carrega, anexa, mas também **desanexa** ao encerrar (Ctrl+C), deixar um programa XDP "grudado" numa interface depois que seu processo Go morreu é um problema real de operação, não só falta de acabamento
- Decisão do programa XDP hoje é sempre `XDP_PASS` (deixa o pacote seguir normalmente pro resto da pilha), hoje é só **contar os pacotes**, não interferir. Isso depois vamos fazer.

## Bonus (se sobrar tempo)

- Separar contagem por protocolo também (TCP vs UDP), não só porta
- Expor as métricas via um endpoint HTTP simples em vez de só print no terminal

## O que será observado

Se você entende a diferença entre onde o código C roda (dentro do kernel, restrito, sem exceções, sem heap) e onde o código Go roda (userspace normal) e se a comunicação entre os dois acontece só via eBPF map, sem você inventar nenhum outro canal.

---

Primeiro passo, antes de escrever qualquer C: confirma tem a toolchain de clang, llvm, headers do kernel (linux-headers da sua distro), e a lib cilium/ebpf já baixada. Roda clang -target bpf -c teste.c -o teste.o num arquivo C vazio só pra confirmar que o compilador consegue gerar bytecode BPF antes de escrever lógica de verdade. Se isso já funcionar, você está pronto pra começar o parsing do pacote se não, esse é o primeiro obstáculo a resolver antes de tudo.

Existe exemplos de código assim no repo do eBPF: `github.com/cilium/ebpf/tree/main/examples/xdp`

Em `ebpf-go.dev/guides/getting-started/` tem um walkthrough anotado, clicável, explicando linha por linha o que SEC("xdp"), SEC(".maps"), e o resto do C significam. Esse é o melhor lugar pra realmente entender a mecânica, não só copiar.