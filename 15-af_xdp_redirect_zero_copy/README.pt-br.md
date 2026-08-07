# CHALLENGE 15: XDP_REDIRECT + ZERO-COPY NEGOTIATION

**Categoria**: Networking / Kernel Bypass
**Tempo**: 1h
**Builda em cima de**: 14-af_xdp_socket

## Estudo antes (10-15min)

A flag de bind do socket AF_XDP determina o modo de operação:
`XDP_ZEROCOPY` pede ao driver pra escrever o pacote direto no
UMEM (sem cópia intermediária); XDP_COPY aceita que o kernel copie do
buffer interno da NIC pro UMEM (uma cópia extra, mas funciona em qualquer
driver). Sem nenhuma das duas flags, o kernel escolhe automaticamente
mas em produção você quer saber explicitamente qual modo está ativo, não
descobrir por acaso.

**Contexto**:

O challenge 14 fez bind sem especificar modo, então o kernel
escolheu por conta própria (se não tiver NIC vai ser copy, no meu caso sim
já que r8169 não suporta zero-copy). Hoje você torna essa escolha explícita e observável:
tenta zero-copy primeiro, cai pra copy automaticamente se o bind falhar,
documenta qual modo está rodando de fato.

O que construir:

1. Bind tentando `XDP_ZEROCOPY` primeiro
2. Se o bind retornar erro específico de modo não suportado, tenta de
   novo com XDP_COPY
3. Loga explicitamente qual modo ficou ativo após a negociação
4. Roda na interface física real (ex.:. enp2s0f1), não mais veth agora faz
   sentido testar em hardware real, já que a pergunta é sobre driver

## Requisitos obrigatórios:

- Negociação de modo não pode mascarar outros erros de bind (só cai pra
  copy se o erro for especificamente sobre suporte a zero-copy, não
  qualquer erro)
- README documenta: driver da máquina (r8169), modo que rodou de fato,
  e o que seria necessário pra observar zero-copy real (NIC de servidor
  com driver ixgbe/i40e/mlx5)

O que será observado: se o fallback é condicional ao erro certo (não um
catch-all silencioso que esconderia bugs reais), e se a limitação de
hardware foi documentada com honestidade, não escondida

---

Primeiro passo prático: lembra qual constante de flag (XDP_ZEROCOPY/XDP_COPY) e em qual struct ela entra no bind, é a mesma SockaddrXDP que já foi usado (Flags é campo dela), ou é outro lugar? Dá uma olhada no unix.SockaddrXDP que você já tem importado.

## O que muda do challenge 14:

Fica igual:

- **UMEM**: a região de memória compartilhada continua existindo do mesmo jeito, registrada via `XDP_UMEM_REG`, mesmos frames de tamanho fixo. - Zero-copy não cria uma região de memória diferente, é a mesma região; o que muda é só quem coloca o pacote lá dentro.
- **Os 4 rings** (fill, completion, rx, tx) estrutura idêntica, mesmo mecanismo de producer/consumer index, mesmo jeito de ler/escrever. Nenhuma mudança de código nos fill/receive.
- O socket AF_XDP mesma criação (syscall.Socket(unix.AF_XDP, ...)), mesmo setup de UMEM_FILL_RING/RX_RING via setsockopt.

O que muda, e é a única peça nova:

- A chamada de bind, hoje o unix.Bind(fd, sa) provavelmente não passa nenhuma flag em SockaddrXDP.Flags (ou passa zero). A negociação de hoje é: tenta bind com sa.Flags = unix.XDP_ZEROCOPY; se o kernel recusar especificamente por falta de suporte do driver, você refaz o bind com sa.Flags = unix.XDP_COPY.

Diferença conceitual, pra fixar o porquê: pensa na UMEM como uma caixa de correio compartilhada entre você e o carteiro (kernel). Zero-copy = o carteiro (driver da NIC) coloca a carta direto na sua caixa, sem intermediário. Copy mode = o carteiro primeiro recebe a carta na central dos correios (buffer interno da NIC/kernel), e só depois copia ela pra sua caixa. A caixa (UMEM) é a mesma caixa nos dois casos muda só se tem uma cópia extra no meio do caminho ou não.

Então, na prática, o socket.go que você já tem quase não muda só a função que faz bind ganha um parâmetro de modo desejado e um retry condicional. Não precisa recriar UMEM nem rings nenhum.

---

## Hardware testado

- Interface: enp2s0f1
- Driver: r8169 (confirmado via readlink -f /sys/class/net/enp2s0f1/device/driver)
- Resultado: bind com zero-copy falhou com "operation not supported" r8169 não implementa AF_XDP zero-copy (ndo_bpf XDP_SETUP_XSK_POOL). Fallback pra copy mode funcionou, socket recebe pacote real corretamente.

### Como checar suporte de driver em outra máquina

Sem depender de ethtool:

```bash
ip link show                                          # lista interfaces
readlink -f /sys/class/net/<iface>/device/driver       # resolve o nome do driver
```

Drivers conhecidos por suportar AF_XDP zero-copy: ixgbe, i40e, ice, mlx5_core, mlx4_en (majoritariamente NICs de servidor 10G+). NICs de consumidor/desktop (r8169, e1000e, maioria dos wifi de notebook) geralmente não suportam.

### O que mudou em relação ao challenge 14

Mesma UMEM, mesmos 4 rings, mesmo setup de socket o bind agora negocia o modo explicitamente: tenta XDP_ZEROCOPY primeiro, cai pra XDP_COPY em [erro real que seu errors.Is está checando], registrando qual modo ficou ativo de fato em vez de deixar o kernel escolher silenciosamente.

### O que zero-copy exigiria aqui

Uma NIC de servidor com driver compatível (ixgbe/i40e/ice/mlx5) não disponível nessa máquina. A lógica de negociação está correta e usaria zero-copy automaticamente nesse tipo de hardware, sem mudança de código.

Como é sua interface real, qualquer tráfego normal da sua máquina (abrir um site, ping 8.8.8.8) já deveria gerar pacotes chegando nela. Não precisa simular nada como no veth:

```bash
# em outro terminal, só pra garantir que tem pacote fluindo
ping -c 5 8.8.8.8
```

Com a correção + fallback que foi feita:

```bash
(base) victor@pop-os:~/Pessoal/go-network-engineering/15-xdp_redirect_zero_copy$ sudo ./bin/af_xdp enp2s0f1
[sudo] senha para victor: 
2026/08/07 20:42:51 WARN zero-copy bind failed, falling back to copy mode error="operation not supported"
2026/08/07 20:42:51 INFO AF_XDP socket bound in copy mode
2026/08/07 20:42:51 AF_XDP socket bound to enp2s0f1 queue 0, waiting for packets...
2026/08/07 20:42:52 packet: 114 bytes, first bytes: d4939029510964614055d76f86dd600e
2026/08/07 20:42:52 packet: 114 bytes, first bytes: d4939029510964614055d76f86dd600e
2026/08/07 20:42:53 packet: 114 bytes, first bytes: d4939029510964614055d76f86dd600b
2026/08/07 20:42:54 packet: 114 bytes, first bytes: d4939029510964614055d76f86dd600d
2026/08/07 20:42:54 packet: 66 bytes, first bytes: d4939029510964614055d76f08004500
2026/08/07 20:42:55 packet: 78 bytes, first bytes: d4939029510964614055d76f08004500
2026/08/07 20:42:55 packet: 114 bytes, first bytes: d4939029510964614055d76f86dd6005
2026/08/07 20:42:55 packet: 78 bytes, first bytes: d4939029510964614055d76f08004500
2026/08/07 20:42:56 packet: 78 bytes, first bytes: d4939029510964614055d76f08004500
```