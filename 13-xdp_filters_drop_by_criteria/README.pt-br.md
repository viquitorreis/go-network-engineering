# XDP - FILTRA/DROPA POR CRITÉRIO

**Categoria**: Network programming / Kernel bypass
**Tempo**: 2h
**Builda em cima de**: challenge 12 (XDP conta pacotes UDP por porta), mesma toolchain, mesmo .c, mesma estrutura de loader Go

## O que muda de verdade em relação ao challenge 12

Estruturalmente, pouca coisa muda e isso é intencional, o roadmap que fiz chama isso de "fechar a fase" porque é a mesma base, só virando a chave de observar pra agir. Duas mudanças centrais:

1. **XDP_PASS** deixa de ser a única resposta possível. Hoje o programa do challenge 12, sempre retornava XDP_PASS (só contava, nunca interferia). Agora, baseado num critério (porta bloqueada, por exemplo), ele retorna **XDP_DROP** o pacote é descartado ali mesmo, no driver, antes de custar qualquer processamento a mais no kernel. Isso é literalmente o mecanismo por trás de mitigação de DDoS em produção.

2. A política de bloqueio vem de um segundo eBPF map, escrito pelo Go não hardcoded no C. Isso é a peça nova mais importante do dia: até agora, o Go anterior só lia um map (port_count). Hoje ele também escreve num map (a lista de portas bloqueadas), e o C lê esse map pra decidir. Isso é comunicação nos dois sentidos entre userspace e kernel antes era só kernel -> userspace (contagem subindo), agora também é userspace -> kernel (política descendo).

## O que construir

1. Segundo map (`blocked_ports`, tipo `BPF_MAP_TYPE_HASH`, chave `__u16` porta, valor `__u8` só como flag de presença), se a porta está no map, está bloqueada
2. No `.c`: depois de extrair a porta de destino (já foi feito isso no challenge 12), faz `bpf_map_lookup_elem` nesse novo map se achar, return `XDP_DROP`; se não achar, segue pro fluxo de contagem normal e `XDP_PASS`
3. No Go: além de ler `port_count` periodicamente, adiciona um jeito de escrever no `blocked_ports` (pode ser via flag de linha de comando, ex: ./counter lo -block 9999, ou lendo de um arquivo de config simples) isso usa `objs.BlockedPorts.Put(port, value)`, o inverso do Iterate() que já foi usado

## Requisitos obrigatórios

- Pacotes de portas bloqueadas são de fato descartados testável comparando `curl`/`nc` numa porta bloqueada (deve falhar/não responder) vs numa porta livre (deve funcionar normal)
- A lista de bloqueio é configurável em runtime, sem precisar recompilar o .c nem reiniciar o programa Go atualiza o map, efeito imediato no próximo pacote
- Contador do challenge 12 continua funcionando só pra pacotes que passaram (não conta o que foi dropado), decisão de design que vale documentar: você quer saber "quanto tráfego bom passou", ou também "quanto tráfego foi rejeitado"? (isso muda se você quer 1 ou 2 contadores)
- Relatório final mostra separadamente: pacotes permitidos por porta, pacotes bloqueados por porta

## Bonus, se sobrar tempo

- Critério mais rico que porta fixa: bloquear por faixa de IP de origem (CIDR), reaproveitando a mesma ideia de segundo map, só que a chave vira um prefixo de IP em vez de porta
- Rate limiting simples: em vez de bloqueio binário, permite N pacotes por porta por segundo, usando timestamp dentro do map como estado
O que será observado

Se você entende a diferença entre "decisão hardcoded no C, precisa recompilar pra mudar" (o que você faria se tivesse feito if (port == 9999) return XDP_DROP direto no código) vs "decisão orientada a dado, o C só consulta um map que o Go controla" (o que o desafio pede) essa segunda abordagem é o padrão real usado em produção (é literalmente como o Cilium/Katran fazem: o plano de controle em userspace decide política, o plano de dados em kernel só executa).

---

Referencia de docs:

`github.com/RaniaMidaoui/ebpf-pingkiller`: dropa pacotes ICMP especificamente, com um "KSP" (kernel space program, o .c) e um "USP" (user space program, o Go usando Cilium) bem separados conceitualmente, do jeito que a gente também estruturou. Vale ler a wiki do repo pra ver como eles dividiram as responsabilidades.

Pra aprofundar o conceito de "XDP como camada de filtro" no nível de produção real, o BPF and XDP Reference Guide da Cilium (`docs.cilium.io/en/latest/reference-guides/bpf/`) tem a seção de hooks explicando por que XDP é o ponto certo pra esse tipo de decisão (comparado a rodar o mesmo filtro em `tc`, que roda depois, com mais overhead).