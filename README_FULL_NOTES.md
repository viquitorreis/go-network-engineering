## Overview

## 1. Event Bus

### Aprendizados desse challenge:

- Fan-out pattern - um evento distribuído para múltiplos subscribers
- Closure bug em goroutines - passar parâmetros vs capturar variáveis do loop
- Thread-safety - usar sync.RWMutex com maps compartilhados
- Buffered vs unbuffered channels - trade-offs de performance
- Ordem vs concorrência - quando goroutines ajudam e quando atrapalham
- Trade-offs de sistemas distribuídos - consistency vs availability

Implemente um sistema simples de Event Bus onde:

- Publishers enviam eventos (strings como "user.login", "order.created")
- Subscribers se registram para receber todos os eventos
- Quando um evento é publicado, todos os subscribers recebem concorrentemente
- Cada subscriber processa eventos de forma independente

### Requisitos Funcionais

1. EventBus deve permitir

- Subscribe(name string) <-chan string - registra um subscriber, retorna channel read-only
- Publish(event string) - envia evento para todos subscribers
- Close() - fecha o bus e todos os channels de subscribers

2. Comportamento esperado

- Publicar evento quando não há subscribers -> evento é perdido (ok)
- Subscribers recebem eventos na ordem publicada
- Se um subscriber está lento, não bloqueia outros subscribers
- Fechar o bus -> fecha todos os channels de subscribers

**Constraints**

- Não usar libs externas (só stdlib)
- Implemente uma fila para cada subscriber
- Tempo: ~1 hora
- Código: 80-120 linhas
- Usar sync package conforme necessário

## 2. Log Aggregator (Fan-in)

### O Desafio

Você tem múltiplos serviços (API, Database, Cache) gerando logs concorrentemente. Precisa de um agregador central que coleta todos esses logs e garante que nenhum se perde.

Aprendizados desse challenge:

- Fan-in pattern (N -> 1) - múltiplos producers enviando para um único channel intermediário (bridge) que é consumido por uma goroutine
- for range em channels - o padrão idiomático para processar todos os valores até o channel fechar, bloqueia automaticamente quando não há dados
- Select com default cria busy-waiting - evitar usar default quando você quer bloquear esperando dados, channels já dormem eficientemente
- break dentro de select - só quebra o select, não o for externo (use return para sair da goroutine inteira)
- WaitGroup regra crítica: Add antes de Wait - nunca chamar Wait() antes que todos os Add() tenham sido executados, causa race condition no contador interno
- WaitGroup para coordenação - Add(1) antes de criar goroutine, defer wg.Done() na goroutine, Wait() para bloquear até todas terminarem
- Onde criar a goroutine auxiliar importa - se criar no Start() e ela chamar Wait() imediatamente, vai retornar antes dos Register() adicionarem ao WaitGroup; melhor chamar Wait() no Stop()
- Variable shadowing com := - usar bridge := make(chan) cria variável local que esconde la.bridge, deixando o campo do struct como nil; sempre usar la.bridge = make(chan) para inicializar campos
- Inicializar bridge no lugar certo - criar no Start() ao invés do construtor, senão a goroutine auxiliar pode fechar o channel antes de qualquer Register() acontecer
- Done channel para sincronizar término - usar chan struct{} que a goroutine consumidora fecha quando termina, permitindo Stop() esperar sem race condition em la.logs
- defer dentro de loop acumula - não executa a cada iteração, só no final da função (causa deadlock com mutex em loops)
- Single writer pattern - quando só uma goroutine escreve em uma estrutura, não precisa de mutex (sem race condition)
- Ownership de channels - quem cria o channel deve fechá-lo, não tente fechar channels de terceiros (causa panic)
- Graceful shutdown em camadas - producers fecham channels -> leitoras terminam e Done() -> Stop() fecha bridge -> consumidora termina e fecha done channel -> Stop() retorna logs
- Channel buffering - buffer no bridge permite leitoras continuarem enviando mesmo se consumidora estiver ocupada (trade-off latência vs throughput)
- Send on closed channel panic - tentar enviar para channel fechado causa panic; garantir que bridge só é fechado quando todas as goroutines leitoras já terminaram

### Os 3 Problemas Principais

**1. Fan-in (N -> 1)**: Como ler de múltiplos channels ao mesmo tempo?
- Não dá pra fazer `for` em cada channel separado (seria sequencial)
- Precisa de algo que espera em TODOS simultaneamente

**2. Graceful Shutdown**: Como saber quando TODOS os producers terminaram?
- Um producer fecha seu channel... mas ainda tem outros rodando
- Você só pode parar quando o ÚLTIMO fechar

**3. Coordenação**: Como avisar o aggregator que pode parar?
- Quem vai fechar o channel central?
- Como sincronizar isso com os producers?

### De forma simples:

**Fan-in**

- Vários produtores.
- Um consimudor.

**Produtores:** criam logs, api, database, etc
**Consumidor:** aggregator.

Aggregator deve registrar os channels para ler depois. Estrutura simples.

**O que é Fan-in de verdade?**

Fan-in significa que você precisa estar lendo de TODOS os channels ao mesmo tempo.
Quando qualquer um deles tiver um log pronto, processa. É concorrente, não sequencial.

### Requisitos Funcionais

1. **LogAggregator deve permitir:**
   - `Register(logChan <-chan LogEntry)` - registra um channel de producer para agregar
   - `Start()` - inicia o processamento de logs de todos os producers
   - `Stop() []LogEntry` - para gracefully e retorna todos os logs coletados

2. **Comportamento esperado:**
   - Múltiplos producers podem enviar logs simultaneamente
   - Nenhum log pode ser perdido (todos devem ser coletados)
   - Se um producer termina antes dos outros, seus logs já devem estar agregados
   - `Stop()` só retorna quando TODOS os producers terminaram e todos os logs foram processados
   - Producers mais lentos não bloqueiam producers mais rápidos

### Constraints

- Não usar libs externas (só stdlib)
- Tempo: ~1 hora
- Código: 80-160 linhas
- Usar `sync` package conforme necessário
- Implementar graceful shutdown sem perder logs

### Aprendizados desse challenge:

### Template

```go
```

## 3. Image Processor Pipeline

Você vai construir um pipeline de 4 stages onde cada stage processa imagens e passa para o próximo, tudo rodando concorrentemente. É como uma linha de montagem, enquanto o Stage 1 está listando novos arquivos, o Stage 2 já está carregando imagens anteriores, o Stage 3 processando (converter em gray scale), e o Stage 4 salvando.

**O que fazer**: stages encadeados onde cada um procesa e passa para o próximo.

**Importante**: No Fan-In tinhamos multiplos produtores enviando para UM channel central. Aqui temos uma **corrente de channels**:

```
Generator -> [channel1] -> Loader -> [channel2] -> Processor -> [channel3] -> Saver
   (lista)              (carrega)              (processa)              (salva)
   arquivos             imagem                 grayscale               disco
```

### Os 4 Stages

**Stage 1** - Generator: Lista todos os arquivos .jpg e .png do diretório de input

- Saída: channel de ImageJob com apenas o Path preenchido

**Stage 2** - Loader: Carrega cada imagem do disco para memória

- Entrada: jobs com path
- Saída: jobs com Image carregado

**Stage 3** - Processor: Converte imagens para grayscale

- Entrada: jobs com imagem colorida
- Saída: jobs com imagem em tons de cinza

**Stage 4** - Saver: Salva imagens processadas no disco

- Entrada: jobs com imagem processada
- Saída: nenhuma (é o fim do pipeline)

### Os 3 Desafios Principais

#### 1. Conectar os Stages com Channels

- Cada stage lê de um channel e escreve no próximo
- Você precisa criar os channels intermediários e conectar tudo no Run()

#### 2. Graceful Shutdown

- Cada stage precisa fechar seu output channel quando o input channel fechar
- Como saber quando o pipeline inteiro terminou? (WaitGroup para cada stage?)

#### 3. Context Cancellation

- Se o context cancelar (timeout ou erro), todos os stages devem parar imediatamente
- Use select com ctx.Done() em pontos estratégicos

### Como começar

1. Implementar o ```generator```
	- Use ```filepath.Glob()``` para listar os arquivos

```go
files, _ := filepath.Glob(filepath.Join(inputDir, "*.jpg"))
```

2. Implementar o ```loader```
	- Use ```image.Decode()```

```go
file, _ := os.Open(job.Path)
img, _, _ := image.Decode(file)
```

3. Implemente o ```processor```
	- Loop pelos pixels e converta para grayscale:

```
grayImg := image.NewGray(img.Bounds())
// loop pelos pixels, calcular média RGB, setar no grayImg
```

4. Implementar o ```saver```. Use ```jpeg.Encode()```

jpeg.Encode(file, job.Image, &jpeg.Options{Quality: 90})

5. Conecte tudo no ```Run()``` criando channels e lançando goroutines

### Dicas Importantes

- Fechar channels: Cada stage deve fazer defer close(outputChan) no início
- WaitGroup: Você vai precisar de um WaitGroup para cada stage ou um para todos? Pensa no fluxo
- Buffered channels: Use buffer nos channels intermediários (ex: 10) para evitar bloqueios
- Context: Em cada loop, faça select { case <-ctx.Done(): return } para respeitar cancellation

### Para Testar

```
# Rodar testes
go test -v

# Verificar race conditions
go test -race
```

### Main e assinaturas:

```go
package main

import (
	"context"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"time"
)

func main() {
	fmt.Println("=== Image Processing Pipeline ===")
	fmt.Println("Pipeline Pattern: Generator -> Loader -> Processor -> Saver")

	// Criar diretórios se não existirem
	inputDir := "./input_images"
	outputDir := "./output_images"
	
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		fmt.Printf("Erro criando input dir: %v\n", err)
		return
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Erro criando output dir: %v\n", err)
		return
	}

	// Criar algumas imagens de teste se não existirem
	createTestImages(inputDir)

	// Criar pipeline
	pipeline := NewPipeline(inputDir, outputDir)

	// Context com timeout de 30 segundos
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Executar pipeline
	fmt.Printf("Processando imagens de %s...\n", inputDir)
	start := time.Now()

	if err := pipeline.Run(ctx); err != nil {
		fmt.Printf("Erro no pipeline: %v\n", err)
		return
	}

	elapsed := time.Since(start)
	fmt.Printf("\n✓ Pipeline concluído em %v\n", elapsed)
	fmt.Printf("✓ Imagens salvas em %s\n", outputDir)
	fmt.Println("Run 'go test -v' to verify your implementation")
}

// ImageJob representa uma imagem no pipeline
type ImageJob struct {
	Path   string      // caminho do arquivo original
	Image  image.Image // imagem carregada (nil nos primeiros stages)
	Error  error       // erro se algo deu errado
	StageNum int       // apenas para debug
}

// Pipeline gerencia os 4 stages de processamento
type Pipeline struct {
	// TODO: adicione campos necessários para:
	// - Channels entre stages
	// - Context para cancellation
	// - WaitGroup para coordenação
	// - Caminho dos diretórios input/output
}

func NewPipeline(inputDir, outputDir string) *Pipeline {
	return &Pipeline{
		// TODO: inicialize campos
	}
}

// Run executa o pipeline completo
func (p *Pipeline) Run(ctx context.Context) error {
	// TODO: conecte os 4 stages usando channels
	// Generator -> fileChan -> Loader -> imageChan -> Processor -> processedChan -> Saver
	
	// Dica: cada stage deve ser uma goroutine
	// Dica: use WaitGroup para saber quando tudo terminou
	// Dica: passe o context para poder cancelar em caso de erro
	
	return nil
}

// Stage 1: Generator - lista arquivos de imagens no diretório
func (p *Pipeline) generator(ctx context.Context, outputChan chan<- ImageJob) {
	// TODO: 
	// 1. Listar arquivos .jpg e .png no inputDir usando filepath.Glob ou filepath.Walk
	// 2. Para cada arquivo, criar um ImageJob com o Path
	// 3. Enviar para o outputChan
	// 4. Fechar o outputChan quando terminar (importante!)
	// 5. Respeitar o context - se ctx.Done(), parar imediatamente
	
	defer close(outputChan)
	
	// Hint: filepath.Glob("dir/*.jpg") ou filepath.Walk
	// Hint: select { case <-ctx.Done(): return; case outputChan <- job: }
}

// Stage 2: Loader - carrega imagens do disco
func (p *Pipeline) loader(ctx context.Context, inputChan <-chan ImageJob, outputChan chan<- ImageJob) {
	// TODO:
	// 1. Receber jobs do inputChan
	// 2. Para cada job, abrir o arquivo (os.Open)
	// 3. Decodificar a imagem (image.Decode ou jpeg.Decode/png.Decode)
	// 4. Colocar a imagem no job.Image
	// 5. Se der erro, colocar em job.Error
	// 6. Enviar job para outputChan
	// 7. Fechar outputChan quando inputChan fechar
	
	defer close(outputChan)
	
	// Hint: import "image/jpeg" e "image/png"
	// Hint: image.Decode detecta o formato automaticamente
	// Hint: não esqueça de fechar o arquivo: defer file.Close()
}

// Stage 3: Processor - processa as imagens (grayscale)
func (p *Pipeline) processor(ctx context.Context, inputChan <-chan ImageJob, outputChan chan<- ImageJob) {
	// TODO:
	// 1. Receber jobs com imagens carregadas
	// 2. Converter para grayscale (percorrer pixels e calcular média RGB)
	// 3. Ou redimensionar (usar bounds e criar nova imagem menor)
	// 4. Atualizar job.Image com a imagem processada
	// 5. Enviar para outputChan
	
	defer close(outputChan)
	
	// Hint: bounds := img.Bounds()
	// Hint: new image: image.NewGray(bounds) ou image.NewRGBA(bounds)
	// Hint: loop: for y := bounds.Min.Y; y < bounds.Max.Y; y++ { for x := ... }
}

// Stage 4: Saver - salva imagens processadas
func (p *Pipeline) saver(ctx context.Context, inputChan <-chan ImageJob) {
	// TODO:
	// 1. Receber jobs com imagens processadas
	// 2. Criar arquivo no outputDir (mesmo nome, adicionar sufixo "_processed")
	// 3. Encode da imagem no formato JPEG
	// 4. Fechar arquivo
	// 5. Imprimir sucesso ou erro
	
	// Hint: novoNome := strings.TrimSuffix(basename, ext) + "_processed.jpg"
	// Hint: jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
	// Não precisa fechar channel aqui - é o último stage
}
```

## 4. TCP Chat Server

Este desafio é diferente dos anteriores porque você vai trabalhar com sockets TCP de baixo nível. Quando alguém conecta com telnet localhost 6969, você está recebendo uma conexão TCP bruta. Você precisa ler bytes dessa conexão, interpretar como mensagens, e enviar bytes de volta. É exatamente assim que servidores reais funcionam por baixo do capô.

### A Arquitetura do Sistema

Pensa num chat como Discord ou Slack. Quando você envia uma mensagem, ela precisa chegar em todos os outros usuários que estão online naquele canal. Mas tem algumas complexidades interessantes aqui.

- Primeiro, cada usuário tem sua própria conexão de rede independente, você não pode simplesmente "gritar" a mensagem e todo mundo ouve. Você precisa enviar ativamente para cada conexão individual.
- Segundo, essas conexões podem estar em velocidades diferentes, um usuário pode estar numa conexão lenta de celular enquanto outro está em fibra ótica. Se você esperar o usuário lento responder antes de enviar para o próximo, todos ficam esperando.
- Terceiro, usuários podem desconectar a qualquer momento sem avisar, o cabo de rede pode ser desconectado fisicamente.

### Os três Componentes Principais

Você vai construir três sistemas que trabalham juntos.

1. O primeiro é o **sistema de lobby**, que é uma sala de espera. Imagine que você está organizando um jogo de futebol e precisa esperar pelo menos dois jogadores chegarem antes de começar. Enquanto só tem um jogador, ele fica esperando. Quando o segundo chega, você libera ambos para jogar. Este sistema usa **sync.Cond**, que é uma primitiva de sincronização perfeita para "acordar" múltiplas goroutines quando uma condição muda. É como um alarme que toca para todo mundo ao mesmo tempo.

2. O segundo componente é o **sistema de broadcast**. Quando um cliente envia uma mensagem, você precisa distribuir para todos os outros clientes conectados. A forma mais idiomática em Go é dar para cada cliente seu próprio channel de mensagens. Quando alguém envia uma mensagem, você itera pelos clientes e envia a mensagem para o channel de cada um. Cada cliente tem uma goroutine dedicada lendo desse channel e escrevendo na conexão TCP. Isso resolve o problema de clientes lentos - se o channel de um cliente encher porque ele está lento, você simplesmente pula ele ao invés de bloquear todos os outros.

3. O terceiro componente é o **gerenciamento de conexões**. Cada cliente precisa de pelo menos duas goroutines, uma que fica lendo da conexão TCP (esperando mensagens chegarem) e outra que fica lendo do channel de mensagens e escrevendo na conexão TCP (enviando mensagens para o cliente). Quando um cliente desconecta, você precisa limpar esses recursos - fechar goroutines, remover do map de clientes ativos, fechar channels. Se não fizer isso direito, você tem vazamento de goroutines e memória.

### Como começar

Comece pelo mais simples possível. Implemente apenas o ```Start``` e faça ele aceitar uma conexão. Nem precisa fazer nada com a conexão ainda, só aceitar e imprimir que alguém conectou. Teste com telnet localhost 6969 em outro terminal. Quando você vir "cliente conectou", você sabe que a parte de networking básica funciona.


```go
package main

import (
	"context"
	"net"
	"time"
)

/*
═══════════════════════════════════════════════════════════════════════════
TODO - PASSOS PARA IMPLEMENTAR O TCP CHAT SERVER
═══════════════════════════════════════════════════════════════════════════

1. CRIAR O SERVER
   - Usar net.Listen para escutar em uma porta (ex: :6969)
   - Aceitar conexões de clientes com Accept() em um loop
   - Para cada cliente que conecta, criar uma goroutine

2. LOBBY (SALA DE ESPERA)
   - Usar sync.Cond para bloquear clientes até ter mínimo de 2 players
   - Quando cliente conecta, incrementar contador
   - Se atingir mínimo, fazer Broadcast() para liberar todos
   - Clientes ficam esperando em Wait() até o Broadcast

3. GERENCIAR CLIENTES
   - Guardar cada cliente em um map (clientID -> Client)
   - Cada cliente precisa de um channel para receber mensagens
   - Quando cliente envia mensagem, fazer broadcast para TODOS os outros

4. BROADCAST DE MENSAGENS
   - Ler mensagem do cliente A
   - Enviar para os channels de todos os outros clientes (B, C, D...)
   - Usar goroutines separadas: uma lê, outra escreve

5. HANDLE DISCONNECT
   - Quando cliente desconecta, remover do map
   - Notificar outros clientes
   - Fechar o channel do cliente que saiu

6. GRACEFUL SHUTDOWN
   - Context para cancelar tudo quando server parar
   - Fechar todas as conexões ativas
   - Esperar goroutines terminarem com WaitGroup

═══════════════════════════════════════════════════════════════════════════
*/


func main() {
	fmt.Println("=== TCP Chat Server com Lobby ===")
	fmt.Println("Network Programming: TCP + sync.Cond + Broadcast")

	// Criar servidor que espera mínimo de 2 players
	server := NewChatServer("6969", 2)

	// Context com timeout de 5 minutos
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Iniciar servidor em goroutine
	go func() {
		fmt.Println("👽 Server listening on :6969")
		fmt.Println("Connect with: telnet localhost 6969")
		fmt.Println("Waiting for at least 2 players to start...")
		
		if err := server.Start(ctx); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	// Simular alguns clients para teste (remova isso depois)
	time.Sleep(1 * time.Second)
	
	fmt.Println("To test:")
	fmt.Println("   Terminal 1: telnet localhost 6969")
	fmt.Println("   Terminal 2: telnet localhost 6969")
	fmt.Println("   Type messages and press Enter")
	fmt.Println("   Press Ctrl+C to stop server")

	// Manter servidor rodando
	<-ctx.Done()
	fmt.Println("Timeout reached, stopping server...")


	server.Stop()
	fmt.Println("Server stopped")
}

// Message representa uma mensagem no chat
type Message struct {
	From    string
	Content string
	Time    time.Time
}

// Client representa um cliente conectado
type Client struct {
	// TODO: adicione campos necessários:
	// - ID único do cliente
	// - Conexão net.Conn
	// - Channel para receber mensagens
	// - Reader/Writer para ler/escrever na conexão
}

// ChatServer gerencia o servidor de chat
type ChatServer struct {
	// TODO: adicione campos necessários:
	// - Porta do servidor
	// - Map de clientes ativos
	// - Mutex para proteger o map
	// - sync.Cond para o lobby
	// - Contador de clientes
	// - Mínimo de players para começar
	// - WaitGroup para coordenação
}

type IChatServer interface {
	Start(ctx context.Context) error
	Stop() error
}

// NewChatServer cria um novo servidor de chat
func NewChatServer(port string, minPlayers int) IChatServer {
	return &ChatServer{
		// TODO: inicialize os campos
	}
}

// Start inicia o servidor TCP
func (s *ChatServer) Start(ctx context.Context) error {
	// TODO:
	// 1. Criar listener TCP com net.Listen("tcp", ":"+port)
	// 2. Loop infinito aceitando conexões com listener.Accept()
	// 3. Para cada conexão aceita, criar um Client e chamar handleClient em goroutine
	// 4. Respeitar ctx.Done() para shutdown

	// Hint:
	// listener, err := net.Listen("tcp", ":6969")
	// for { conn, err := listener.Accept() }

	return nil
}

// handleClient gerencia um cliente conectado
func (s *ChatServer) handleClient(ctx context.Context, conn net.Conn) {
	// TODO:
	// 1. Criar um Client com ID único e conexão
	// 2. Adicionar cliente ao map (thread-safe com mutex)
	// 3. LOBBY: Verificar se atingiu mínimo de players
	//    - Se sim, fazer Broadcast() para liberar todos
	//    - Se não, fazer Wait() para esperar
	// 4. Iniciar duas goroutines:
	//    - readLoop: lê mensagens do cliente
	//    - writeLoop: envia mensagens para o cliente
	// 5. Quando cliente desconectar, remover do map e notificar outros

	defer conn.Close()

	// Hint: Use sync.Cond para o lobby
	// s.cond.L.Lock()
	// s.playerCount++
	// if s.playerCount >= s.minPlayers {
	//     s.cond.Broadcast()
	// } else {
	//     s.cond.Wait()
	// }
	// s.cond.L.Unlock()
}

// readLoop lê mensagens de um cliente
func (s *ChatServer) readLoop(ctx context.Context, client *Client) {
	// TODO:
	// 1. Criar um bufio.Scanner ou bufio.Reader na conexão
	// 2. Loop lendo linhas da conexão
	// 3. Para cada mensagem recebida, fazer broadcast para todos outros clientes
	// 4. Se erro de leitura (cliente desconectou), sair do loop

	// Hint:
	// scanner := bufio.NewScanner(client.conn)
	// for scanner.Scan() {
	//     msg := scanner.Text()
	//     s.broadcast(client.ID, msg)
	// }
}

// writeLoop envia mensagens para um cliente
func (s *ChatServer) writeLoop(ctx context.Context, client *Client) {
	// TODO:
	// 1. Loop lendo do channel de mensagens do cliente
	// 2. Para cada mensagem, escrever na conexão
	// 3. Se ctx cancelar ou channel fechar, sair

	// Hint:
	// for msg := range client.messages {
	//     fmt.Fprintf(client.conn, "%s: %s\n", msg.From, msg.Content)
	// }
}

// broadcast envia uma mensagem para todos os clientes exceto o remetente
func (s *ChatServer) broadcast(fromID string, content string) {
	// TODO:
	// 1. Criar uma mensagem
	// 2. Iterar pelo map de clientes (thread-safe com mutex)
	// 3. Para cada cliente diferente do remetente:
	//    - Enviar mensagem para o channel do cliente
	//    - Usar select com default para não bloquear se channel cheio

	// Hint: Use select com default para não bloquear
	// select {
	// case client.messages <- msg:
	// default:
	//     // Cliente está lento/desconectado, pular
	// }
}

// removeClient remove um cliente e notifica outros
func (s *ChatServer) removeClient(clientID string) {
	// TODO:
	// 1. Lock no mutex
	// 2. Remover cliente do map
	// 3. Fechar o channel de mensagens do cliente
	// 4. Unlock no mutex
	// 5. Fazer broadcast que o cliente saiu
	// 6. Decrementar playerCount
}

// Stop para o servidor gracefully
func (s *ChatServer) Stop() error {
	// TODO:
	// 1. Fechar listener
	// 2. Fechar todas as conexões ativas
	// 3. Esperar WaitGroup terminar

	return nil
}
```

- The test file is: 

```go
package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestServer_AcceptsConnections(t *testing.T) {
	server := NewChatServer("18080", 1) // porta diferente para não conflitar

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Iniciar servidor
	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond) // dar tempo pro servidor subir

	// Conectar cliente
	conn, err := net.Dial("tcp", "localhost:18080")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	t.Log("Client connected successfully")
}

func TestServer_LobbyWaitsForMinPlayers(t *testing.T) {
	server := NewChatServer("18081", 2) // precisa de 2 players

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Conectar primeiro cliente
	conn1, err := net.Dial("tcp", "localhost:18081")
	if err != nil {
		t.Fatalf("Client 1 failed to connect: %v", err)
	}
	defer conn1.Close()

	// Dar tempo para entrar no lobby
	time.Sleep(200 * time.Millisecond)

	// Conectar segundo cliente
	conn2, err := net.Dial("tcp", "localhost:18081")
	if err != nil {
		t.Fatalf("Client 2 failed to connect: %v", err)
	}
	defer conn2.Close()

	// Quando 2o cliente conecta, ambos devem ser liberados do lobby
	time.Sleep(200 * time.Millisecond)

	t.Log("Lobby released after 2 players connected")
}

func TestServer_BroadcastMessage(t *testing.T) {
	server := NewChatServer("18082", 2)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Conectar dois clientes
	conn1, err := net.Dial("tcp", "localhost:18082")
	if err != nil {
		t.Fatalf("Client 1 failed: %v", err)
	}
	defer conn1.Close()

	conn2, err := net.Dial("tcp", "localhost:18082")
	if err != nil {
		t.Fatalf("Client 2 failed: %v", err)
	}
	defer conn2.Close()

	// Esperar lobby liberar
	time.Sleep(300 * time.Millisecond)

	// Cliente 1 envia mensagem
	fmt.Fprintf(conn1, "Hello from client 1\n")

	// Cliente 2 deve receber
	reader := bufio.NewReader(conn2)
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))

	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Client 2 didn't receive message: %v", err)
	}

	if !strings.Contains(line, "Hello from client 1") {
		t.Errorf("Expected message with 'Hello from client 1', got: %s", line)
	}

	t.Log("Message broadcast successfully")
}

func TestServer_MultipleClients(t *testing.T) {
	server := NewChatServer("18083", 2)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Conectar 3 clientes
	var conns []net.Conn
	for i := 0; i < 3; i++ {
		conn, err := net.Dial("tcp", "localhost:18083")
		if err != nil {
			t.Fatalf("Client %d failed: %v", i+1, err)
		}
		conns = append(conns, conn)
		defer conn.Close()
	}

	// Esperar lobby liberar
	time.Sleep(300 * time.Millisecond)

	// Cliente 1 envia mensagem
	fmt.Fprintf(conns[0], "Broadcast test\n")

	// Clientes 2 e 3 devem receber
	for i := 1; i < 3; i++ {
		reader := bufio.NewReader(conns[i])
		conns[i].SetReadDeadline(time.Now().Add(2 * time.Second))

		line, err := reader.ReadString('\n')
		if err != nil {
			t.Errorf("Client %d didn't receive: %v", i+1, err)
			continue
		}

		if !strings.Contains(line, "Broadcast test") {
			t.Errorf("Client %d got wrong message: %s", i+1, line)
		}
	}

	t.Log("Message broadcast to all clients")
}

func TestServer_ClientDisconnect(t *testing.T) {
	server := NewChatServer("18084", 2)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Conectar 3 clientes
	conn1, _ := net.Dial("tcp", "localhost:18084")
	conn2, _ := net.Dial("tcp", "localhost:18084")
	conn3, _ := net.Dial("tcp", "localhost:18084")

	time.Sleep(300 * time.Millisecond)

	// Cliente 1 desconecta
	conn1.Close()
	time.Sleep(200 * time.Millisecond)

	// Cliente 2 envia mensagem
	fmt.Fprintf(conn2, "After disconnect\n")

	// Cliente 3 deve receber (cliente 1 não, pois desconectou)
	reader := bufio.NewReader(conn3)
	conn3.SetReadDeadline(time.Now().Add(2 * time.Second))

	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Client 3 should still receive: %v", err)
	}

	if !strings.Contains(line, "After disconnect") {
		t.Errorf("Got wrong message: %s", line)
	}

	conn2.Close()
	conn3.Close()

	t.Log("Server handles disconnect correctly")
}

func TestServer_EmptyMessage(t *testing.T) {
	server := NewChatServer("18085", 1)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", "localhost:18085")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Enviar mensagem vazia
	fmt.Fprintf(conn, "\n")

	time.Sleep(100 * time.Millisecond)

	t.Log("Server handles empty messages")
}

// Teste de stress - múltiplas mensagens rápidas
func TestServer_RapidMessages(t *testing.T) {
	server := NewChatServer("18086", 2)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go server.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	conn1, _ := net.Dial("tcp", "localhost:18086")
	defer conn1.Close()

	conn2, _ := net.Dial("tcp", "localhost:18086")
	defer conn2.Close()

	time.Sleep(300 * time.Millisecond)

	// Enviar 50 mensagens rápidas
	go func() {
		for i := 0; i < 50; i++ {
			fmt.Fprintf(conn1, "Message %d\n", i)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Cliente 2 deve receber todas (ou a maioria se buffer encher)
	reader := bufio.NewReader(conn2)
	received := 0

	for i := 0; i < 50; i++ {
		conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		received++
	}

	if received < 45 { // Aceitar perder algumas mensagens por buffer cheio
		t.Errorf("Only received %d/50 messages", received)
	} else {
		t.Logf("Received %d/50 rapid messages", received)
	}
}
```

## 5. Rate Limiter Com Token Bucket

Você vai implementar um rate limiter thread-safe usando o algoritmo **Token Bucket**. 

Este é um componente fundamental em sistemas de produção, toda API pública usa alguma forma de rate limiting. Empresas como Stripe, GitHub, e AWS usam variações desse algoritmo para proteger seus serviços contra sobrecarga.

**O que você vai construir:** Um rate limiter que aceita ou rejeita requisições baseado em um bucket de tokens que se reabastece continuamente. Múltiplas goroutines vão tentar consumir tokens simultaneamente enquanto uma goroutine em background adiciona tokens periodicamente.

### Introdução

**Por quê fazer esse challenge?**

Rate limiting aparece em todo lugar em engenharia de backend. Quando você faz uma chamada para a API do GitHub, eles te dão 5000 requests/hora. Quando você usa Redis Cloud, eles limitam operações por segundo. Quando você constrói uma API REST, você precisa proteger seus endpoints de abuse.

**Empresas que usam:** Stripe (payment processing), CloudFlare (DDoS protection), Redis (cloud limits), toda API REST moderna. Em entrevistas para Tailscale, LiveKit, ou GitLab, e FAANGS saber implementar rate limiting do zero é diferencial forte.

**O que exatamente você vai construir?**

Um rate **limiter** baseado em **Token Bucket** que:

- Mantém um bucket com capacidade máxima de N tokens
- Adiciona tokens ao bucket a uma taxa constante (ex: 10 tokens/segundo)
- Quando uma requisição chega, tenta consumir 1 token
- Se tem token disponível -> aceita a requisição
- Se não tem token -> rejeita (ou opcionalmente espera)

Você vai testar isso simulando 100 goroutines fazendo requests simultaneamente. Algumas vão passar, outras vão ser rejeitadas conforme os tokens esgotam e se reabastecem.

**Quais são os 2-3 desafios principais?**

1. **Reabastecimento contínuo de tokens:** Como adicionar tokens periodicamente sem criar um timer por requisição? Pense direito para não gerar goroutines leaks.
2. **sincronização entre consumidores e producers:** Múltiplas goroutines tentando consumir tokens simultaneamente, enquanto uma goroutine adiciona tokens. Você precisa de mutex, mas onde? Se for um mutex geral, vai ter uma performance ruim. Se não sincronizar corretamente, perde token ou conta errado.
3. **Modelar o bucket de forma eficiente:** *Não é permitido* ter um array gigante com todos os tokens. Precisa apenas rastrear quantos tokens existem atualmente e quando foi a última vez que adicionou tokens. É um problema de modelagem de estado.

### Background Técnico

**Pattern: Token Bucket Algorithm**

Imagine um balde que comporta no máximo 100 moedas (tokens). A cada segundo, alguém joga 10 moedas novas no balde. Se já está cheio, as moedas extras caem fora (capacidade máxima). Quando uma pessoa quer fazer alguma coisa, ela precisa pegar uma moeda do balde. Se não tem moeda disponível, ela não pode prosseguir.

**Token bucket vs Leaky Bucket:** Token Bucket permite bursts (se o bucket está cheio, você pode consumir vários tokens rapidamente). Leaky Bucket Mantém taxa constante (como um funil que drena em velocidade fixa). Para APIs, Token Bucket é mais comum pois permite clientes fazer pequenos bursts sem punição.

#### DSA Usada

Por mais que tenha "bucket" no nome, a implementação não tem um balde como estrutura literal. O truque está em **rastrear estado em vez de elementos individuais:**

```
Estado do bucket:
- tokens (int): quantos tokens existem agora
- capacity (int): máximo de tokens permitidos
- refillRate (int): tokens adicionados por intervalo
- lastRefill (time.Time): quando foi o último reabastecimento
```

Quando alguém tenta consumir um token, primeiro deve calcular **quantos tokens deveriam ter sido adicionados desde lastRefill**, adiciona eles respeitando a capcidade, e atualiza lastRefill, e então tenta consumir.

Isso é mais eficiente do que usar um timer, pois usa **lazy evaluation**, só calcula tokens quando precisam, não fica acordando goroutine a cada tick...

### Libs Go Relavantes

- sync.Mutex: Proteger leitura/escrita do estado do bucket
- time.Time e time.Since(): Calcular quanto tempo passou desde último refill
- time.Duration: Definir intervalos de refill (ex: 100ms)

**NÃO precisa de**:

- time.Ticker (abordagem mais simples usa lazy evaluation)
- Channels (adiciona complexidade desnecessária aqui)

### Trade-offs Principais

**Precisão vs Performance**

- Usar mutex para cada operação garante precisão mas pode criar contenção em alta carga
- Usar atomic operations é mais rápido mas dificulta lógica de refill
- Para este challenge, prefira clareza e correção (use mutex)

**Lazy Refill vs Active Refill**

- Lazy: calcula tokens na hora da requisição (mais simples, menos goroutines)
- Active: goroutine em background usando ticker (mais preciso em cenários específicos)
- Vamos usar Lazy porque é mais idiomático para este caso

**Rejeitar vs bloquear**

- Rejeitar imediatamente: retorna erro se não tem token
- Bloquear e esperar: goroutine espera até token ficar disponível
- Vamos implementar rejeição, pois bloquear complica testes e pode causar deadlocks.

### Função main:

```go
package main

import (
	"fmt"
	"sync"
	"time"
)

// TokenBucket implementa rate limiting usando o algoritmo Token Bucket
type TokenBucket struct {
	// TODO: adicionar campos necessários
	// Dica: você precisa rastrear:
	// - quantos tokens existem agora
	// - capacidade máxima
	// - taxa de reabastecimento (tokens por intervalo)
	// - intervalo de reabastecimento
	// - timestamp do último refill
	// - mutex para proteger acesso concorrente
}

// NewTokenBucket cria um novo rate limiter
// capacity: número máximo de tokens no bucket
// refillRate: quantos tokens adicionar por intervalo
// refillInterval: com que frequência adicionar tokens (ex: 100ms)
func NewTokenBucket(capacity int, refillRate int, refillInterval time.Duration) *TokenBucket {
	// TODO: inicializar bucket começando cheio (todos os tokens disponíveis)
	return nil
}

// Allow tenta consumir um token
// Retorna true se conseguiu (requisição permitida)
// Retorna false se não tem tokens disponíveis (requisição rejeitada)
func (tb *TokenBucket) Allow() bool {
	// TODO: implementar lógica
	// Passos:
	// 1. Lock do mutex
	// 2. Calcular quantos tokens deveriam ter sido adicionados desde lastRefill
	// 3. Adicionar esses tokens (respeitando capacidade máxima)
	// 4. Atualizar lastRefill para agora
	// 5. Tentar consumir 1 token
	// 6. Unlock e retornar resultado
	return false
}

// refill é um método helper para calcular e adicionar tokens
// Retorna quantos tokens foram adicionados
func (tb *TokenBucket) refill() int {
	// TODO: implementar cálculo de refill
	// Quanto tempo passou desde lastRefill?
	// Quantos "intervalos" completos aconteceram nesse tempo?
	// Quantos tokens isso representa?
	// Não esqueça de respeitar a capacidade máxima!
	return 0
}

// Stats retorna estatísticas atuais do bucket (útil para debugging)
func (tb *TokenBucket) Stats() (available int, capacity int) {
	// TODO: retornar tokens disponíveis e capacidade
	// Precisa de lock? Por quê?
	return 0, 0
}

func main() {
	// Criar rate limiter: 10 tokens no máximo, adiciona 5 tokens a cada 100ms
	// Isso dá ~50 requests/segundo no steady state
	limiter := NewTokenBucket(10, 5, 100*time.Millisecond)

	fmt.Println("Starting rate limiter test...")
	fmt.Printf("Bucket: capacity=%d, refill rate=%d tokens per 100ms\n", 10, 5)
	
	// TODO: Simular múltiplas goroutines fazendo requests
	// Dica: use WaitGroup, lance ~50 goroutines, cada uma tenta Allow()
	// Conte quantas foram aceitas vs rejeitadas
	// Print os resultados
	
	fmt.Println("Test completed!")
}
```

### Testes

```go
package main

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenBucket_InitialTokens(t *testing.T) {
	// Bucket deve começar cheio
	tb := NewTokenBucket(10, 5, 100*time.Millisecond)
	
	available, capacity := tb.Stats()
	if available != capacity {
		t.Errorf("expected bucket to start full: got %d/%d", available, capacity)
	}
}

func TestTokenBucket_ConsumeAllTokens(t *testing.T) {
	tb := NewTokenBucket(5, 1, 100*time.Millisecond)
	
	// Deve conseguir consumir exatamente 5 tokens
	for i := 0; i < 5; i++ {
		if !tb.Allow() {
			t.Fatalf("expected token %d to be available", i+1)
		}
	}
	
	// Sexto deve falhar
	if tb.Allow() {
		t.Error("expected 6th request to be rejected")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	// Bucket pequeno: 5 tokens, adiciona 5 a cada 50ms
	tb := NewTokenBucket(5, 5, 50*time.Millisecond)
	
	// Consumir todos
	for i := 0; i < 5; i++ {
		tb.Allow()
	}
	
	// Verificar que está vazio
	if tb.Allow() {
		t.Error("bucket should be empty")
	}
	
	// Esperar tempo suficiente para refill completo
	time.Sleep(60 * time.Millisecond)
	
	// Deve ter 5 tokens novamente
	allowed := 0
	for i := 0; i < 10; i++ {
		if tb.Allow() {
			allowed++
		}
	}
	
	if allowed != 5 {
		t.Errorf("expected 5 tokens after refill, got %d", allowed)
	}
}

func TestTokenBucket_ConcurrentAccess(t *testing.T) {
	tb := NewTokenBucket(100, 10, 50*time.Millisecond)
	
	var allowed atomic.Int32
	var rejected atomic.Int32
	
	var wg sync.WaitGroup
	// 200 goroutines competindo por 100 tokens
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tb.Allow() {
				allowed.Add(1)
			} else {
				rejected.Add(1)
			}
		}()
	}
	
	wg.Wait()
	
	total := int(allowed.Load()) + int(rejected.Load())
	if total != 200 {
		t.Errorf("expected 200 total requests, got %d", total)
	}
	
	// Deve ter aceito aproximadamente 100 (pode variar um pouco por causa de refill)
	if allowed.Load() < 95 || allowed.Load() > 105 {
		t.Errorf("expected ~100 allowed, got %d", allowed.Load())
	}
}

func TestTokenBucket_RateLimiting(t *testing.T) {
	// 10 tokens, refill 10 a cada 100ms = 100 tokens/segundo
	tb := NewTokenBucket(10, 10, 100*time.Millisecond)
	
	start := time.Now()
	allowed := 0
	
	// Tentar consumir 50 tokens o mais rápido possível
	for allowed < 50 {
		if tb.Allow() {
			allowed++
		} else {
			time.Sleep(10 * time.Millisecond) // Pequeno sleep para não criar busy loop
		}
	}
	
	elapsed := time.Since(start)
	
	// 50 tokens a 100/segundo deveria levar ~500ms
	// Damos margem de erro (300ms a 700ms)
	if elapsed < 300*time.Millisecond || elapsed > 700*time.Millisecond {
		t.Errorf("expected ~500ms to get 50 tokens, took %v", elapsed)
	}
}

// IMPORTANTE: Rode com go test -race
func TestTokenBucket_RaceConditions(t *testing.T) {
	tb := NewTokenBucket(50, 25, 50*time.Millisecond)
	
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				tb.Allow()
				time.Sleep(5 * time.Millisecond)
			}
		}()
	}
	
	wg.Wait()
}
```

## 6. Worker Pool com Priority Quieu

Você vai implementar um sistema de processamento de jobs com priorização thread-safe. Pensa em Sidekiq, Celery, ou qualquer background job processor que toda startup usa.

Jobs não são todos iguais. Alguns são críticos (email de recuperação de senha), outros podem esperar (gerar relatório mensal). Você precisa garantir que jobs de alta prioridade sejam processados primeiro, mesmo que cheguem depois.

### Por Que Este Challenge Importa

Todo backend de verdade tem um job queue. Quando usuário faz "esqueci minha senha" no GitHub, esse job não pode esperar atrás de 10 mil jobs de "gerar relatório de commits". Priorização é crítica.

Empresas que usam: Sidekiq (Ruby on Rails), Celery (Python/Django), Bull (Node.js), literalmente toda empresa que você está mirando (GitLab, Render, Tailscale). Em entrevistas, "design a task scheduler" ou "implement a job queue with priorities" são perguntas clássicas.

O que você vai construir:

Um worker pool onde múltiplos workers processam jobs de uma priority queue compartilhada. Jobs de maior prioridade (menor número) são processados primeiro. Múltiplos producers adicionando jobs simultaneamente, múltiplos workers consumindo. Tudo thread-safe.
Os Desafios Principais
1. Heap + Concorrência
container/heap do Go não é thread-safe. Se múltiplas goroutines chamarem Push/Pop simultaneamente, a heap corrompe. Você precisa envolver com mutex, mas onde? Lock muito amplo mata performance. Lock insuficiente causa race conditions.
2. Workers Bloqueando Eficientemente
Workers precisam bloquear quando queue está vazia, mas acordar imediatamente quando job chegar ou shutdown acontecer. Não pode ficar em busy loop (desperdiça CPU). Não pode usar polling com sleep (adiciona latência). A solução é sync.Cond.
3. Backpressure
O que acontece quando jobs chegam mais rápido que workers conseguem processar? Queue cresce infinitamente até matar o processo por falta de memória. Você precisa decidir: bloquear producers (backpressure), dropar jobs (perder dados), ou deixar unbounded (risco de OOM).
Background Técnico
Min Heap para Priority Queue
Uma heap é árvore binária onde cada nó é menor (ou maior) que seus filhos. container/heap do Go implementa min-heap: menor elemento sempre no topo.
Para priority queue onde menor priority number = maior prioridade, min-heap é perfeito. Job com priority 0 (Critical) vem antes de priority 3 (Low).
Interface heap.Interface exige:

Len(), Less(), Swap(): para ordenação
Push(x), Pop(): para adicionar/remover

IMPORTANTE: Você sempre chama heap.Push(h, item) (função do package), nunca h.Push(item) diretamente. O package gerencia invariantes da heap.
Complexidade: Push e Pop são O(log n). Muito mais eficiente que manter slice ordenado (O(n) para inserção).
sync.Cond para Coordenação
Você já usou sync.Cond no TCP Chat Server. É exatamente o mesmo pattern aqui.
Workers chamam cond.Wait() quando queue vazia (bloqueia). Quando Enqueue adiciona job, chama cond.Signal() (acorda um worker). Quando Shutdown, chama cond.Broadcast() (acorda todos).
Regra de ouro: cond.Wait() SEMPRE dentro de um loop que verifica condição, SEMPRE com lock do mutex associado.
Backpressure Strategies
Unbounded Queue: Queue cresce infinitamente. Nunca perde jobs, mas pode matar processo por falta de memória.
Bounded + Bloqueio: Queue tem tamanho máximo. Quando cheia, Enqueue bloqueia até ter espaço. Aplica backpressure nos producers.
Bounded + Drop: Queue tem máximo. Quando cheia, Enqueue retorna erro e incrementa contador de dropped. Producer decide o que fazer.
Não tem resposta certa universal. Sidekiq usa bounded + bloqueio. Kafka deixa producer escolher. Para este challenge, vamos fazer bounded + drop (mais simples).
Libs Go Relevantes
Precisa:

container/heap: implementação de heap
sync.Mutex: proteger acesso à heap
sync.Cond: coordenar workers
sync.WaitGroup: esperar workers em shutdown

Trade-offs:
Precisão vs Performance: Lock em toda operação garante correção mas pode criar contenção. Para este challenge, prefira correção (use mutex).
Drain vs Cancel em Shutdown: Quando shutdown acontece, você drena queue (processa jobs restantes) ou cancela tudo? Vamos drenar, mas só jobs já enfileirados.
Como Começar
Passo 1: Implementar JobHeap (15 min)
Comece com a heap isolada, sem concorrência. Implemente os 5 métodos de heap.Interface. Rode teste básico até passar.
Dica: em Less(), menor priority number vem primeiro. Se empate, pode desempatar por ID.
Passo 2: PriorityQueue Thread-Safe (20 min)
Envolva heap com mutex e cond. Campos: heap, mutex, cond, shutdown flag, métricas.
Dequeue() é o mais interessante: loop com cond.Wait() enquanto queue vazia E não está em shutdown. Quando sair do loop, verificar se é porque tem job ou porque shutdown aconteceu.
Passo 3: WorkerPool (15 min)
Com queue funcionando, pool é simples. Cada worker é loop: Dequeue -> Processa -> Repete. Use WaitGroup para coordenar shutdown.
Passo 4: Testar com Race Detector (10 min)
Rode go test -race -v. O teste TestPriorityOrdering é crítico: verifica que jobs são processados por prioridade. Se passar com -race, está correto.

Decisões de Implementação
1. Bounded ou Unbounded?
Comece bounded. É mais seguro e força você a pensar em backpressure. maxSize como parâmetro do construtor.
2. Como Acordar Workers?
Em Enqueue(): depois de Push, chame cond.Signal() (acorda um worker).
Em Shutdown(): depois de setar flag, chame cond.Broadcast() (acorda todos).
3. Métricas?
Adicione campos int para enqueued, processed, dropped. Útil para debugging e produção real.

```go
package main

import (
	"container/heap"
	"fmt"
	"sync"
	"time"
)

/*
TODO - PASSOS PARA IMPLEMENTAR WORKER POOL COM PRIORITY QUEUE

1. IMPLEMENTAR JobHeap COM heap.Interface
   - Len(), Less(), Swap() para ordenação
   - Push() e Pop() para container/heap
   - Menor priority number = maior prioridade (vem primeiro)

2. CRIAR PriorityQueue THREAD-SAFE
   - Envolver JobHeap com mutex
   - Usar sync.Cond para acordar workers quando job chegar
   - Implementar Enqueue (adiciona job, sinaliza workers)
   - Implementar Dequeue (bloqueia se vazio, acorda em shutdown)

3. IMPLEMENTAR WorkerPool
   - Criar N workers em goroutines
   - Cada worker loop: Dequeue -> Processa -> Repete
   - WaitGroup para coordenar shutdown
   
4. GRACEFUL SHUTDOWN
   - Parar de aceitar novos jobs
   - Broadcast para acordar todos os workers bloqueados
   - Esperar workers terminarem jobs atuais
*/
func main() {
	fmt.Println("=== Worker Pool com Priority Queue ===\n")

	processor := func(job *Job) error {
		priorityName := map[int]string{
			PriorityCritical: "CRITICAL",
			PriorityHigh:     "HIGH",
			PriorityNormal:   "NORMAL",
			PriorityLow:      "LOW",
		}
		fmt.Printf("[Worker] Processing job %d [%s]: %s\n",
			job.ID, priorityName[job.Priority], job.Payload)
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	pool := NewWorkerPool(3, 10, processor)
	pool.Start()

	jobs := []*Job{
		{ID: 1, Priority: PriorityNormal, Payload: "Send newsletter"},
		{ID: 2, Priority: PriorityCritical, Payload: "Password reset email"},
		{ID: 3, Priority: PriorityLow, Payload: "Cleanup old logs"},
		{ID: 4, Priority: PriorityHigh, Payload: "Welcome email"},
		{ID: 5, Priority: PriorityCritical, Payload: "Payment confirmation"},
		{ID: 6, Priority: PriorityNormal, Payload: "Weekly report"},
		{ID: 7, Priority: PriorityLow, Payload: "Aggregate metrics"},
		{ID: 8, Priority: PriorityHigh, Payload: "Push notification"},
	}

	fmt.Println("Submitting jobs...")
	for _, job := range jobs {
		if err := pool.Submit(job); err != nil {
			fmt.Printf("Failed to submit job %d: %v\n", job.ID, err)
		}
	}

	time.Sleep(2 * time.Second)

	enq, proc, drop, qs := pool.Stats()
	fmt.Printf("\n=== Stats ===\n")
	fmt.Printf("Enqueued: %d, Processed: %d, Dropped: %d, Queue Size: %d\n", enq, proc, drop, qs)

	fmt.Println("\nShutting down...")
	pool.Shutdown()
	fmt.Println("All workers stopped. Done!")
}

const (
	PriorityCritical = 0 // Password reset, payment confirmation
	PriorityHigh     = 1 // Welcome emails, notifications
	PriorityNormal   = 2 // Newsletter, analytics
	PriorityLow      = 3 // Cleanup tasks, logs aggregation
)

type Job struct {
	ID       int
	Priority int
	Payload  string
}

type JobHeap []*Job

func (h JobHeap) Len() int { return len(h) }

func (h JobHeap) Less(i, j int) bool {
	return h[i].Priority < h[j].Priority
}

func (h JobHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *JobHeap) Push(x any) {
	*h = append(*h, x.(*Job))
}

func (h *JobHeap) Pop() any {
	old := *h
	n := len(old)
	job := old[n-1]
	*h = old[:n-1]
	return job
}

type PriorityQueue struct {
	heap           *JobHeap
	maxSize        int
	mu             sync.Mutex
	cond           *sync.Cond
	isShuttingDown bool
	enqueued       int
	processed      int
	dropped        int
}

func NewPriorityQueue(maxSize int) *PriorityQueue {
	pq := &PriorityQueue{
		heap:    &JobHeap{},
		maxSize: maxSize,
	}
	heap.Init(pq.heap)
	pq.cond = sync.NewCond(&pq.mu)
	return pq
}

func (pq *PriorityQueue) Enqueue(job *Job) error {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.isShuttingDown {
		return fmt.Errorf("shutting down")
	}

	if pq.maxSize > 0 && pq.heap.Len() >= pq.maxSize {
		pq.dropped++
		return fmt.Errorf("queue full")
	}

	heap.Push(pq.heap, job)
	pq.enqueued++
	pq.cond.Signal()
	return nil
}

func (pq *PriorityQueue) Dequeue() (*Job, bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	for pq.heap.Len() == 0 && !pq.isShuttingDown {
		pq.cond.Wait()
	}

	if pq.isShuttingDown && pq.heap.Len() == 0 {
		return nil, false
	}

	job := heap.Pop(pq.heap).(*Job)
	pq.processed++
	return job, true
}

func (pq *PriorityQueue) Shutdown() {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.isShuttingDown = true
	pq.cond.Broadcast()
}

func (pq *PriorityQueue) Stats() (enqueued, processed, dropped, queueSize int) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return pq.enqueued, pq.processed, pq.dropped, pq.heap.Len()
}

type WorkerPool struct {
	queue       *PriorityQueue
	numWorkers  int
	processFunc func(*Job) error
	wg          sync.WaitGroup
}

func NewWorkerPool(numWorkers, queueSize int, processor func(*Job) error) *WorkerPool {
	return &WorkerPool{
		queue:       NewPriorityQueue(queueSize),
		numWorkers:  numWorkers,
		processFunc: processor,
	}
}

func (wp *WorkerPool) Start() {
	for i := 0; i < wp.numWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()
	for {
		job, ok := wp.queue.Dequeue()
		if !ok {
			return
		}
		if err := wp.processFunc(job); err != nil {
			fmt.Printf("Worker %d error processing job %d: %v\n", id, job.ID, err)
		}
	}
}

func (wp *WorkerPool) Submit(job *Job) error {
	return wp.queue.Enqueue(job)
}

func (wp *WorkerPool) Shutdown() {
	wp.queue.Shutdown()
	wp.wg.Wait()
}

func (wp *WorkerPool) Stats() (enqueued, processed, dropped, queueSize int) {
	return wp.queue.Stats()
}
```
testes:

```go
package main

import (
	"container/heap"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestJobHeapBasic testa operações básicas da heap
func TestJobHeapBasic(t *testing.T) {
	h := &JobHeap{}
	heap.Init(h)

	jobs := []*Job{
		{ID: 1, Priority: PriorityNormal},
		{ID: 2, Priority: PriorityCritical},
		{ID: 3, Priority: PriorityLow},
		{ID: 4, Priority: PriorityHigh},
	}

	// Push jobs
	for _, job := range jobs {
		heap.Push(h, job)
	}

	// Pop deve retornar em ordem de prioridade
	expected := []int{PriorityCritical, PriorityHigh, PriorityNormal, PriorityLow}
	for i, exp := range expected {
		job := heap.Pop(h).(*Job)
		if job.Priority != exp {
			t.Errorf("Pop %d: expected priority %d, got %d", i, exp, job.Priority)
		}
	}
}

// TestPriorityQueueThreadSafety testa acesso concorrente
func TestPriorityQueueThreadSafety(t *testing.T) {
	pq := NewPriorityQueue(100)

	var wg sync.WaitGroup
	numProducers := 10
	numConsumers := 5
	jobsPerProducer := 20

	// Producers
	for i := 0; i < numProducers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < jobsPerProducer; j++ {
				job := &Job{
					ID:       id*jobsPerProducer + j,
					Priority: j % 4, // Variar prioridades
					Payload:  "test",
				}
				pq.Enqueue(job)
			}
		}(i)
	}

	// Consumers
	consumed := int32(0)
	for i := 0; i < numConsumers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				job, ok := pq.Dequeue()
				if !ok {
					return
				}
				if job != nil {
					atomic.AddInt32(&consumed, 1)
				}
			}
		}()
	}

	// Esperar producers terminarem
	time.Sleep(100 * time.Millisecond)

	// Shutdown
	pq.Shutdown()
	wg.Wait()

	expected := int32(numProducers * jobsPerProducer)
	if consumed != expected {
		t.Errorf("Expected %d jobs consumed, got %d", expected, consumed)
	}
}

// TestBoundedQueue testa comportamento quando queue enche
func TestBoundedQueue(t *testing.T) {
	maxSize := 5
	pq := NewPriorityQueue(maxSize)

	// Encher queue
	for i := 0; i < maxSize; i++ {
		err := pq.Enqueue(&Job{ID: i, Priority: PriorityNormal})
		if err != nil {
			t.Fatalf("Failed to enqueue job %d: %v", i, err)
		}
	}

	// Próximo enqueue deve falhar ou bloquear dependendo da implementação
	err := pq.Enqueue(&Job{ID: 999, Priority: PriorityCritical})
	if err == nil && pq.Len() > maxSize {
		t.Error("Queue exceeded max size without error")
	}
}

// TestWorkerPoolProcessing testa que workers processam jobs
func TestWorkerPoolProcessing(t *testing.T) {
	processed := int32(0)
	processor := func(job *Job) error {
		atomic.AddInt32(&processed, 1)
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	pool := NewWorkerPool(3, 20, processor)
	pool.Start()

	// Submit jobs
	numJobs := 15
	for i := 0; i < numJobs; i++ {
		pool.Submit(&Job{
			ID:       i,
			Priority: i % 4,
			Payload:  "test",
		})
	}

	// Esperar processar
	time.Sleep(500 * time.Millisecond)
	pool.Shutdown()

	if processed != int32(numJobs) {
		t.Errorf("Expected %d jobs processed, got %d", numJobs, processed)
	}
}

// TestPriorityOrdering testa que jobs de alta prioridade são processados primeiro
func TestPriorityOrdering(t *testing.T) {
	var mu sync.Mutex
	var processOrder []int

	processor := func(job *Job) error {
		mu.Lock()
		processOrder = append(processOrder, job.ID)
		mu.Unlock()
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	pool := NewWorkerPool(1, 10, processor) // 1 worker para ordem determinística
	pool.Start()

	// Submit em ordem aleatória mas IDs baixos = prioridade alta
	jobs := []*Job{
		{ID: 3, Priority: PriorityLow},
		{ID: 1, Priority: PriorityCritical},
		{ID: 4, Priority: PriorityLow},
		{ID: 2, Priority: PriorityHigh},
	}

	for _, job := range jobs {
		pool.Submit(job)
	}

	time.Sleep(500 * time.Millisecond)
	pool.Shutdown()

	// Verificar que processou em ordem de prioridade
	if len(processOrder) != 4 {
		t.Fatalf("Expected 4 jobs processed, got %d", len(processOrder))
	}

	// ID 1 (Critical) deve vir antes de ID 2 (High)
	// ID 2 (High) deve vir antes de IDs 3,4 (Low)
	if processOrder[0] != 1 || processOrder[1] != 2 {
		t.Errorf("Wrong processing order: %v", processOrder)
	}
}

```


---

## 7. LRU Cache Thread-Safe com TTL

Você vai implementar um cache completo de produção com política de eviction LRU (Least Recently Used) e expiração automática por TTL (Time To Live). Este é o tipo de sistema que está rodando em todo lugar.

Cache com eviction está em tudo. Redis usa LRU. Memcached usa LRU. Browsers usam LRU para cache de assets. CDNs usam variações de LRU. Quando você acessa reddit.com, tem múltiplas camadas de cache usando políticas parecidas.

## Por Que Este Challenge Importa

Todo sistema de verdade usa cache. API lenta? Adiciona cache. Database sobrecarregado? Cache. Latência internacional alta? CDN com cache. LRU é a política de eviction mais comum e mais perguntada em entrevistas.

**Empresas que usam:** Redis (open source e cloud), Memcached, CloudFlare (CDN), praticamente toda API de grande empresa em produção. Em entrevistas FAANG e tier 2/3, "implement an LRU cache" é pergunta clássica. Adicionar TTL e thread safety te eleva para nível senior.

**O que você vai construir:**

Um cache que mantém até N items em memória. Quando você acessa um item (Get), ele se torna o "mais recentemente usado". Quando o cache enche e você adiciona novo item, o "menos recentemente usado" é removido (evicted). Items também expiram automaticamente após TTL.

Múltiplas goroutines vão ler e escrever simultaneamente. Você precisa garantir thread safety sem matar performance. Uma goroutine em background vai limpar items expirados periodicamente.

## Os Desafios Principais

**1. Estrutura de Dados Eficiente**

Precisa de O(1) para Get, Set, e eviction. Isso exige **HashMap + Doubly Linked List**. HashMap dá O(1) lookup. Linked list dá O(1) para mover items e remover do final. Slice não funciona porque remover do meio é O(n).

Você precisa entender ponteiros. Cada node tem prev e next. Quando move item para frente, precisa atualizar 4 ponteiros (prev.next, next.prev, node.prev, node.next). Um erro aqui e você vaza memória ou corrompe a lista.

**2. Concorrência Sem Matar Performance**

Solução ingênua: mutex global. Todo Get bloqueia todo Set. Com 100 goroutines lendo, apenas uma processa por vez. Performance horrível.

Solução melhor: RWMutex. Múltiplas leituras simultâneas, apenas writes bloqueiam tudo. Mas Get também escreve (move node na lista), então ainda tem contenção.

Solução avançada (se sobrar tempo): sharding. Múltiplos caches menores, cada um com próprio lock. Hash da key determina qual shard. Reduz contenção drasticamente.

**3. TTL Cleanup Coordenado**

Items expiram após TTL. Você não pode verificar em todo Get se expirou (adiciona latência). Precisa de goroutine em background que periodicamente varre e remove expirados.

Mas cleanup goroutine precisa coordenar com Gets/Sets. Se cleanup está varrendo lista e alguém chama Set, pode corromper. Precisa do mesmo lock. Como fazer cleanup não travar cache inteiro por muito tempo?

## Background Técnico

### HashMap + Doubly Linked List

```
HashMap: key -> *Node  (O(1) lookup)

Doubly Linked List: (ordem de uso, mais recente na frente)
head -> [Node A] <-> [Node B] <-> [Node C] <- tail
         (newest)                  (oldest/LRU)
```

#### Get "B":

1. Lookup no map: O(1)
2. Move B para head: O(1) (atualizar ponteiros)
3. Retorna value

#### Set novo item quando cheio:

1. Remove tail (LRU): O(1)
2. Delete do map: O(1)
3. Cria novo node na head: O(1)
4. Adiciona no map: O(1)

Tudo O(1). Por isso HashMap + Linked List e não outras estruturas.

#### Por Que Doubly Linked List?

Singly linked list não funciona porque para remover node, você precisa do anterior (para atualizar seu next). Com doubly, cada node sabe seu prev, então remoção é O(1).

#### TTL e Cleanup

Opção 1 - Lazy expiration: Só verifica TTL em Get. Simples mas items expirados ocupam memória até serem acessados.
Opção 2 - Active expiration: Goroutine periodicamente varre e remove expirados. Mais complexo mas libera memória.

Vamos fazer opção 2. Ticker rodando a cada X segundos, varre lista do final (items mais antigos), remove expirados, para quando encontrar item válido (otimização: lista está ordenada por tempo de acesso).

#### Thread Safety com RWMutex

```go
type LRUCache struct {
    mu sync.RWMutex  // Read-Write mutex
}

func (c *LRUCache) Get(key string) (any, bool) {
    c.mu.Lock()    // Precisa write lock (move node)
    defer c.mu.Unlock()
    // ...
}

func (c *LRUCache) Size() int {
    c.mu.RLock()   // Read lock suficiente
    defer c.mu.RUnlock()
    return len(c.items)
}
```

### Trade-offs Principais

#### Mutex vs RWMutex vs Sharding

Mutex é simples, funciona, mas serializa tudo.
RWMutex: permite múltiplas leituras mas Get ainda precisa write lock.
Sharding: melhor performance mas é mais complexo.

Para este challenge use Mutex, que é mais simples.

Se a performance for um problema, o sharding pode ser um pŕoximo passo.

#### Lazy vs Active TTL Cleanup

Lazy: verifica o TTL e limpa durante o Get, é simples.
Active: goroutine em background, mais eficiente em memória pois evita itens muito antigos.

Para esse challenge, use o Active prioritariamente, e também se conseguir o Lazy, obtendo uma solução híbrida.

#### Frequência de cleanup

Se a frequencia for alta, vai ter overhead de lock, Se pouco frequente, os itens expirados ocupam mais memória do que deveria.

Use um intervalo de 100ms.

### Passos

**Passo 1**: Doubly Linked List (30 min)

Implemente a lista primeiro, sem cache. Struct Node com key, value, timestamp, prev, next. Métodos: addToFront, remove, moveToFront. Teste que ponteiros estão corretos.

Dica: desenhe no papel. Quando move node do meio para frente, você precisa:

Conectar prev e next entre si (pular o node)
Colocar node como novo head
Atualizar head.prev para apontar para node

**Passo 2**: Cache Básico sem TTL (30 min)

Adicione HashMap + Linked List. Implemente Get e Set com eviction LRU. Ignore TTL por enquanto. Adicione mutex para thread safety.

**Passo 3**: Adicionar TTL (20 min)

Adicione campo timestamp em cada node. Em Get, verifique se expirou. Se expirou, remova e retorne miss. Em Set, sempre atualize timestamp para time.Now().

**Passo 4**: Cleanup Goroutine (20 min)

Inicie goroutine em NewLRUCache. Use time.NewTicker. A cada tick, varra lista do tail (mais antigos primeiro). Remova items expirados. Para quando encontrar item válido (otimização).

**Passo 5**: Graceful Shutdown (10 min)

Adicione Close() que sinaliza cleanup goroutine para parar. Use channel ou context.

**Passo 6**: Testes com Race Detector (10 min)

Rode go test -race -v. Teste TestLRUCache_Concurrent e TestLRUCache_Race são críticos.

```go
package main

import (
	"fmt"
	"sync"
	"time"
)

/*
TODO - PASSOS PARA IMPLEMENTAR LRU CACHE COM TTL

1. CRIAR A DOUBLY LINKED LIST
   - Node com ponteiros para prev e next
   - Métodos: addToFront, remove, moveToFront
   - Manter ponteiros head e tail atualizados

2. CRIAR O CACHE
   - HashMap (map) para O(1) lookup: key -> node
   - Linked list para rastrear ordem de uso
   - Fields: capacity, size, mutex

3. IMPLEMENTAR Get()
   - Verificar se key existe no map
   - Verificar se expirou (comparar timestamp com TTL)
   - Se válido: mover node para frente da lista, retornar valor
   - Se expirado: remover do cache, retornar miss

4. IMPLEMENTAR Set()
   - Se key já existe: atualizar valor, mover para frente, atualizar timestamp
   - Se não existe: criar node, adicionar no map e na frente da lista
   - Se cache cheio: remover node do final da lista antes de adicionar novo
   - Sempre atualizar timestamp com time.Now()

5. IMPLEMENTAR CLEANUP AUTOMÁTICO
   - Goroutine em background roda periodicamente
   - Varre lista do final (itens mais antigos)
   - Remove itens expirados
   - Para quando encontrar item não expirado (otimização)

6. GRACEFUL SHUTDOWN
   - Sinalizar cleanup goroutine para parar
   - Esperar goroutine terminar
*/

func main() {
	fmt.Println("=== LRU Cache com TTL ===")

	// Cache: max 5 items, TTL de 2 segundos
	cache := NewLRUCache(5, 2*time.Second)
	defer cache.Close()

	// Adicionar alguns items
	fmt.Println("Adding items...")
	cache.Set("user:1", "Alice")
	cache.Set("user:2", "Bob")
	cache.Set("user:3", "Charlie")
	cache.Set("user:4", "Diana")
	cache.Set("user:5", "Eve")

	// Tentar get
	if val, ok := cache.Get("user:1"); ok {
		fmt.Printf("Got user:1 = %v\n", val)
	}

	// Adicionar 6º item (deve evict LRU)
	fmt.Println("\nAdding 6th item (cache full, should evict LRU)...")
	cache.Set("user:6", "Frank")

	// user:2 deve ter sido evicted (era o LRU, user:1 foi acessado)
	if _, ok := cache.Get("user:2"); !ok {
		fmt.Println("user:2 was evicted (LRU)")
	}

	// Testar TTL
	fmt.Println("\nWaiting for TTL expiration (2.5s)...")
	time.Sleep(2500 * time.Millisecond)

	// Todos os items devem ter expirado
	if _, ok := cache.Get("user:1"); !ok {
		fmt.Println("user:1 expired (TTL)")
	}

	// Adicionar novos items
	cache.Set("user:7", "Grace")
	cache.Set("user:8", "Henry")

	fmt.Printf("\nFinal cache size: %d\n", cache.Size())
	fmt.Println("Done!")
}

// CacheItem representa um item no cache
type CacheItem struct {
	key       string
	value     any
	timestamp time.Time
	// TODO: adicionar ponteiros prev e next para doubly linked list
}

// LRUCache é um cache thread-safe com política LRU e TTL
type LRUCache struct {
	// TODO: adicionar campos necessários
	// - capacity (max items)
	// - map[string]*CacheItem para O(1) lookup
	// - head e tail da linked list
	// - mutex para thread safety
	// - ttl (time to live)
	// - channel para shutdown do cleanup
}

// NewLRUCache cria novo cache
// capacity: número máximo de items
// ttl: quanto tempo items vivem antes de expirar
func NewLRUCache(capacity int, ttl time.Duration) *LRUCache {
	// TODO: implementar
	// Inicializar map, linked list vazia (head/tail nil)
	// Iniciar goroutine de cleanup
	return nil
}

// Get busca valor por key
// Retorna (value, true) se encontrou e não expirou
// Retorna (nil, false) se não encontrou ou expirou
func (c *LRUCache) Get(key string) (any, bool) {
	// TODO: implementar
	// 1. Lock mutex
	// 2. Verificar se key existe no map
	// 3. Verificar se expirou (time.Since(timestamp) > ttl)
	// 4. Se expirou: remover do cache, retornar false
	// 5. Se válido: mover para frente da lista, retornar value
	// 6. Unlock mutex
	return nil, false
}

// Set adiciona ou atualiza item no cache
func (c *LRUCache) Set(key string, value any) {
	// TODO: implementar
	// 1. Lock mutex
	// 2. Se key já existe: atualizar valor, mover para frente, atualizar timestamp
	// 3. Se não existe:
	//    a. Se cache cheio: remover item do final (LRU)
	//    b. Criar novo node
	//    c. Adicionar no map e na frente da lista
	//    d. Atualizar timestamp
	// 4. Unlock mutex
}

// Delete remove item do cache
func (c *LRUCache) Delete(key string) {
	// TODO: implementar
	// Lock, remover do map e da lista, unlock
}

// Size retorna número de items no cache
func (c *LRUCache) Size() int {
	// TODO: implementar com lock
	return 0
}

// addToFront adiciona node no início da lista
func (c *LRUCache) addToFront(node *CacheItem) {
	// TODO: implementar
	// Atualizar ponteiros head/tail
}

// remove retira node da lista
func (c *LRUCache) remove(node *CacheItem) {
	// TODO: implementar
	// Atualizar ponteiros dos vizinhos e head/tail se necessário
}

// moveToFront move node existente para início
func (c *LRUCache) moveToFront(node *CacheItem) {
	// TODO: implementar
	// Remover da posição atual e adicionar na frente
}

// removeLRU remove item menos recentemente usado (final da lista)
func (c *LRUCache) removeLRU() {
	// TODO: implementar
	// Remover tail, atualizar map
}

// cleanup goroutine que remove items expirados periodicamente
func (c *LRUCache) cleanup() {
	// TODO: implementar
	// ticker := time.NewTicker(interval)
	// Loop: a cada tick, varrer lista do final
	// Remover items expirados
	// Parar quando receber sinal de shutdown
}

// Close para cleanup goroutine
func (c *LRUCache) Close() {
	// TODO: implementar
	// Enviar sinal para shutdown channel
}
```

Testes:

```go
package main

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLRUCache_BasicOperations(t *testing.T) {
	cache := NewLRUCache(3, 10*time.Second)
	defer cache.Close()

	cache.Set("a", 1)
	cache.Set("b", 2)
	cache.Set("c", 3)

	if val, ok := cache.Get("a"); !ok || val != 1 {
		t.Errorf("expected a=1, got %v, %v", val, ok)
	}

	if cache.Size() != 3 {
		t.Errorf("expected size 3, got %d", cache.Size())
	}
}

func TestLRUCache_Eviction(t *testing.T) {
	cache := NewLRUCache(3, 10*time.Second)
	defer cache.Close()

	cache.Set("a", 1)
	cache.Set("b", 2)
	cache.Set("c", 3)

	// a é o LRU
	// Adicionar d deve evict a
	cache.Set("d", 4)

	if _, ok := cache.Get("a"); ok {
		t.Error("a should have been evicted")
	}

	if cache.Size() != 3 {
		t.Errorf("expected size 3, got %d", cache.Size())
	}
}

func TestLRUCache_LRUOrdering(t *testing.T) {
	cache := NewLRUCache(3, 10*time.Second)
	defer cache.Close()

	cache.Set("a", 1)
	cache.Set("b", 2)
	cache.Set("c", 3)

	// Acessar a (move para frente)
	cache.Get("a")

	// Agora b é o LRU
	cache.Set("d", 4)

	// b deve ter sido evicted
	if _, ok := cache.Get("b"); ok {
		t.Error("b should have been evicted")
	}

	// a ainda deve existir
	if _, ok := cache.Get("a"); !ok {
		t.Error("a should still exist")
	}
}

func TestLRUCache_Update(t *testing.T) {
	cache := NewLRUCache(3, 10*time.Second)
	defer cache.Close()

	cache.Set("a", 1)
	cache.Set("a", 2) // update

	if val, ok := cache.Get("a"); !ok || val != 2 {
		t.Errorf("expected a=2 after update, got %v", val)
	}

	if cache.Size() != 1 {
		t.Errorf("expected size 1, got %d", cache.Size())
	}
}

func TestLRUCache_TTL(t *testing.T) {
	cache := NewLRUCache(10, 100*time.Millisecond)
	defer cache.Close()

	cache.Set("a", 1)

	// Dentro do TTL
	if _, ok := cache.Get("a"); !ok {
		t.Error("a should exist within TTL")
	}

	// Esperar expirar
	time.Sleep(150 * time.Millisecond)

	// Deve ter expirado
	if _, ok := cache.Get("a"); ok {
		t.Error("a should have expired")
	}
}

func TestLRUCache_TTLRefresh(t *testing.T) {
	cache := NewLRUCache(10, 100*time.Millisecond)
	defer cache.Close()

	cache.Set("a", 1)
	time.Sleep(60 * time.Millisecond)

	// Atualizar (refresh TTL)
	cache.Set("a", 2)
	time.Sleep(60 * time.Millisecond)

	// Não deve ter expirado (TTL foi refreshed)
	if val, ok := cache.Get("a"); !ok || val != 2 {
		t.Error("a should still exist after TTL refresh")
	}
}

func TestLRUCache_Concurrent(t *testing.T) {
	cache := NewLRUCache(100, 5*time.Second)
	defer cache.Close()

	var wg sync.WaitGroup
	numGoroutines := 50

	// Writers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := fmt.Sprintf("key:%d:%d", id, j)
				cache.Set(key, j)
			}
		}(i)
	}

	// Readers
	var hits, misses atomic.Int32
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := fmt.Sprintf("key:%d:%d", id, j)
				if _, ok := cache.Get(key); ok {
					hits.Add(1)
				} else {
					misses.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()

	totalOps := int(hits.Load()) + int(misses.Load())
	if totalOps != numGoroutines*100 {
		t.Errorf("expected %d total ops, got %d", numGoroutines*100, totalOps)
	}
}

func TestLRUCache_CleanupExpired(t *testing.T) {
	cache := NewLRUCache(10, 50*time.Millisecond)
	defer cache.Close()

	// Adicionar items
	for i := 0; i < 5; i++ {
		cache.Set(fmt.Sprintf("key:%d", i), i)
	}

	if cache.Size() != 5 {
		t.Errorf("expected size 5, got %d", cache.Size())
	}

	// Esperar cleanup remover items expirados
	time.Sleep(200 * time.Millisecond)

	// Cache deve estar vazio ou quase
	size := cache.Size()
	if size > 1 { // Pode ter 0 ou 1 dependendo de timing
		t.Errorf("expected size ~0 after cleanup, got %d", size)
	}
}

// CRÍTICO: rodar com -race
func TestLRUCache_Race(t *testing.T) {
	cache := NewLRUCache(50, 1*time.Second)
	defer cache.Close()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				key := fmt.Sprintf("k:%d", j)
				cache.Set(key, j)
				cache.Get(key)
				if j%10 == 0 {
					cache.Delete(key)
				}
			}
		}(i)
	}
	wg.Wait()
}
```

## 8 - Health Check Poller com Circuit Breaker

Você vai construir um sistema de health checking distribuído que monitora múltiplos endpoints HTTP ou TCP simultaneamente.

É como o que o Kubernetes Kubelet faz para monitorar as PODs, ou o que load balancer usam para saber se um backend está saudável.

**O que você vai construir:** Um poller que recebe uma lista de endpoints para monitorar, faz health checks concorrentes em intervalos regulares, agrega status de todos os endpoints, e implementa circuit breaker pattern. Se um endpoint falha N vezes seguidas, o circuit abre e você para de fazer requests por um tempo antes de tentar novamente.

Esqueleto do código:

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

/*
TODO - HEALTH CHECK POLLER

1. CRIAR ENDPOINT CONFIG
   - URL do endpoint
   - Interval de polling (quanto tempo entre checks)
   - Timeout do request
   - Threshold de falhas para circuit breaker abrir

2. IMPLEMENTAR HEALTH CHECKER
   - Goroutine por endpoint fazendo polling periódico
   - HTTP GET com timeout configurável
   - Classificar response: healthy (2xx), unhealthy (outros), error (timeout/network)
   
3. CIRCUIT BREAKER
   - Contar falhas consecutivas
   - Abrir circuit após N falhas (para de fazer requests)
   - Após X tempo, tentar novamente (half-open)
   - Se suceder, fechar circuit (volta ao normal)
   
4. AGREGAÇÃO DE STATUS
   - Channel para resultados de health checks
   - Goroutine agregadora mantém mapa de status atual
   - Detectar transições (healthy->unhealthy, unhealthy->healthy)
   - Notificar via callback quando status muda

5. GRACEFUL SHUTDOWN
   - Context para cancelar todas as goroutines de polling
   - Esperar todas terminarem antes de sair
*/

func main() {
	poller := NewHealthPoller()
	
	// Configurar callback para notificações
	poller.onStatusChange = func(endpoint string, oldStatus, newStatus bool) {
		if newStatus && !oldStatus {
			fmt.Printf("- %s is now HEALTHY\n", endpoint)
		} else if !newStatus && oldStatus {
			fmt.Printf("❌ %s is now UNHEALTHY\n", endpoint)
		}
	}
	
	// Adicionar endpoints para monitorar
	poller.AddEndpoint(EndpointConfig{
		URL:              "https://www.google.com",
		PollInterval:     2 * time.Second,
		Timeout:          1 * time.Second,
		FailureThreshold: 3,
		RecoveryTimeout:  10 * time.Second,
	})
	
	poller.AddEndpoint(EndpointConfig{
		URL:              "http://localhost:8080/health", // Vai falhar
		PollInterval:     2 * time.Second,
		Timeout:          500 * time.Millisecond,
		FailureThreshold: 3,
		RecoveryTimeout:  10 * time.Second,
	})
	
	poller.Start()
	
	// Rodar por 30 segundos
	time.Sleep(30 * time.Second)
	
	// Mostrar status final
	fmt.Println("\n=== Final Status ===")
	for endpoint, status := range poller.GetAllStatuses() {
		healthy := "HEALTHY"
		if !status.Healthy {
			healthy = "UNHEALTHY"
		}
		circuit := ""
		if status.CircuitOpen {
			circuit = " [CIRCUIT OPEN]"
		}
		fmt.Printf("%s: %s (fails: %d)%s\n", 
			endpoint, healthy, status.ConsecutiveFails, circuit)
	}
	
	poller.Stop()
	fmt.Println("Done!")
}

// EndpointConfig define configuração de um endpoint monitorado
type EndpointConfig struct {
	URL              string
	PollInterval     time.Duration
	Timeout          time.Duration
	FailureThreshold int // Falhas consecutivas para abrir circuit
	RecoveryTimeout  time.Duration // Quanto tempo esperar antes de retry quando circuit aberto
}

// HealthStatus representa estado de saúde de um endpoint
type HealthStatus struct {
	Endpoint       string
	Healthy        bool
	LastCheckTime  time.Time
	ConsecutiveFails int
	CircuitOpen    bool
	Error          error
}

// HealthPoller gerencia health checks de múltiplos endpoints
type HealthPoller struct {
	endpoints map[string]*EndpointConfig
	statuses  map[string]*HealthStatus
	mu        sync.RWMutex
	
	results   chan HealthStatus
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	
	// Callback chamado quando status de endpoint muda
	onStatusChange func(endpoint string, oldStatus, newStatus bool)
}

// NewHealthPoller cria novo poller
func NewHealthPoller() *HealthPoller {
	ctx, cancel := context.WithCancel(context.Background())
	return &HealthPoller{
		endpoints: make(map[string]*EndpointConfig),
		statuses:  make(map[string]*HealthStatus),
		results:   make(chan HealthStatus, 100),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// AddEndpoint adiciona endpoint para monitorar
func (hp *HealthPoller) AddEndpoint(config EndpointConfig) {
	// TODO: implementar
	// Adicionar no map de endpoints
	// Inicializar status
}

// Start inicia polling de todos os endpoints
func (hp *HealthPoller) Start() {
	// TODO: implementar
	// Iniciar goroutine agregadora (processa results channel)
	// Para cada endpoint, iniciar goroutine de polling
}

// pollEndpoint faz health check periódico de um endpoint
func (hp *HealthPoller) pollEndpoint(config EndpointConfig) {
	// TODO: implementar
	// Loop com ticker no PollInterval
	// A cada tick:
	//   - Se circuit aberto, verificar se pode tentar (RecoveryTimeout passou)
	//   - Fazer HTTP GET com timeout
	//   - Avaliar response (2xx = healthy)
	//   - Atualizar circuit breaker state
	//   - Enviar resultado para results channel
}

// checkEndpoint faz um health check
func (hp *HealthPoller) checkEndpoint(config EndpointConfig) HealthStatus {
	// TODO: implementar
	// Criar HTTP client com timeout
	// GET no endpoint
	// Retornar HealthStatus baseado em response
	return HealthStatus{}
}

// aggregateResults processa resultados e detecta mudanças de status
func (hp *HealthPoller) aggregateResults() {
	// TODO: implementar
	// Loop recebendo do results channel
	// Atualizar statuses map
	// Detectar transições (healthy->unhealthy ou vice-versa)
	// Chamar callback se status mudou
}

// GetStatus retorna status atual de um endpoint
func (hp *HealthPoller) GetStatus(endpoint string) (HealthStatus, bool) {
	// TODO: implementar com read lock
	return HealthStatus{}, false
}

// GetAllStatuses retorna status de todos os endpoints
func (hp *HealthPoller) GetAllStatuses() map[string]HealthStatus {
	// TODO: implementar com read lock
	return nil
}

// Stop para todos os health checks
func (hp *HealthPoller) Stop() {
	// TODO: implementar
	// Cancelar context
	// Esperar WaitGroup
	// Fechar results channel
}
```

### Como começar

1. Implemente primeiro um health check básico em checkEndpoint. Crie um http client com timeout, faça GET, classifique a response como saudável ou Não. Testa isso de forma isolada antes de adicionar polling.

2. Implemente pollEndpoint sem circuit breaker ainda. Apenas ticker fazendo checks em intervalos e enviando os resultados para o channel. Teste em um endpoint que sabe que vai funcionar.

3. Adicione uma goroutine agregadora em aggregateResults.  Ela recebe o channel e atualiza o map de statuses com lock. Adicione prints para ver status mudando.

4. Implemente o circuit breaker em pollEndpoint. Conte falhas consecutivas, abra o circuit breaker quando atingir um threshold específico, e espere RecoveryTimeout antes de tentar novamente

5. Adicione detecção de mudançás de estados na função agregadora e chama o callback quando detectar uma transição.

---

## 9. Trie thread safe

Imagine que você está construindo o autocomplete do GitHub quando você digita nome de repositório. Enquanto você está digitando "react", milhares de outros usuários estão fazendo buscas simultaneamente. O sistema precisa ser thread-safe, mas se você simplesmente colocar um mutex global na trie inteira, vai virar um gargalo massivo, pois só uma pessoa pode buscar por vez.

Uma solução para isso é criar um **locking granular por subtree**. Esse mecanismos faz com que buscas por "react" e buscas por "golang" não compitam pelo mesmo lock, pois estão em partes diferentes da árvore.

HTTP routes como Gin e Echo usam tries para fazer routing URLs, e precisam ser muito rápidos, pois todas requisições passam por lá. DNS servers usam tries aara lookup de domínios. IP routing tables também.

É uma ótima estrutura quando precisamos **achar prefixos de forma rápida com alto throughput concorrente.**

### O que você vai construir

Uma trie thread-safe que suporta:

- **Insert**: adiciona palavras na árvore de forma concorrente
- **Search**: verifica se uma palavra completa existe (exact match)
- **StartsWith**: verifica se existe alguma palavra com determinado prefixo
- **AutoComplete**: retorna todas as palavras que começam com um prefixo (limit de resultados)
- **Delete**: remove palavras da trie

Dessa forma, não podemos ter um locking na árvore inteira, e sim um lock por node. Dessa forma, ao percorrer a árvore, vamos adquirindo e liberando lock a pedida que descemos ela. Isso vai permitir diversas threads percorrendo caminhos diferentes de forma simultânea.

### Background Técnico

Uma Trie é uma árvore onde cada node representa um caractere. Diferente de uma binary search tree onde cada node é uma palavra completa, na trie os caracteres formam palavras ao longo dos caminhos da raiz até as folhas.

Exemplo visual para as palavras "cat", "car", "card", "dog":

```bash
	   root
      /    \
     c      d
     |      |
     a      o
    / \     |
   t   r    g*
   *   |\
       d*
       |
       *
```

* -> marcam fim da palavra.

**Complexidade**

- Insert/Search: O(m) onde m é o tamanho da palavra (não depende de quantas palavras existem)
- Space: O(ALPHABET_SIZE × N × M) no pior caso, mas na prática muito melhor por causa do compartilhamento de prefixos
- AutoComplete: O(p + n) onde p é tamanho do prefix e n é número de palavras com aquele prefix


```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	trie := NewTrie()

	fmt.Println("=== Testando Insert e Search ===")
	words := []string{"cat", "car", "card", "care", "careful", "dog", "dodge", "door"}

	for _, word := range words {
		trie.Insert(word)
		fmt.Printf("Inserted: %s\n", word)
	}

	fmt.Println("\n=== Testando Search (exact match) ===")
	testWords := []string{"car", "card", "careful", "cat", "can", "do"}
	for _, word := range testWords {
		found := trie.Search(word)
		fmt.Printf("Search('%s'): %v\n", word, found)
	}

	fmt.Println("\n=== Testando StartsWith ===")
	prefixes := []string{"ca", "car", "do", "doo", "cat", "x"}
	for _, prefix := range prefixes {
		exists := trie.StartsWith(prefix)
		fmt.Printf("StartsWith('%s'): %v\n", prefix, exists)
	}

	fmt.Println("\n=== Testando AutoComplete ===")
	testPrefixes := []string{"ca", "car", "do"}
	for _, prefix := range testPrefixes {
		suggestions := trie.AutoComplete(prefix, 5)
		fmt.Printf("AutoComplete('%s', limit=5): %v\n", prefix, suggestions)
	}

	fmt.Println("\n=== Testando AutoComplete sem limite ===")
	allCar := trie.AutoComplete("car", 0)
	fmt.Printf("AutoComplete('car', no limit): %v\n", allCar)

	fmt.Println("\n=== Testando operações concorrentes ===")
	var wg sync.WaitGroup

	// 10 goroutines inserindo palavras simultaneamente
	newWords := []string{
		"apple", "application", "apply", "banana", "band", "bandana",
		"can", "candy", "candle", "canon",
	}

	for _, word := range newWords {
		wg.Add(1)
		go func(w string) {
			defer wg.Done()
			trie.Insert(w)
		}(word)
	}

	// 10 goroutines fazendo searches simultaneamente
	for _, word := range testWords {
		wg.Add(1)
		go func(w string) {
			defer wg.Done()
			trie.Search(w)
		}(word)
	}

	// 10 goroutines fazendo autocomplete simultaneamente
	for _, prefix := range testPrefixes {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			trie.AutoComplete(p, 3)
		}(prefix)
	}

	wg.Wait()

	fmt.Println("\n=== Verificando palavras inseridas concorrentemente ===")
	for _, word := range newWords {
		found := trie.Search(word)
		fmt.Printf("Search('%s'): %v\n", word, found)
	}

	fmt.Println("\n=== Testando AutoComplete em 'can' ===")
	canWords := trie.AutoComplete("can", 10)
	fmt.Printf("AutoComplete('can'): %v\n", canWords)
}

// TrieNode representa um node na trie
// Cada node tem um map de children (um por caractere possível)
// e uma flag indicando se é fim de palavra
type TrieNode struct {
	children map[rune]*TrieNode
	isWord    bool
	mu       sync.RWMutex // Lock granular por node
}

// Trie é a estrutura principal
type Trie struct {
	root *TrieNode
}

// NewTrie cria uma nova trie vazia
func NewTrie() *Trie {
	return &Trie{
		root: &TrieNode{
			children: make(map[rune]*TrieNode),
		},
	}
}

// Insert adiciona uma palavra na trie
// Thread-safe: múltiplas goroutines podem inserir simultaneamente
// TODO: Implementar usando hand-over-hand locking
// Dica: você precisa adquirir write lock do node atual, depois do child,
// depois liberar o lock do node pai antes de continuar descendo
func (t *Trie) Insert(word string) {
	// TODO: Implementar
	// Lembre-se: ao criar um novo child node, você precisa inicializar
	// o map de children dele também (make)
}

// Search verifica se uma palavra completa existe na trie
// Retorna true apenas se a palavra existe E está marcada como fim de palavra
// TODO: Implementar usando read locks
// Dica: você pode usar RLock aqui porque não está modificando
func (t *Trie) Search(word string) bool {
	// TODO: Implementar
	return false
}

// StartsWith verifica se existe alguma palavra na trie que começa com o prefix dado
// TODO: Implementar
// Dica: diferente do Search, você não precisa verificar isEnd,
// basta chegar até o final do prefix
func (t *Trie) StartsWith(prefix string) bool {
	// TODO: Implementar
	return false
}

// AutoComplete retorna todas as palavras que começam com o prefix dado
// limit define o número máximo de sugestões (0 = sem limite)
// TODO: Esta é a parte mais desafiadora!
// Dica: você precisa:
//  1. Navegar até o node do prefix (com locks)
//  2. Fazer DFS a partir dali coletando palavras
//  3. O problema: você NÃO PODE segurar locks durante toda a DFS
//     porque isso bloquearia outras operações por muito tempo
//  4. Solução: copiar a estrutura necessária ou usar snapshot approach
func (t *Trie) AutoComplete(prefix string, limit int) []string {
	// TODO: Implementar
	return nil
}

// Delete remove uma palavra da trie
// TODO: Implementar (bonus se tiver tempo)
// Esta é complexa porque você precisa remover nodes que não são mais necessários
// Mas cuidado: não pode remover um node que é prefix de outra palavra
// Exemplo: se você tem "car" e "card", ao deletar "car" não pode remover o node 'r'
func (t *Trie) Delete(word string) bool {
	// TODO: Implementar
	return false
}

// Funções auxiliares que você provavelmente vai precisar

// findNode navega até o node que representa o final de um prefix
// Retorna o node encontrado (ou nil se prefix não existe)
// TODO: Implementar esta helper - será útil para Search, StartsWith e AutoComplete
func (t *Trie) findNode(prefix string) *TrieNode {
	// TODO: Implementar
	return nil
}

// collectWords faz DFS a partir de um node coletando todas as palavras
// currentWord é o prefix acumulado até chegar neste node
// TODO: Implementar esta helper para usar no AutoComplete
// Dica: esta função vai ser recursiva
func collectWords(node *TrieNode, currentWord string, words *[]string, limit int) {
	// TODO: Implementar DFS recursivo
	// Lembre-se de verificar se já coletou o limite de palavras
}
```

## 10. Skip List Thread-Safe

Skip Lists são a base dos Sorted Sets do Redis (ZADD, ZRANGE, ZRANK) e do storage engine do LevelDB/RocksDB. O ponto central: ao invés de balancear uma árvore deterministicamente (que é complexo), você usa probabilidade para manter O(log n) amortizado. Mais simples de implementar lock-free ou com fine-grained locking do que uma BST balanceada.

read: `https://selfboot.cn/en/2024/09/09/leveldb_source_skiplist/`

**Visualização**

```
Level 3: head ──────────────────────────── 50 ──────────── tail
Level 2: head ────────── 20 ─────────────  50 ──── 70 ──── tail
Level 1: head ─── 10 ─── 20 ─── 30 ──────  50 ─── 70 ──── tail
Level 0: head ─── 10 ─── 20 ─── 30 ─── 40 50 ─── 70 ─── 90 tail
```

Cada node tem um array de `forward` pointers, um por nível onde ele participa. Quando inserimos um node, você joga uma moeda para deciodir quantos níveis ele sobe.

Uma skip list é várias linked lists empilhadas. O nível 0 tem todos os nodes. O nível 1 tem ~1/2. O nível 2 ~1/4. E assim por diante.

**Probabilidade = p 0.5** -> em média, então metade dos nodes estão no nível 1, um quarto no nível 2, etc.

**O desafio de concorrência vs a Trie**:

- Na trie: locking vertical (desce por chars, cada node tem seu mutex)
- Na Skip List: locking **horizontal** na lista (varre níveis, então precisa do **update array**, que é um array de predecessores, antes de modificar ponteiros em múltiplos níveis ao mesmo tempo)

**Update array pattern**: antes de inserir / deletar, percorremos todos os níveis de cima para baixo, guardando em `update[level]` o último node que ficou "à esquerda" do ponto de inserção naquele nível. Depois, é modificado os ponteiros usando esses predecessores. Sem isso, não conseguimos saber onde conectar o novo node em cada nível.

### Trade-offs vs alternativas

| Estrutura         |   Busca   |  Insert    |  Delete    |      Concorrência       |
| :---------------- |  :------: |  :------:  |  :------:  |  ---------------------: | 
| Skip List         |  O(log n) |  O(log n)  |  O(log n)  |  Fácil (update array)   |
| BST balanceada    |  O(log n) |  O(log n)  |  O(log n)  |  Difícil (rotações)     |
| B-Tree    	    |  O(log n) |  O(log n)  |  O(log n)  |  Moderado               |
| Hash Map 			|    O(1)   |    O(1)    |    O(1)    |  Simples, mas sem ordem |


A skip list vai ser uma boa estrutura ideal quando precisamos de **range queries ordenadas** com implementação simples.

**Por que a probabilidade garante O(log n)?**

Um ponto importante de entender dessas estruturas, é que para resolver o problema de atualizar e deletar precisar re-organizar o nível inteiro, o autor William Pugh sugere a abordagem probabilistica, para não precisar re-organizar o nível inteiro.

Quando inserimos um node, jogamos uma moeda (`rand < p`) para decidir se ela "sobe" de nível. Com p = 0.5, na *média*:

- ~50% dos nodes existem no nível 1
- ~25% no nível 2
- ~12,5% no nível 3
- etc...

E todos no nível 0...

Isso vai criar a mesma distribuição de uma busca binária (binary search), a busca começa no nível mais alto (poucos nodes, que dá pulos grandes) e vai descendo, ou seja O(log n).

**Por que sem probabilidade seria O(n) na atualização?**

Se quisessemos garantir O(log n) deterministicamente, teriamos que manter a invariante de que exatamente metade dos nodes estão em cada nível acima.

Quando inserimos ou deletamos um node, precisamos **reorganizar quais nodes participam de cada nível** para manter esse balanceamento. Ou seja, reconstruir os níveis afetados, O(n).

Com probabilidade, simplesmente abandonamos a garantia determinística e aceitamos O(log n) esperado. O node decide seu próprio nível na hora do insert, uma vez, e nunca muda. Ou seja **nada precisa ser rebalanceado**.

Ë o mesmo trade-off do treap (árvore + heap com prioridade aleatória) vs AVL Tree.

### Estrutura de arquivos

`main.go`

```go
package main

import (
	"fmt"
	"math/rand"
)

func main() {
	sl := NewSkipList(4, 0.5, rand.New(rand.NewSource(42)))

	// Insert
	sl.Insert(10, "ten")
	sl.Insert(50, "fifty")
	sl.Insert(20, "twenty")
	sl.Insert(30, "thirty")
	sl.Insert(70, "seventy")

	// Search
	if val, ok := sl.Search(30); ok {
		fmt.Printf("Found 30: %v\n", val) // "thirty"
	}

	// RangeSearch: retorna todos os valores com score entre min e max (inclusive)
	results := sl.RangeSearch(15, 55)
	fmt.Printf("Range [15, 55]: %v\n", results) // [twenty thirty fifty]

	// Delete
	sl.Delete(20)
	if _, ok := sl.Search(20); !ok {
		fmt.Println("20 deleted successfully")
	}

	// Stats (útil para debugging)
	fmt.Printf("Size: %d\n", sl.Size())
}
```

`skiplist.go`

```go
package main

import (
	"math/rand"
	"sync"
)

const MaxLevel = 16

// SkipListNode representa um node em múltiplos níveis
type SkipListNode struct {
	score   int
	value   any
	forward []*SkipListNode // forward[i] = próximo node no nível i
	mu      sync.RWMutex    // opcional: se quiser locking por node
}

// SkipList é a estrutura principal
type SkipList struct {
	head     *SkipListNode
	maxLevel int
	p        float64 // probabilidade de subir de nível (geralmente 0.5)
	level    int     // nível atual máximo utilizado
	size     int
	mu       sync.RWMutex
	rng      *rand.Rand
}

// NewSkipList cria uma nova skip list
// maxLevel: altura máxima permitida
// p: probabilidade de um node subir de nível (0.5 = 50%)
// rng: source de random (injetado para testes determinísticos)
func NewSkipList(maxLevel int, p float64, rng *rand.Rand) *SkipList {
	// TODO: criar head node com maxLevel forward pointers (todos nil)
	// head.score pode ser -infinito (math.MinInt)
	panic("not implemented")
}

// randomLevel gera a altura de um novo node probabilisticamente
// Começa em 1 e incrementa enquanto rand < p E level < maxLevel
func (sl *SkipList) randomLevel() int {
	// TODO
	panic("not implemented")
}

// Insert insere ou atualiza um score+value na lista
// Usa o update array pattern:
//  1. Percorre do nível mais alto até 0, guardando predecessores em update[]
//  2. Gera randomLevel para o novo node
//  3. Reconecta ponteiros usando update[]
func (sl *SkipList) Insert(score int, value any) {
	// TODO
	panic("not implemented")
}

// Search busca por score. Retorna (value, true) se encontrado.
// Percorre de cima para baixo: enquanto forward[level] existe e score < target,
// avança. Quando não pode mais avançar, desce um nível.
func (sl *SkipList) Search(score int) (any, bool) {
	// TODO
	panic("not implemented")
}

// Delete remove o node com o score dado, se existir.
// Mesmo padrão do Insert: update array primeiro, depois reconecta ponteiros.
func (sl *SkipList) Delete(score int) bool {
	// TODO
	panic("not implemented")
}

// RangeSearch retorna todos os valores com score entre min e max (inclusive),
// em ordem crescente. Hint: chegue até min usando a lógica de Search,
// depois percorra o nível 0 enquanto score <= max.
func (sl *SkipList) RangeSearch(min, max int) []any {
	// TODO
	panic("not implemented")
}

// Size retorna o número de elementos na lista
func (sl *SkipList) Size() int {
	// TODO
	panic("not implemented")
}
```

`skiplist_test.go`

```go
package main

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestSkipList() *SkipList {
	return NewSkipList(4, 0.5, rand.New(rand.NewSource(42)))
}

// --- Testes Básicos ---

func TestInsertAndSearch(t *testing.T) {
	sl := newTestSkipList()

	sl.Insert(10, "ten")
	sl.Insert(30, "thirty")
	sl.Insert(20, "twenty")

	val, ok := sl.Search(20)
	assert.True(t, ok)
	assert.Equal(t, "twenty", val)

	_, ok = sl.Search(99)
	assert.False(t, ok)
}

func TestInsertUpdate(t *testing.T) {
	sl := newTestSkipList()
	sl.Insert(10, "ten")
	sl.Insert(10, "TEN-updated") // mesmo score, deve atualizar

	val, ok := sl.Search(10)
	assert.True(t, ok)
	assert.Equal(t, "TEN-updated", val)
	assert.Equal(t, 1, sl.Size()) // não deve duplicar
}

func TestDelete(t *testing.T) {
	sl := newTestSkipList()
	sl.Insert(10, "ten")
	sl.Insert(20, "twenty")
	sl.Insert(30, "thirty")

	deleted := sl.Delete(20)
	assert.True(t, deleted)
	assert.Equal(t, 2, sl.Size())

	_, ok := sl.Search(20)
	assert.False(t, ok)

	deleted = sl.Delete(99) // não existe
	assert.False(t, deleted)
}

func TestRangeSearch(t *testing.T) {
	sl := newTestSkipList()
	for _, s := range []int{10, 20, 30, 40, 50, 60, 70} {
		sl.Insert(s, fmt.Sprintf("%d", s))
	}

	results := sl.RangeSearch(20, 50)
	assert.Equal(t, []any{"20", "30", "40", "50"}, results)

	// range vazio
	results = sl.RangeSearch(100, 200)
	assert.Empty(t, results)
}

func TestRangeSearchEmptyList(t *testing.T) {
	sl := newTestSkipList()
	results := sl.RangeSearch(0, 100)
	assert.Empty(t, results)
}

// --- Testes de Ordem ---

func TestInsertionOrder(t *testing.T) {
	sl := newTestSkipList()
	scores := []int{50, 10, 90, 30, 70, 20, 80, 40, 60}
	for _, s := range scores {
		sl.Insert(s, s)
	}

	results := sl.RangeSearch(10, 90)
	assert.Len(t, results, 9)
	for i := 1; i < len(results); i++ {
		assert.LessOrEqual(t, results[i-1].(int), results[i].(int))
	}
}

// --- Testes de Concorrência ---

func TestConcurrentInserts(t *testing.T) {
	sl := NewSkipList(8, 0.5, rand.New(rand.NewSource(0)))
	var wg sync.WaitGroup
	n := 100

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(score int) {
			defer wg.Done()
			sl.Insert(score, score)
		}(i)
	}
	wg.Wait()

	assert.Equal(t, n, sl.Size())
	for i := 0; i < n; i++ {
		_, ok := sl.Search(i)
		assert.True(t, ok, "score %d not found", i)
	}
}

func TestConcurrentReadsAndWrites(t *testing.T) {
	sl := NewSkipList(8, 0.5, rand.New(rand.NewSource(0)))
	for i := 0; i < 50; i++ {
		sl.Insert(i, i)
	}

	var wg sync.WaitGroup
	// Writers
	for i := 50; i < 100; i++ {
		wg.Add(1)
		go func(score int) {
			defer wg.Done()
			sl.Insert(score, score)
		}(i)
	}
	// Readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(score int) {
			defer wg.Done()
			sl.Search(score)
		}(i)
	}
	// Range readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sl.RangeSearch(0, 100)
		}()
	}
	wg.Wait()
}

func TestConcurrentDeletes(t *testing.T) {
	sl := NewSkipList(8, 0.5, rand.New(rand.NewSource(0)))
	n := 50
	for i := 0; i < n; i++ {
		sl.Insert(i, i)
	}

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(score int) {
			defer wg.Done()
			sl.Delete(score)
		}(i)
	}
	wg.Wait()

	assert.Equal(t, 0, sl.Size())
}

// Run with: go test -race ./...
```

---

## Dicas Chave (sem entregar a solução)

**Update array pattern** — o coração do Insert e Delete:

```go
update := make([]*SkipListNode, sl.maxLevel)
curr := sl.head
for i := sl.level - 1; i >= 0; i-- {
    for curr.forward[i] != nil && curr.forward[i].score < score {
        curr = curr.forward[i]
    }
    update[i] = curr  // predecessor nesse nível
}
// Agora curr.forward[0] é o candidato (verificar se score == target)
```

**randomLevel:**

```go
level := 1
for level < sl.maxLevel && sl.rng.Float64() < sl.p {
    level++
}
return level
```

## 11. Exchange Order Book

Antes de implementar:

Um order book é simplesmente duas listas ordenadas:

- **bids**: quem quer comprar
- **asks**: quem quer vender

O matching acontece quando o melhor bid encontra o melhor ask num preço aceitável para os dois lados.

```
BIDS (compra)         		ASKS (venda)
preço | qty           		preço | qty
 102  |  5    <- max     	103  |  3  <- min
 101  | 10              	104  |  8
 100  |  2              	105  | 15
```

O matching ocorre quando `max(bids) >= min(asks)`, ou seja, alguém quer pagar **pelo menos tanto** quando alguém quer receber.

No exemplo acima, não tem match. Se chegasse um bid de 103, ele ia bater com o ask de 103.

**Price-time priority**: essa é a regra de desempate entre duas ordens no mesmo preço, a que chegou primeiro é executada primeiro. Isso significa que cada nível de preço tem uma fila fifo de ordens.

---

### Estrutura de dados ideal

Inicialmente, você pode achar que uma heap pode ser boa para esse challenge. Mas não é ideal.

Problema com uma heap: ela te dá O(1) para peek no topo e O(log n) para insert/remove, mas **não te dá cancelamento eficiente**. Se um usuário cancela uma ordem que está no meio da heap, você precisa encontrar essa ordem, que vai ser O(n).

Uma estrutura boa para o order book em produção é um **sorted map**, de price -> fila de ordens. Em go seria algo como `map[int][]*Order`, combinado com uma lista de preços ordenados.

- Para os bids (compra), queremos o maior preço primeiro
- Para os asks (venda), queremos o menor preço primeiro

Mas pode ser usado heap, como indice de preços e não de ordens individuais. Cada entrada no heap, é um nível de preço, e cada nível aponta para uma fila FIFO de ordens. Que vai dar o melhor dos dois mundos.

---

### Goroutines e buckets por ticker

Em produção, cada ticekr tem seu próprio order book isolado, como um bucket, e teriamos uma goroutine (ou pool) dedicada por ticker.

Centralizar isso tudo em um único lock seria um baita bottleneck e ponto de falha que uma exchange não pode ter.

Aqui, vamos fazer um único order book com um mutex, mas a arquitetura de buckets é o que teriamos em produção, é bom ter isso em mente.

---

### Challenge

main.go

```go
package main

import "fmt"

func main() {
    ob := NewOrderBook("BTC-USD")

    ob.AddOrder(NewOrder("o1", "alice", Bid, 102, 5))
    ob.AddOrder(NewOrder("o2", "bob",   Ask, 103, 3))
    ob.AddOrder(NewOrder("o3", "carol", Bid, 103, 2)) // deve matchear com o2

    trades := ob.Match()
    for _, t := range trades {
        fmt.Printf("TRADE: %s x %s @ %.2f qty %d\n",
            t.BidOrderID, t.AskOrderID, t.Price, t.Quantity)
    }

    ob.Cancel("o1")
    fmt.Printf("Book depth — bids: %d asks: %d\n",
        ob.BidDepth(), ob.AskDepth())
}
```

orderbook.go -> structs e assinaturas para implementar

```go
package main

import "sync"

type Side int
const (
    Bid Side = iota
    Ask
)

// Order representa uma ordem individual no book
type Order struct {
    ID        string
    UserID    string
    Side      Side
    Price     int       // em centavos para evitar float
    Quantity  int
    Timestamp int64     // unix nano — usado para price-time priority
}

// Trade representa uma execução — quando bid e ask se encontram
type Trade struct {
    BidOrderID string
    AskOrderID string
    Price      int
    Quantity   int
}

// PriceLevel agrupa todas as ordens num mesmo preço (fila FIFO)
type PriceLevel struct {
    Price  int
    Orders []*Order
}

// OrderBook mantém bids e asks para um único ticker
type OrderBook struct {
    Symbol string

    // TODO: estrutura para bids ordenados (maior preço primeiro)
    // TODO: estrutura para asks ordenados (menor preço primeiro)
    // TODO: índice de orderID -> Order para cancelamento O(1)

    mu sync.Mutex
}

func NewOrderBook(symbol string) *OrderBook { panic("not implemented") }
func NewOrder(id, userID string, side Side, price, qty int) *Order { panic("not implemented") }

// AddOrder adiciona uma ordem ao lado correto do book
func (ob *OrderBook) AddOrder(o *Order) { panic("not implemented") }

// Match executa todas as ordens que se cruzam e retorna os trades gerados.
// Regra: enquanto max(bid) >= min(ask), executa pelo preço do ask (ou bid, sua escolha).
// Quantidade executada = min(bid.Quantity, ask.Quantity).
// Ordens parcialmente executadas permanecem no book com quantidade reduzida.
func (ob *OrderBook) Match() []Trade { panic("not implemented") }

// Cancel remove uma ordem do book por ID
func (ob *OrderBook) Cancel(orderID string) bool { panic("not implemented") }

// BidDepth retorna o número de ordens ativas no lado bid
func (ob *OrderBook) BidDepth() int { panic("not implemented") }

// AskDepth retorna o número de ordens ativas no lado ask
func (ob *OrderBook) AskDepth() int { panic("not implemented") }
```

## 12. TCP Server com Worker Pool e Backpressure

### Por que importa

Servidores em produção não podem aceitar conexões ilimitadas, precisamos controlar quantas requisições simultâneas processa para não esgotar a CPU, memória ou file descriptors.

Worker pools com filas limitadas nos permitem fazer isso, podemos rejeitar ou postergar o aceite de novas conexões quando o sistema está saturado. Isso é um backpressure natural, o servidor comunica ao cliente "servidor sobrecarregado, tente depois", ao invés de crashar o degradar o sistema todo.

### Entendimento

Existem duas arquiteturas clássicas para TCP servers em Go:

**Goroutine-per-connection** (padrão ingênuo)

Cada conexão nova vai spawnar uma goroutine dedicata que vive até a conexão fechar. Simples de implementar, mas não tem controle de backpressure. Se chegar 10k conexões simultâneas, você cria 10k de goroutines e pode estourar a quantidade de memória usada ou degradas performance por context switching excessivo no go scheduler.

**Worker pool com fila limitada**

Temos N workers fixos (goroutines) que pegam conexões de uma fila com capacidade M. Se a fila encher, novas conexões são rejeitadas ou ficam em espera. Isso dá um controle total sobre consumação de recursos. Podemos escolher quantos workers podemos sustentar e quantas conexões podem enfileiras antes de rejeitar.

A diferença central é que no worker pool você **desacopla** aceitar conexões de processar conexões. O `Accept()` loop continua rodando, mas as conexões vão para um channel com buffer. Se o channel encher, o `select` com `default` rejeita a conexão imediatamente sem bloquear o accept loop.

### Modelo mental

Imagina um restaurante com 4 garçons (workers), e uma fila de espera com 10 lugares (queue). Quando chega o 15 cliente e a fila está cheia, o host diz "estamos lotados, volte em 20 minutos". O restaurante não contrata mais garçons na hroa nem deixa 100 pesoas esperando indefinidamente. O restaurante não contrata mais garços na hora e nem deixa 100 pessoas esperando indefinidamente, ele sabe a sua capacidade e comunica seus limites.

No TCP server com worker pool, o accept loop é o host que recebe clientes. Os workers são os garçons que processam pedidos. A fila `connCh` é a área de espera. Quando a fila enche, você rejeita a conexão imediatamente (close) em vez de bloquear o accept loop ou criar goroutines sem limite.

---

### O que você vai implementar

Um TCP server que aceita conexões em `:8080`, enfileira até `QueueSize` conexões num channel, e processa com `Workers` goroutines fixas. Quando a fila está cheia, novas conexões são rejeitadas (close imediato). O servidor também suporta shutdown graceful, para de aceitar novas conexões mas espera as ativas terminarem.

`main.go`

```go
package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	config := ServerConfig{
		Addr:        ":8080",
		Workers:     4,
		QueueSize:   10,
		IdleTimeout: 30 * time.Second,
	}

	server := NewTCPServer(config, EchoHandler)

	log.Printf("Starting TCP server on %s with %d workers, queue size %d\n",
		config.Addr, config.Workers, config.QueueSize)

	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}

// EchoHandler é um handler simples que faz echo de tudo que recebe
func EchoHandler(conn *Connection) error {
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return err
		}

		if _, err := conn.Write(buf[:n]); err != nil {
			return err
		}
	}
}
```

`server.go`

```go
package main

import (
	"net"
	"sync"
	"time"
)

type ServerConfig struct {
	Addr        string        // endereço para bind, ex: ":8080"
	Workers     int           // número de workers no pool
	QueueSize   int           // capacidade da fila de conexões
	IdleTimeout time.Duration // timeout para conexões idle
}

// Connection wraps net.Conn com métodos auxiliares
type Connection struct {
	net.Conn
	// TODO: adicionar campos úteis como ID, timestamp, etc
}

// Handler processa uma conexão. Retorna erro se algo der errado.
type Handler func(*Connection) error

// TCPServer gerencia o worker pool e a fila de conexões
type TCPServer struct {
	config   ServerConfig
	handler  Handler
	listener net.Listener
	connCh   chan *Connection // fila de conexões pendentes
	wg       sync.WaitGroup
	// TODO: campos para shutdown graceful
}

func NewTCPServer(config ServerConfig, handler Handler) *TCPServer {
	// TODO: inicializar server
	panic("not implemented")
}

// Start inicia o servidor — accept loop + worker pool
func (s *TCPServer) Start() error {
	// TODO:
	// 1. net.Listen no config.Addr
	// 2. Spawnar workers (s.config.Workers goroutines)
	// 3. Accept loop: aceita conexões e tenta enfileirar em s.connCh
	//    - Se connCh estiver cheio, rejeita a conexão (close imediato)
	// 4. Cada worker pega conexões de s.connCh e chama s.handler
	panic("not implemented")
}

// Shutdown para o servidor gracefully
func (s *TCPServer) Shutdown() error {
	// TODO:
	// 1. Fecha s.listener (para de aceitar novas conexões)
	// 2. Fecha s.connCh (workers terminam quando drenam a fila)
	// 3. s.wg.Wait() até todos os workers terminarem
	panic("not implemented")
}
```

`server_test.go`

```go
func TestEchoServer(t *testing.T) {
	config := ServerConfig{
		Addr:        ":0", // porta aleatória
		Workers:     2,
		QueueSize:   5,
		IdleTimeout: 5 * time.Second,
	}

	server := NewTCPServer(config, EchoHandler)
	go server.Start()
	defer server.Shutdown()

	// espera servidor subir
	<-server.ready

	conn, err := net.Dial("tcp", server.Addr())
	require.NoError(t, err)
	defer conn.Close()

	// envia mensagem
	_, err = conn.Write([]byte("hello\n"))
	require.NoError(t, err)

	// lê echo
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	require.NoError(t, err)
	assert.Equal(t, "hello\n", response)
}

func TestBackpressure(t *testing.T) {
	config := ServerConfig{
		Addr:      ":0",
		Workers:   2,
		QueueSize: 2, // fila pequena pra forçar rejeição
	}

	// handler lento — segura conexão por 1s
	slowHandler := func(conn *Connection) error {
		defer conn.Close()
		time.Sleep(1 * time.Second)
		return nil
	}

	server := NewTCPServer(config, slowHandler)
	go server.Start()
	defer server.Shutdown()

	<-server.ready

	// abre Workers + QueueSize conexões (todas devem ser aceitas)
	var wg sync.WaitGroup
	var mu sync.Mutex
	accepted := 0
	rejected := 0

	for range 10 {
		wg.Go(func() {
			conn, err := net.Dial("tcp", server.Addr())
			if err != nil {
				mu.Lock()
				rejected++
				mu.Unlock()
				return
			}
			defer conn.Close()

			buf := make([]byte, 1)
			conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			_, err = conn.Read(buf)

			mu.Lock()
			if err != nil {
				rejected++
			} else {
				accepted++
			}
			mu.Unlock()
		})
	}

	wg.Wait()

	// deve ter rejeitado algumas conexões
	assert.Greater(t, rejected, 0, "server should reject connections when saturated")
}

func TestGracefulShutdown(t *testing.T) {
	config := ServerConfig{
		Addr:      ":0",
		Workers:   2,
		QueueSize: 5,
	}

	server := NewTCPServer(config, EchoHandler)
	go server.Start()
	<-server.ready

	// abre conexão
	conn, err := net.Dial("tcp", server.Addr())
	require.NoError(t, err)

	// shutdown deve esperar conexões ativas terminarem
	done := make(chan struct{})
	go func() {
		server.Shutdown()
		close(done)
	}()

	// fecha conexão
	conn.Close()

	// shutdown deve completar
	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown took too long")
	}
}
```

## 13. Idempotent Payment Processing with PostgreSQL

Em qualquer sistema de pagamento (Stripe, Square, etc), as requisições podem falhar de forma imprevizível: timeout de rede, crash do servidor após processar, mas antes de responder o cliente. O cliente faz retry com a mesma requisição, mas **não pode processar o pagamento duas vezes**

Idempotência resolve isso: a mesma requisição com a mesma chave sempre retorna o mesmo resultado, sem duplicatas.

Existem várias formas de resolver isso:

1. Usar advisory lock no PostgreSQL: porém causa contenção nos recursos, outras requisições podem ficar esperando.
2. Usar sharding por idempotencyKey. É o que vamos fazer.

**Solução:**

Sharding por idempotencyKey. Cada shard é uma fila com 1 worker. Requisições com a mesma chave vão sempre pro mesmo shard -> processam sequencialmente.

Chaves diferentes vão rodar em paralelo.

**Worker goroutine* *:

Cada shard tem 1 worker que recebe tasks via channel, processa sequencialmente. Requisições com chaves diferentes vão pra shards diferentes -> paralelo.

---

### Schema PostgreSQL

```sql
IF NOT EXISTS (
	SELECT 1
	FROM pg_type
	WHERE typname = 'payment_status'
) THEN
	CREATE TYPE payment_status AS ENUM (
		'pending',
		'completed',
		'failed'
	);

	RAISE NOTICE 'ENUM payment_status created.';
END IF;

CREATE TABLE IF NOT EXISTS payments (
    id BIGSERIAL PRIMARY KEY,
    idempotency_key VARCHAR(255) NOT NULL UNIQUE,
    user_id VARCHAR(255) NOT NULL,
    amount BIGINT NOT NULL, -- centavos
    currency VARCHAR(3) NOT NULL,
    status payment_status NOT NULL, -- 'pending', 'completed', 'failed'
    stripe_charge_id VARCHAR(255),
    error_message VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_idempotency_key ON payments(idempotency_key);
CREATE INDEX IF NOT EXISTSidx_user_id ON payments(user_id);
```

---

### Challenge

**main.go**

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)


func main() {
	db := NewDB()
	defer db.conn.Close()

	service := NewPaymentService(db, 4) // 4 shards

	// Primeira requisição — processa
	result1, err := service.ProcessPayment(context.Background(), &PaymentRequest{
		IdempotencyKey: "key-123",
		UserID:         "alice",
		Amount:         10000,
		Currency:       "USD",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Payment 1: ID=%d Status=%s\n", result1.PaymentID, result1.Status)

	// Segunda requisição com MESMA chave — retorna resultado anterior
	result2, err := service.ProcessPayment(context.Background(), &PaymentRequest{
		IdempotencyKey: "key-123",
		UserID:         "alice",
		Amount:         10000,
		Currency:       "USD",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Payment 2 (retry): ID=%d Status=%s\n", result2.PaymentID, result2.Status)

	if result1.PaymentID == result2.PaymentID {
		fmt.Println("Idempotência garantida: mesma chave, mesmo resultado")
	}

	// Terceira requisição com chave diferente — cria novo pagamento
	result3, err := service.ProcessPayment(context.Background(), &PaymentRequest{
		IdempotencyKey: "key-456",
		UserID:         "alice",
		Amount:         5000,
		Currency:       "USD",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Payment 3: ID=%d Status=%s\n", result3.PaymentID, result3.Status)
}
```

**payment.go**

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"time"
)

type PaymentRequest struct {
	IdempotencyKey string
	UserID         string
	Amount         int64
	Currency       string
}

type PaymentResult struct {
	PaymentID      int64
	Status         string
	StripeChargeID string
	ErrorMessage   string
}

type PaymentService struct {
	db     *sql.DB
	shards []*Shard
}

// Shard processa pagamentos sequencialmente para um conjunto de chaves
type Shard struct {
	id   int
	reqCh chan *paymentTask
	// TODO: adicionar campos necessários
}

type paymentTask struct {
	req    *PaymentRequest
	result chan *paymentTaskResult
}

type paymentTaskResult struct {
	result *PaymentResult
	err    error
}

func NewPaymentService(db *sql.DB, numShards int) *PaymentService {
	// TODO: inicializar service com N shards
	// cada shard tem sua goroutine worker que processa a fila
	panic("not implemented")
}

// ProcessPayment enfileira um pagamento no shard correto baseado no hash da chave
// Retorna o resultado (novo ou anterior se já processado)
func (ps *PaymentService) ProcessPayment(ctx context.Context, req *PaymentRequest) (*PaymentResult, error) {
	// TODO: 
	// 1. hash(idempotencyKey) % numShards -> shard index
	// 2. Enfileira requisição no shard via channel
	// 3. Aguarda resultado do shard
	panic("not implemented")
}

// shardWorker processa requisições sequencialmente para um shard
// Cada requisição: SELECT + INSERT ON CONFLICT DO NOTHING
func (s *Shard) worker(db *sql.DB) {
	// TODO:
	// loop infinito: recebe task do s.reqCh
	// para cada task:
	//   1. SELECT * FROM payments WHERE idempotency_key = ? -> já existe?
	//      SIM: retorna resultado anterior
	//      NÃO: continua
	//   2. Processa pagamento (simula Stripe)
	//   3. INSERT ... ON CONFLICT DO NOTHING
	//   4. SELECT * para recuperar o resultado inserido
	//   5. Envia resultado via task.result
	panic("not implemented")
}

func hashKey(key string, shards int) int64 {
	h := fnv.New64a()
	h.Write([]byte(key))
	return int64(h.Sum64() % uint64(shards))
}

func processStripeCharge(ctx context.Context, amount int64, userID string) (chargeID string, err error) {
	// TODO: simular processamento da Stripe
	// retorna um charge ID fictício e nil error
	panic("not implemented")
}
```

**db.go**

```go
package main

import (
	"database/sql"
	"log"
)

type DB struct {
	conn *sql.DB
}

func NewDB() *DB {
	db, err := sql.Open("postgres",
		"host=localhost user=postgres password=postgres dbname=payments_test port=5444 sslmode=disable")
	if err != nil {
		log.Fatalf("err opening db: %v", err)
	}

	if err := createTables(db); err != nil {
		log.Fatalf("err opening db: %v", err)
	}

	return &DB{
		conn: db,
	}
}

func createTables(db *sql.DB) error {
	_, err := db.Exec(`
DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_status') THEN
		CREATE TYPE payment_status AS ENUM ('pending', 'completed', 'failed');
	END IF;
END $$;

CREATE TABLE IF NOT EXISTS payments (
    id BIGSERIAL PRIMARY KEY,
    idempotency_key VARCHAR(255) NOT NULL UNIQUE,
    user_id VARCHAR(255) NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    status payment_status NOT NULL,
    stripe_charge_id VARCHAR(255),
    error_message VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_idempotency_key ON payments(idempotency_key);
CREATE INDEX IF NOT EXISTS idx_user_id ON payments(user_id);
	`)
	if err != nil {
		return err
	}
	return nil
}
```

**payment_test.go**

```go
package main

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("postgres",
		"host=localhost user=postgres password=postgres dbname=payments_test sslmode=disable")
	require.NoError(t, err)

	_, err = db.Exec(`DROP TYPE IF EXISTS payment_status CASCADE`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TYPE payment_status AS ENUM ('pending', 'completed', 'failed')`)
	require.NoError(t, err)
	_, err = db.Exec(`DROP TABLE IF EXISTS payments`)
	require.NoError(t, err)
	_, err = db.Exec(`
		CREATE TABLE payments (
			id BIGSERIAL PRIMARY KEY,
			idempotency_key VARCHAR(255) NOT NULL UNIQUE,
			user_id VARCHAR(255) NOT NULL,
			amount BIGINT NOT NULL,
			currency VARCHAR(3) NOT NULL,
			status payment_status NOT NULL,
			stripe_charge_id VARCHAR(255),
			error_message VARCHAR(255),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)

	return db
}

// TestIdempotency — mesma chave retorna mesmo resultado
func TestIdempotency(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewPaymentService(db, 4)
	ctx := context.Background()

	req := &PaymentRequest{
		IdempotencyKey: "idempotent-key-1",
		UserID:         "alice",
		Amount:         10000,
		Currency:       "USD",
	}

	result1, err := service.ProcessPayment(ctx, req)
	require.NoError(t, err)
	assert.NotZero(t, result1.PaymentID)
	assert.Equal(t, "completed", result1.Status)

	result2, err := service.ProcessPayment(ctx, req)
	require.NoError(t, err)

	assert.Equal(t, result1.PaymentID, result2.PaymentID)
	assert.Equal(t, result1.Status, result2.Status)
	assert.Equal(t, result1.StripeChargeID, result2.StripeChargeID)
}

// TestConcurrentIdempotency — 10 goroutines com mesma chave não duplicam
func TestConcurrentIdempotency(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewPaymentService(db, 4)
	ctx := context.Background()

	req := &PaymentRequest{
		IdempotencyKey: "concurrent-key-1",
		UserID:         "bob",
		Amount:         5000,
		Currency:       "USD",
	}

	results := make([]*PaymentResult, 10)
	errs := make([]error, 10)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = service.ProcessPayment(ctx, req)
		}(i)
	}

	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "request %d failed", i)
	}

	firstID := results[0].PaymentID
	for i, result := range results {
		assert.Equal(t, firstID, result.PaymentID,
			"request %d got different payment ID", i)
	}

	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM payments WHERE idempotency_key = $1`,
		req.IdempotencyKey).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should have exactly 1 payment in DB")
}

// TestDifferentKeysParallel — chaves diferentes processam em paralelo, sem bloquear
func TestDifferentKeysParallel(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewPaymentService(db, 4)
	ctx := context.Background()

	results := make([]*PaymentResult, 10)
	errs := make([]error, 10)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := &PaymentRequest{
				IdempotencyKey: fmt.Sprintf("key-%d", idx),
				UserID:         fmt.Sprintf("user-%d", idx),
				Amount:         10000,
				Currency:       "USD",
			}
			results[idx], errs[idx] = service.ProcessPayment(ctx, req)
		}(i)
	}

	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "request %d failed", i)
	}

	for i := 0; i < 10; i++ {
		for j := i + 1; j < 10; j++ {
			assert.NotEqual(t, results[i].PaymentID, results[j].PaymentID,
				"different keys should create different payments")
		}
	}
}
```

---

### Dicas Chave

**Hash para shard:**

```go
shard := hashKey(req.IdempotencyKey) % int64(len(ps.shards))
```

**SELECT + INSERT ON CONFLICT:**

```sql
-- Verifica se já existe
SELECT id, status, stripe_charge_id FROM payments 
WHERE idempotency_key = $1

-- Se não, insere
INSERT INTO payments (...) VALUES (...)
ON CONFLICT (idempotency_key) DO NOTHING
RETURNING id, status, stripe_charge_id

-- Se conflict, SELECT novamente para pegar o resultado anterior
```

## 14. Mining Pool with Stratum Protocol

### O que é um Stratum?

Stratum é o protocolo que mineradores de Bitcoin usam para se comunicam com mining pools.

É tipo um **JSON-RCP sobre TCP** com delimitador de linha, ou seja, cada mensagem é um JSON em uma linha nova. É simples para implementar em 1 hora, útil para entender parsing, dispatch de mensagens, e concorrência.

Uma pool age como um **servidor que distribui trabalho** (que são blocos para tentar minerar) e coleta **shares** (resultados parciais que provam se o minerador está trabalhando ou não).

Analogia de produção: pensa numa pool como um load balancer que distribui jobs, e os mineradores como workers. O que vai deixar a pool interessante é que cada minerador tem uma **dificuldade diferente**, um minerador fraco recebe trabalho mais fácil de fazera, e a pool agrega o hashrate total.

### As 3 mensagens principais do protocolo

O Stratum simplificado que você vai implementar tem exatamente 3 tipos:

```mining.subscribe``` - o minerador se conecta, bassicamente diz "quero trabalhar". A pool responde com o ID de sessão. Esse ID da sessão é o handshake.

```mining.notify``` - a pool envia trabalho para o minerador. Vai conter o job ID, o bloco parcial, e a dificuldade alvo do trabalho. Isso é **server-push**, e não request-response, a pool manda quando quer.

```mining.submit``` - o minerador encontrou uma share válida e envia de volta. A pool valida, registra o hashrate, e responde com sucesso ou rejeição.

### Hashrate Marketplace vs Exchange

Nesse desafio, deve implementar **dois modelos de negócio** diferentes para a pool:

No **marketplace**: mineradores venden hashrate diretamente para compradores por um preço fixo por TH/s. É como a Amazon por exemplo, você lista seu produto, o comprador paga o preço pedido. Simples, não precisa de matching engine e order book.

Na **exchange**: existe um order book real (geralmente por ação / asset). Aqui os compradores postam bids ("quero 100 TH/s por $0,05/TH/s"), os vendedores postam asks ("tenho 100 TH/s por $0.06/TH/s"), e o sistema faz o match quando os preços se cruzam. No challenge 11, implementamos um order book de uma exchange, o modelo mental aqui é o mesmo, mas aplicado a hashrate em vez de ativos financeiros.

Diferença do design: no marketplace só precisamos de um ```map[minerID]hashrate``` com um RWMutex. Na exchange precisamos de um order book completo com price-time priority.

Implemente os dois, e exponha usando uma interface comum.

---

### Pontos importantes dessa implementação

1. Porque `bufio.Scanner` com SplitFunc para \n ao invés de `json.NewDecoder` diretamente?

Por backpressure. O decoder lê ahead de forma muito agressiva, e pode consumir bytes da próxima mensagem, isso quebraria um protocolo framing-based.

2. Por que a interface `HashrateMarket` ao invés de um type switch no **Server**?

O Server nào deve saber nada sobre o modelo de negócio implementado. Podemos trocar a lógica de um Marketplace por Exchange em runtime sem tocar na camada TCP.

3. Por quê goroutines separadas no BroadCastJob ao invés de fazer sequencial?

Pode parecer meio óbvio (concorrência, mais rápido teoricamente). Porém, para ser mais específico, um minerador com buffer de rede cheio (lento, com lag, ou hostíl), não deveria bloquear os outros.

---

### Estrutura do projeto

```go
challenge14/
├── main.go
├── protocol/
│   └── stratum.go      # tipos de mensagem e parsing
├── pool/
│   ├── server.go       # TCP server + connection handling
│   ├── miner.go        # estado por conexão de minerador
│   └── dispatcher.go   # distribui jobs para mineradores
├── market/
│   ├── marketplace.go  # modelo de preço fixo
│   └── exchange.go     # order book de hashrate
└── pool_test.go
```

**main.go**

```go

func main() {
	mp := market.NewMarketplace(0.05)
	server := pool.NewServer(":8080", mp)

	go func() {
		if err := server.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	// dá tempo pro listener subir antes de conectar
	time.Sleep(100 * time.Millisecond)

	minerID := simulateMiner(":8080", "cgminer/4.10.0")
	time.Sleep(2 * time.Second)
	mp.Tick(1.0) // 1 h
	fmt.Printf("Total hashrate: %.20f\n", mp.TotalHashrate())
	fmt.Printf("Revenue %s: $%.64f\n", minerID, mp.Revenue(minerID))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

// simulateMiner conecta via TCP real e envia as 3 mensagens do protocolo.
// Isso testa o protocolo de ponta a ponta sem precisar de um minerador real.
func simulateMiner(addr, userAgent string) string {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("miner connect error: %v", err)
		return ""
	}
	defer conn.Close()

	// 1. Subscribe
	sendJSON(conn, protocol.Message{
		ID: func() *int {
			num := rand.IntN(math.MaxInt)
			return &num
		}(),
		Method: protocol.Subscribe,
		Params: mustMarshal(protocol.SubscribeParams{
			UserAgent: userAgent,
		}),
	})

	// 2. le a response do servidor
	scanner := bufio.NewScanner(conn)
	scanner.Scan()

	var resp protocol.Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		log.Printf("miner response unmarshal error: %v", err)
		return ""
	}

	log.Printf("miner received response: %+v", resp)

	var result map[string]string
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		log.Printf("miner response result unmarshal error: %v", err)
		return ""
	}

	// id real no mktplc
	serverMinerID := result["session_id"]

	// 3. Submit uma share (simulada — hash pode ser inválido para este exemplo)
	sendJSON(conn, protocol.Message{
		ID: func() *int {
			num := rand.IntN(math.MaxInt)
			return &num
		}(),
		Method: protocol.Submit,
		Params: mustMarshal(protocol.SubmitParams{
			MinerID: serverMinerID,
			JobID:   "job-001",
			Nonce:   42,
			Hash:    "0000abcd1234", // leading zeros simulam dificuldade baixa
		}),
	})

	// dá tempo pro servidor processar a share antes de fechar a conexão
	time.Sleep(500 * time.Millisecond)

	return serverMinerID
}

func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func sendJSON(conn net.Conn, v any) {
	b, _ := json.Marshal(v)
	b = append(b, '\n') // Stratum usa newline como delimitador
	conn.Write(b)
}
```

**protocol/stratum.go**

```go
package protocol

import (
	"encoding/json"
	"fmt"
)

// MessageType representa os 3 tipos de mensagem do Stratum simplificado.
// Em produção (SlushPool, F2Pool) existem mais, mas esses 3 cobrem o core.
type MessageType string

const (
	Subscribe MessageType = "mining.subscribe"
	Notify    MessageType = "mining.notify"
	Submit    MessageType = "mining.submit"
)

// Message é o envelope JSON que toda mensagem Stratum usa.
// O campo Params é raw porque cada tipo de mensagem tem params diferentes —
// você vai fazer unmarshal do params específico depois de conhecer o Method.
type Message struct {
	ID     *int            `json:"id"`     // nil para server-push (notify)
	Method MessageType     `json:"method"`
	Params json.RawMessage `json:"params"`
}

// Response é o que o server manda de volta para requests com ID.
type Response struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *RPCError       `json:"error"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// SubscribeParams são os params do mining.subscribe.
type SubscribeParams struct {
	UserAgent string `json:"user_agent"` // ex: "cgminer/4.10.0"
	SessionID string `json:"session_id"` // vazio na primeira conexão
}

// NotifyParams são os params do mining.notify (server -> miner).
type NotifyParams struct {
	JobID      string `json:"job_id"`
	PrevHash   string `json:"prev_hash"`
	Difficulty uint64 `json:"difficulty"` // alvo simplificado: share válida se hash < target
	CleanJobs  bool   `json:"clean_jobs"` // true = descarta jobs anteriores
}

// SubmitParams são os params do mining.submit (miner -> server).
type SubmitParams struct {
	MinerID string `json:"miner_id"`
	JobID   string `json:"job_id"`
	Nonce   uint64 `json:"nonce"`  // o minerador encontrou esse nonce
	Hash    string `json:"hash"`   // sha256 do bloco com esse nonce
}

// Parse faz o dispatch do JSON cru para o tipo correto.
// Retorna um dos tipos acima ou erro se o método for desconhecido.
func Parse(data []byte) (*Message, error) {
	// TODO: unmarshal em Message, validar Method
	return nil, fmt.Errorf("not implemented")
}

// ValidateShare verifica se o hash submetido satisfaz a dificuldade do job.
// Simplificação: conta leading zeros no hash (hex string).
// Em Bitcoin real, compara com um target de 256 bits.
func ValidateShare(hash string, difficulty uint64) bool {
	// TODO: implementar validação simplificada
	return false
}
```

**pool/miner.go**

```go
package pool

import (
	"bufio"
	"net"
	"sync"
	"time"
)

// MinerState representa o estado de um minerador conectado.
// Cada conexão TCP tem exatamente um MinerState.
type MinerState struct {
	mu sync.RWMutex

	ID        string
	Conn      net.Conn
	Writer    *bufio.Writer // buffered para não fazer syscall por mensagem
	SessionID string

	// Hashrate tracking: usamos uma janela deslizante de 60s.
	// shares aceitas nos últimos 60s * dificuldade média = hashrate estimado.
	ShareWindow []shareEntry
	Difficulty  uint64 // dificuldade atual atribuída a esse minerador

	CurrentJobID string
	ConnectedAt  time.Time
	LastShareAt  time.Time
}

type shareEntry struct {
	At         time.Time
	Difficulty uint64
}

// HashrateTHs retorna o hashrate estimado em TH/s nos últimos windowSec segundos.
// Fórmula: sum(difficulty_of_each_share) / window_seconds / 1e12
func (m *MinerState) HashrateTHs(windowSec int) float64 {
	// TODO: filtrar shares dentro da janela, somar dificuldades, dividir
	return 0
}

// SendMessage serializa e envia uma mensagem para o minerador.
// Usa bufio.Writer — precisa de Flush() depois ou o dado fica no buffer.
func (m *MinerState) SendMessage(v any) error {
	// TODO: json.Marshal, Write, '\n', Flush
	// Lembra: o Writer não é thread-safe sozinho — o mu protege
	return nil
}
```

**pool/server.go**

```go
package pool

import (
	"bufio"
	"context"
	"log"
	"net"
	"sync"
)

// Server é o TCP server da mining pool.
// Aceita conexões de mineradores, faz parsing das mensagens Stratum,
// e despacha para os handlers corretos.
type Server struct {
	addr     string
	listener net.Listener

	mu     sync.RWMutex
	miners map[string]*MinerState // minerID -> estado

	dispatcher *Dispatcher
	market     HashrateMarket // interface — pode ser Marketplace ou Exchange

	wg  sync.WaitGroup
	ctx context.Context
	cancel context.CancelFunc
}

// HashrateMarket é a interface comum entre Marketplace e Exchange.
// Isso permite que o Server não saiba qual modelo está usando.
type HashrateMarket interface {
	RegisterMiner(minerID string, hashrateTHs float64) error
	UnregisterMiner(minerID string)
	TotalHashrate() float64
	Revenue(minerID string) float64 // quanto esse minerador ganhou
}

func NewServer(addr string, market HashrateMarket) *Server {
	// TODO
	return nil
}

func (s *Server) Start() error {
	// TODO: net.Listen, loop de Accept, goroutine por conexão
	return nil
}

func (s *Server) handleConn(conn net.Conn) {
	// TODO: criar MinerState, bufio.Scanner com SplitFunc de newline,
	// loop de leitura, dispatch para handleMessage
}

func (s *Server) handleMessage(miner *MinerState, data []byte) {
	// TODO: protocol.Parse, switch em msg.Method:
	//   Subscribe -> gera SessionID, registra miner, envia response
	//   Submit    -> valida share, atualiza hashrate, responde
	//   Notify    -> mineradores não mandam notify, retornar erro
}

func (s *Server) Shutdown(ctx context.Context) error {
	// TODO: fechar listener, cancelar context, aguardar wg com timeout
	return nil
}

// BroadcastJob envia um novo job para todos os mineradores conectados.
// Chamado pelo Dispatcher quando um novo bloco chega.
func (s *Server) BroadcastJob(job *protocol.NotifyParams) {
	// TODO: RLock miners, iterar, SendMessage em goroutines separadas
	// por que goroutines separadas aqui? pensa em um minerador lento...
}

func (s *Server) Addr() string {
	// TODO: retornar listener addr
}
```

**market/marketplace.go**

```go
package market

import (
	"fmt"
	"sync"
)

// Marketplace implementa HashrateMarket com modelo de preço fixo.
// Mineradores registram hashrate, recebem pagamento proporcional ao total.
// É o modelo mais simples: sem order book, sem matching.
type Marketplace struct {
	mu sync.RWMutex

	pricePerTH float64 // USD por TH/s por hora
	miners     map[string]*minerRecord
}

type minerRecord struct {
	HashrateTHs float64
	EarnedUSD   float64
}

func NewMarketplace(pricePerTH float64) *Marketplace {
	// TODO
	return nil
}

func (m *Marketplace) RegisterMiner(minerID string, hashrateTHs float64) error {
	// TODO
	return nil
}

func (m *Marketplace) UnregisterMiner(minerID string) {
	// TODO
}

func (m *Marketplace) TotalHashrate() float64 {
	// TODO: RLock, somar hashrates
	return 0
}

func (m *Marketplace) Revenue(minerID string) float64 {
	// TODO
	return 0
}

// Tick é chamado periodicamente (ex: a cada minuto) para calcular e distribuir revenue.
// Revenue de cada minerador = (seu hashrate / total hashrate) * pricePerTH * totalHashrate * (1/60 hora)
func (m *Marketplace) Tick(durationHours float64) {
	// TODO: Lock, calcular proporção de cada minerador, atualizar EarnedUSD
}
```

**market/exchange.go**

```go
package market

import (
	"sync"
	"time"
)

// Order representa uma oferta no order book de hashrate.
type Order struct {
	ID          string
	MinerID     string    // quem está vendendo hashrate
	HashrateTHs float64   // quantidade ofertada
	PricePerTH  float64   // preço pedido (ask) ou máximo (bid)
	Side        OrderSide
	Cancelled   bool
	CreatedAt   time.Time
}

type OrderSide int

const (
	Bid OrderSide = iota // comprador: "quero comprar X TH/s por até $Y"
	Ask                  // vendedor (minerador): "tenho X TH/s por $Y"
)

// Exchange implementa HashrateMarket com order book.
// A lógica de matching é price-time priority, igual ao Challenge 11,
// mas aqui o "ativo" é hashrate em vez de um par de moedas.
type Exchange struct {
	mu sync.Mutex

	asks   *askHeap // sorted by price ascending  (vendedores mais baratos primeiro)
	bids   *bidHeap // sorted by price descending (compradores que pagam mais primeiro)

	orders map[string]*Order // orderID -> order (lookup rápido para UnregisterMiner)
	revenue  map[string]float64 // minerID -> USD acumulado
}

func NewExchange() *Exchange {
	// TODO
	return nil
}

// PlaceOrder adiciona um bid ou ask no order book e tenta fazer matching.
func (e *Exchange) PlaceOrder(order *Order) error {
	// TODO: adicionar no lado correto, chamar tryMatch
	return nil
}

func (e *Exchange) tryMatch() {
	// TODO: enquanto melhor ask <= melhor bid, fazer match e distribuir revenue
	// Dica: pensa no que você fez no Challenge 11 — a lógica é a mesma,
	// só muda o que você está "comprando"
}

// RegisterMiner no contexto da exchange significa: colocar um Ask automaticamente.
func (e *Exchange) RegisterMiner(minerID string, hashrateTHs float64) error {
	// TODO: criar um Ask com preço padrão e chamar PlaceOrder
	return nil
}

func (e *Exchange) UnregisterMiner(minerID string) {
	// TODO: remover asks pendentes desse minerador
}

func (e *Exchange) TotalHashrate() float64 {
	// TODO
	return 0
}

func (e *Exchange) Revenue(minerID string) float64 {
	// TODO
	return 0
}
```

**pool_test.go**

```go
package main_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"miningpool/market"
	"miningpool/pool"
	"miningpool/protocol"
)

// TestConcurrentMiners garante que 50 mineradores conectando simultaneamente
// não causam race conditions no mapa de miners.
// Roda com: go test -race ./...
func TestConcurrentMiners(t *testing.T) {
	mp := market.NewMarketplace(0.05)
	srv := pool.NewServer(":0", mp) // porta 0 = OS escolhe porta livre

	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer srv.Shutdown(context.Background())

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// TODO: conectar, enviar Subscribe, verificar response
		}(i)
	}
	wg.Wait()
}

// TestShareValidation testa a validação de shares sem TCP.
func TestShareValidation(t *testing.T) {
	tests := []struct {
		hash       string
		difficulty uint64
		valid      bool
	}{
		{"0000abcd1234", 4, true},  // 4 leading zeros
		{"000abcd1234f", 4, false}, // só 3 leading zeros
		{"0000000012ab", 7, true},  // 7 leading zeros
	}

	for _, tc := range tests {
		got := protocol.ValidateShare(tc.hash, tc.difficulty)
		if got != tc.valid {
			t.Errorf("ValidateShare(%q, %d) = %v, want %v",
				tc.hash, tc.difficulty, got, tc.valid)
		}
	}
}

// TestMarketplaceTick verifica distribuição proporcional de revenue.
func TestMarketplaceTick(t *testing.T) {
	mp := market.NewMarketplace(1.0) // $1/TH/s para math simples

	mp.RegisterMiner("alice", 100) // alice tem 100 TH/s
	mp.RegisterMiner("bob", 300)   // bob tem 300 TH/s — 3x mais

	mp.Tick(1.0) // simula 1 hora

	// alice deve ter 25% do revenue total (100/400 * 1.0 * 400 = 100)
	// bob deve ter 75% do revenue total
	aliceRev := mp.Revenue("alice")
	bobRev := mp.Revenue("bob")

	if bobRev/aliceRev < 2.9 || bobRev/aliceRev > 3.1 {
		t.Errorf("proporção incorreta: alice=%.2f, bob=%.2f, ratio=%.2f",
			aliceRev, bobRev, bobRev/aliceRev)
	}
}

// TestBroadcastNoDeadlock garante que BroadcastJob não causa deadlock
// quando um minerador lento bloqueia.
func TestBroadcastNoDeadlock(t *testing.T) {
	// TODO: criar server com um minerador "lento" (buffer cheio),
	// chamar BroadcastJob, verificar que completa em < 1s
	done := make(chan struct{})
	go func() {
		// TODO: chamar BroadcastJob aqui
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(1 * time.Second):
		t.Fatal("BroadcastJob deadlocked com minerador lento")
	}
}

// TestExchangeMatching verifica que bids e asks fazem match corretamente.
func TestExchangeMatching(t *testing.T) {
	ex := market.NewExchange()

	// Bob quer comprar 100 TH/s por até $0.06
	ex.PlaceOrder(&market.Order{
		ID:          "bid-1",
		MinerID:     "buyer-bob",
		HashrateTHs: 100,
		PricePerTH:  0.06,
		Side:        market.Bid,
	})

	// Alice tem 100 TH/s e aceita $0.05
	ex.PlaceOrder(&market.Order{
		ID:          "ask-1",
		MinerID:     "alice",
		HashrateTHs: 100,
		PricePerTH:  0.05,
		Side:        market.Ask,
	})

	// Deve ter feito match — alice deve ter revenue > 0
	if ex.Revenue("alice") == 0 {
		t.Error("match não aconteceu: alice não recebeu revenue")
	}
}
```

**pool/dispatcher.go**

```go
package pool

import (
    "context"
    "fmt"
    "math/rand"
    "time"
    
    "challenge13/protocol"
)

// Dispatcher é responsável por gerar jobs de mineração e distribuí-los
// para todos os mineradores conectados via BroadcastJob do servidor.
// Ele é intencionalmente burro sobre o protocolo TCP — só conhece jobs.
type Dispatcher struct {
    server   *Server       // referência para chamar BroadcastJob
    interval time.Duration // intervalo entre novos jobs
    
    // currentDifficulty simula ajuste dinâmico de dificuldade.
    // Em Bitcoin real, a dificuldade é ajustada a cada 2016 blocos.
    currentDifficulty uint64
}

func NewDispatcher(server *Server, interval time.Duration, initialDifficulty uint64) *Dispatcher {
    return &Dispatcher{
        server:            server,
        interval:          interval,
        currentDifficulty: initialDifficulty,
    }
}

// Run inicia o loop do dispatcher. Deve ser chamado em uma goroutine separada.
// Para quando o context for cancelado — usa o mesmo padrão de shutdown do servidor.
func (d *Dispatcher) Run(ctx context.Context) {
    ticker := time.NewTicker(d.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            job := d.generateJob()
            d.server.BroadcastJob(job)
        }
    }
}

// generateJob cria um novo job de mineração com um JobID único e
// um PrevHash simulado. Em produção isso viria do nó Bitcoin completo.
func (d *Dispatcher) generateJob() *protocol.NotifyParams {
    return &protocol.NotifyParams{
        JobID:      fmt.Sprintf("job-%d", time.Now().UnixNano()),
        PrevHash:   fmt.Sprintf("%064x", rand.Uint64()), // hash simulado de 64 chars hex
        Difficulty: d.currentDifficulty,
        CleanJobs:  true, // descarta jobs anteriores — novo bloco chegou
    }
}
```

### Dicas

Começa pelo protocolo, `protocol/stratum.go`. Sempre começa pelo protocolo quando tem um challenge com comunicação estruturada. O motivo é que o protocolo é o contrato entre todas as outras partes. Se implementar o TCP server primeiro sem saber como é feito o parse das mensagens, não faz muito sentido.

Ordem que faz sentido: protocolo -> estrutura de dados de mercado (Marketplace primeiro, Exchange depois) -> servidor TCP -> integração na main.

## 15. Raft Leader Election

### Por que importa

Raft é um algoritmo de consenso, usado por exemplo no **etcd, Kafka, ScyllaDB, Consul, CockroachDB, TiKV** e praticamente todo sistema distribuído moderno. Quando empresas que usam sistemas distribuídos precisam de coordenação entre nós, quem deve ser o líder desses nós? Quem processa requests? Eles usam alguma variante disso (geralmente).

### Modelo mental

Pensa em uma eleição para o líder, com timeout. Cada nó começa como **Follower** esperando notícias do lider. Se ficar muito tempo em silência (election timeout), esse nó vira **Candidate** e pede votos. Se ganhar a maioria vira o **Leader** e começá a mandar heartbeats para manter o poder.

Os três papéis:

- **Follower**: modo passivo, reseta timer quando recebe heartbeat
- **Candidate**: modo de campanha, pede votos, pode virar Follower se encontrar termo maior
- **Leader**: manda heartbeat a cada X mas para previnir novas eleições

> Importante: os **Terms** (mandatos), o Raft opera em cima deles. Cada eleição incrementa o term. Mensagens com term antigo são ignoradas, isso previne líderes fantasmas, mortos voltando e causando caos no sistema

### Raft na Prática

```
Election Timeout: 150-300ms (randomizado para evitar eleições simultâneas)
Heartbeat Interval: 50ms (deve ser < election timeout)
Quorum: (N/2) + 1 votos para ganhar
```

Por que timeout randomizado: se todos os followers expirarem ao mesmo tempo, vira todo mundo candidate, e aí ninguém ganha maioria, e você fica em loop infinito. RAndomizar quebra a simetria.

### Dicas

Um erro comum nesse desafio vão ser **race conditions no currentTerm**, você vai ler o term sem lock em algum lugar. Lembra do snapshot pattern, captura o term dentro do lock e usa fora.

Segundo erro comum: **channel block**. Quando um nó está morto ou ocupado, você não pode bloquear tentando mandar mensagem para ele. Todo send em peer channel deve usar select -> default

Terceiro: **votedFor reset**. Quando um nó vê um term novo, ele precisa resetar `votedFor = -1` para poder votar naquele term específico. Se esquecer ninguém vai conseguir votar e a eleição vai travar.