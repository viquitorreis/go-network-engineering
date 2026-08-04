# XDP: FILTER/DROP BY CRITERIA

**Category**: Network programming / Kernel bypass
**Time**: 2h
**Builds on**: challenge 12 (XDP counts UDP packets by port), same toolchain, same .c, same Go loader structure

## What actually changes compared to challenge 12

Structurally, not much changes, and that's intentional — the roadmap calls this "closing the phase" because it's the same base, just flipping the switch from observing to acting. Two central changes:

1. **`XDP_PASS` stops being the only possible response.** In challenge 12's program, it always returned `XDP_PASS` (only counted, never interfered). Now, based on a criterion (a blocked port, for example), it returns **`XDP_DROP`** — the packet is discarded right there, in the driver, before costing any further processing in the kernel. This is literally the mechanism behind DDoS mitigation in production.
2. The blocking policy comes from a second eBPF map, written by Go — not hardcoded in C. This is the most important new piece today: until now, the earlier Go program only read one map (`port_count`). Today it also writes to a map (the list of blocked ports), and C reads that map to decide. This is two-way communication between userspace and kernel — before it was only kernel → userspace (count going up), now it's also userspace → kernel (policy coming down).

## What to build

1. Second map (`blocked_ports`, type `BPF_MAP_TYPE_HASH`, key `__u16` port, value `__u8` just as a presence flag) — if the port is in the map, it's blocked
2. In the `.c`: after extracting the destination port (already done in challenge 12), do a `bpf_map_lookup_elem` on this new map — if found, `return XDP_DROP`; if not found, continue into the normal counting flow and `XDP_PASS`
3. In Go: besides reading `port_count` periodically, add a way to write to `blocked_ports` (can be via a command-line flag, e.g. `./counter lo -block 9999`, or reading from a simple config file) — this uses `objs.BlockedPorts.Put(port, value)`, the inverse of the `Iterate()` already used

## Required work

- Packets on blocked ports are actually discarded — testable by comparing `curl`/`nc` on a blocked port (should fail/not respond) vs. a free port (should work normally)
- The block list is configurable at runtime, without needing to recompile the .c or restart the Go program — updates the map, immediate effect on the next packet
- Challenge 12's counter still only counts packets that got through (doesn't count what was dropped) — a design decision worth documenting: do you want to know "how much good traffic passed," or also "how much traffic was rejected"? (this changes whether you want 1 or 2 counters)
- Final report shows separately: allowed packets by port, blocked packets by port

## Bonus, if time allows

- Richer criteria than a fixed port: block by source IP range (CIDR), reusing the same second-map idea, just with the key becoming an IP prefix instead of a port
- Simple rate limiting: instead of binary blocking, allow N packets per port per second, using a timestamp inside the map as state

## What will be evaluated

Whether you understand the difference between "hardcoded decision in C, needs a recompile to change" (what you'd have if you wrote `if (port == 9999) return XDP_DROP` directly in the code) vs. "data-driven decision, C only consults a map that Go controls" (what the challenge asks for) — this second approach is the real pattern used in production (it's literally how Cilium/Katran do it: the userspace control plane decides policy, the kernel data plane just executes).

---

Docs reference:

`github.com/RaniaMidaoui/ebpf-pingkiller`: drops ICMP packets specifically, with a "KSP" (kernel space program, the .c) and a "USP" (user space program, the Go program using Cilium) conceptually well separated, the same way we structured this. Worth reading the repo's wiki to see how they split the responsibilities.

To go deeper on the concept of "XDP as a filtering layer" at real production level, Cilium's BPF and XDP Reference Guide (`docs.cilium.io/en/latest/reference-guides/bpf/`) has a hooks section explaining why XDP is the right point for this kind of decision (compared to running the same filter in `tc`, which runs later, with more overhead).