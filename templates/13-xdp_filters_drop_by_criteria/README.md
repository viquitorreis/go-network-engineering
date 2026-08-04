# XDP: PROGRAM COUNTS PACKETS BY PORT

**Category**: Network programming / Kernel bypass
**Time**: 4h (30min theory + ~3h30 challenge — today the theory weighs more than usual, it's genuinely new content)
**Builds on**: conceptually nothing from earlier challenges (it's a layer below everything done so far) — but Monday you should have already read about the packet's path through the kernel + set up the toolchain, so part of this is already familiar

## Study before

### What eBPF is

Well, first it's worth reading about what eBPF is: https://ebpf.io/what-is-ebpf/

Summing up eBPF in one sentence: it's a way to **run programs inside the kernel** safely, without needing to recompile the kernel or write a traditional kernel module (which is dangerous — a bug there can take down the whole machine). The kernel runs a **verifier** that analyzes your program before accepting to load it, rejecting infinite loops, out-of-bounds memory access, anything that could hang or crash the system. This is what makes eBPF safe enough to run third-party code inside the kernel.

### What XDP specifically is

XDP is one of the possible attachment points for an eBPF program — specifically, the **earliest possible** point in a network packet's path: **before** the kernel even allocates the `sk_buff` structure (the internal packet representation the kernel uses through the rest of the network stack). XDP runs literally in the **network card driver**, at the moment the packet has just physically arrived.

**Why this matters for performance**: processing a packet in XDP costs **orders of magnitude less** than letting it climb the kernel's normal network stack (which allocates `sk_buff`, goes through netfilter/iptables, routing, etc — all of this **before** any userspace application even knows the packet exists). This is why XDP is used in production for DDoS mitigation, load balancing (Facebook/Meta uses this in Katran), and any scenario that needs to decide "process, drop, or redirect" a large volume of packets with the least possible overhead.

What needs to be understood, and why it's different from everything done so far: the program that runs **inside the kernel** is written in a restricted subset of C, compiled by `clang`/LLVM into eBPF bytecode — it's not full C, no heap, no arbitrary function calls, loops need a statically provable bound. The program that runs in **userspace** (which loads that bytecode into the kernel, attaches it to a network interface, and reads the results) is real Go, using cilium's ebpf lib `github.com/cilium/ebpf`. We're not going to write Rust or C++ for this, just a small restricted piece of C, and the rest stays Go.

**How kernel and userspace talk to each other, the piece that closes the puzzle**: eBPF **maps** are data structures (essentially hash maps or arrays) that both the C program in the kernel and the Go program in userspace can access. This is how you'll wire the pieces together: the C program, running for every packet that arrives, increments a counter inside a map, indexed by destination port. The Go program, running normally in userspace, **reads** that same map periodically — no network communication between the two at all, it's shared memory managed by the kernel.

## Context

You're building the most basic piece of any high-performance network observability/mitigation system: counting how many packets arrive per destination port, without that counting process itself becoming a bottleneck. This is literally the first step before anything more sophisticated (filtering, dropping, redirecting) that we'll build over the next few days this week.

## What to build

1. **XDP program in C**: for every packet received, manually parses the Ethernet → IP → TCP/UDP header (no library at all — it's raw byte parsing, same as what was already done with binary framing, just now in C and on top of a real network packet format) to extract the destination port.
2. **An eBPF map** (type `BPF_MAP_TYPE_HASH` or `BPF_MAP_TYPE_ARRAY`) indexed by port, holding the count of packets seen.
3. **Go loader** using `cilium/ebpf` that compiles/loads the bytecode, attaches the program to a network interface (can be `lo`, loopback, to test locally without needing real network traffic), and periodically reads the map and prints the counts by port.

## Required work

- XDP program compiles and loads without a verifier error (this alone is already a win today — the verifier rejects a lot of things that would look valid in normal C)
- Correct count by port, testable by sending real traffic (e.g. `curl localhost:PORT` from another terminal while XDP is attached to `lo`)
- Clean Go loader: loads, attaches, but also **detaches** on shutdown (Ctrl+C) — leaving an XDP program "stuck" on an interface after your Go process has died is a real operational problem, not just unfinished polish
- Today's XDP program decision is always `XDP_PASS` (lets the packet continue normally to the rest of the stack) — today is just about **counting packets**, not interfering. That comes later.

## Bonus (if time allows)

- Also split the count by protocol (TCP vs UDP), not just port
- Expose the metrics via a simple HTTP endpoint instead of just printing to the terminal

## What will be evaluated

Whether you understand the difference between where the C code runs (inside the kernel, restricted, no exceptions, no heap) and where the Go code runs (normal userspace) — and whether the communication between the two happens only via the eBPF map, without inventing any other channel.

---

First step, before writing any C: confirm you have the clang toolchain, llvm, kernel headers (`linux-headers` for your distro), and the `cilium/ebpf` lib already downloaded. Run `clang -target bpf -c teste.c -o teste.o` on an empty C file just to confirm the compiler can generate BPF bytecode before writing any real logic. If that already works, you're ready to start parsing the packet — if not, that's the first obstacle to solve before anything else.

There are code examples like this in the eBPF repo: `github.com/cilium/ebpf/tree/main/examples/xdp`

At `ebpf-go.dev/guides/getting-started/` there's an annotated, clickable walkthrough explaining line by line what `SEC("xdp")`, `SEC(".maps")`, and the rest of the C means. This is the best place to really understand the mechanics, not just copy.