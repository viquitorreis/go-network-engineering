# AF_XDP SOCKET: FIRST ATTACH

**Category**: Networking / Kernel Bypass
**Time**: 3h
**Builds on**: 13-xdp_filters_drop_by_criteria (same XDP program hook, now redirecting to a socket instead of counting/dropping)

## Study before (10-15min)

The difference between plain XDP (runs inside the kernel, decides DROP/PASS/etc **before the packet enters the network stack**) and AF_XDP (a special socket that receives raw packets straight from the NIC driver, via a kernel/userspace shared buffer called **UMEM**, bypassing the traditional network stack). AF_XDP still needs an attached XDP program to decide *which* packets get redirected to the socket (via XDP_REDIRECT) the two work together, the XDP program decides, AF_XDP delivers. Today we're only setting up the basic socket and the attach, no redirect yet (that's the next challenge).

## Context

This software is a Go program that **intercepts network packets before they pass through the operating system's normal stack**, using two mechanisms:
- A small piece of software that runs inside the kernel (XDP program, compiled from C). It looks at every packet arriving on a specific network interface and decides "does this have an AF_XDP socket waiting for it? If so, send it straight there, skipping the traditional transport layer (TCP/IP, regular sockets, etc.)"
- A special socket created in the Go program (userspace) that receives these redirected packets through a memory region shared with the kernel (the UMEM), with no extra data copy between kernel and application that copy is normally what makes high-volume packet processing slow.

Resulting data path: network driver -> Kernel -> XDP Program -> AF_XDP socket -> Go code

Challenges 12 and 13 processed packets entirely inside the kernel (counting, dropping). That's fast but limited you can't inspect complex payloads or process in userspace without paying the copy cost through the normal stack. AF_XDP exists for that: kernel-bypass throughput, with the flexibility of processing in Go in userspace.

## What to build

1. UMEM: shared memory region registered with the kernel, divided into fixed-size frames, plus the 4 control rings (fill, completion, rx, tx)
2. AF_XDP socket created and bound to a specific network interface + queue id
3. Socket attached to the UMEM via **setsockopt**
4. Basic loop: populate the fill ring with available frames, poll the socket, read the rx ring when a packet arrives, print size and first bytes of the received packet

The ring responsible for telling the kernel "here's a free frame for you to fill" is the **fill ring**. You "fill" the ring buffer with empty buffer addresses, the kernel consumes from that ring, writes the received packet into it, and hands the filled slot back via the **rx ring** (that's what we actually read to get the packet). The completion and tx rings are the symmetric pair on the *sending* side (unused today, focus is receive-only).

**Two differences from challenges 12/13**:

- No Ethernet/IP/UDP parsing here. Challenge 12 needed to inspect the packet to decide how to count it by port. Here the decision is just "does this queue id have a socket registered?", so it doesn't even touch `ctx->data`/`data_end`. This also means: every packet type landing on this queue gets redirected, not just UDP.
- No separate `qidconf_map` (which `asavie/xdp` uses in its own repo) a single `xsks_map` is enough, since an entry existing for that queue already means "there's a socket here, redirect".

## Required

- Focus only on the attach and the first packet arriving through the socket, no XDP-program-driven redirect logic yet (test with traffic that would already hit the host anyway, e.g. loopback via a veth pair, if no physical interface is safely available to test with)
- Document the 4 rings (fill, completion, rx, tx) and each one's role this is the conceptual base for the zero-copy challenge that comes later
- Will need to run as root or with CAP_NET_RAW/CAP_BPF capability document this in the README, since it differs from previous challenges

**Bonus (if time allows)**

- Measure and document how many packets per second you can read from the rx ring in a simple local test, as a baseline to compare against the zero-copy benchmark in a later challenge

What will be observed: whether the 4 rings were understood in terms of what each one does (not just copied from an example), and whether the socket actually receives raw packets, not simulated ones

**Important notes**

- The **XDP program** (runs inside the kernel, decides the redirect) needs to be written in C/C++ (or Rust, etc.) and compiled to eBPF bytecode via `clang`/LLVM. This can't be written in pure Go, since the language needs an OS to run on.
- The **AF_XDP socket** itself (userspace, reading from the rx ring) is pure Go, using a lib like `github.com/cilium/ebpf` or `github.com/asavie/xdp` no C needed for that part.
- In other words: a small amount of C is needed for the XDP program that does the redirect, but most of the work (socket setup, UMEM, rings, packet processing) is Go the C part follows the same pattern already used in challenges 12/13.

## References

- Pure Go lib wrapping an AF_XDP socket, UMEM, and the 4 rings. Has `examples/` with `sendudp.go` and `senddnsqueries.go` ready to run and modify:
  https://github.com/asavie/xdp

- Explains the 4 rings, copy mode vs zero-copy, and why a minimal XDP program is needed for the redirect:
  https://github.com/xdp-project/xdp-tutorial/blob/main/advanced03-AF_XDP/README.org

- XDP + Go + bpf2go example (no manual C wrangling beyond the `.c` file itself, compiled via `go generate`, everything else is Go):
  https://github.com/cilium/ebpf/blob/main/examples/xdp/main.go

- Official kernel doc on AF_XDP, covering the 4 rings (fill, completion, rx, tx) and the `XSKMAP + XDP_REDIRECT` flow:
  https://docs.kernel.org/networking/af_xdp.html

## What was actually built

Rather than the fixed program from `asavie/xdp`, the XDP program here was written in C and compiled via `bpf2go` (same pattern as challenges 12/13), giving full control over the redirect logic instead of relying on a hardcoded bytecode program. The AF_XDP socket (UMEM registration, ring setup, fill/receive loop) was hand-written in Go using raw syscalls (`setsockopt`/`getsockopt`/`mmap`/`bind`) against `golang.org/x/sys/unix` types, rather than depending on a third-party socket wrapper closer to how this would be done in a production system that needs custom packet filtering logic.

---
First step: before any socket code, which of the 4 rings (fill, completion, rx, tx) is responsible for telling the kernel "here's a free buffer frame for you to fill with the next incoming packet"? Think about this first it's the piece that usually causes confusion the first time working with AF_XDP.