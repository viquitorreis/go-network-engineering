# FEC (XOR): SEQUENCE PACKETS + PARITY + RECONSTRUCTION

**Category**: Streaming / Network programming raw
**Time**: 2h (15min theory + ~1h45 challenge)
**Builds on**: challenge 09 (RTP-style packetizer with seq + timestamp). Reuses the same packet format with seq+timestamp

## Study

### What is FEC (Forward Error Correction) via XOR

Worth remembering: the difference between **reliability** (ACK+retransmit, challenge 08) and **timeliness** (RTP, challenge 09), FEC is a **third strategy for transport error control**.

FEC solves a problem neither of the other two solves well: when you can't wait for a retransmit's round trip (something sent in real time), but you also can't simply accept the loss.

The core idea: instead of reacting to loss after it happens (retransmitting), you send redundancy along with the data, so the receiver can reconstruct a lost packet without needing to ask again. This avoids the cost of an entire round trip (which retransmit always pays) — the correction is already embedded in what arrived.

### Why use XOR?

XOR has a very useful mathematical property: if you have:

A XOR B XOR C = P

Where P = the parity packet, and you lose any **one** of the four (say, `B`), you can recover it by doing `B = A XOR C XOR P`. This is literally RAID5 applied to network packets instead of disks, same principle but a different domain.

**Trade-off**: you send 1 parity packet for every N data packets (e.g. 1 for every 4). This is **constant and predictable** bandwidth overhead, paid ALWAYS even when there's no loss at all in the communication.

Compared to retransmit: there, you only pay the cost **when** loss happens (but you pay it in latency, not bandwidth). FEC (proactive, Forward Error Correction) trades extra bandwidth for protection against loss **without** waiting for a retransmit. Great for real time, where 1 lost packet without correction means a visible/audible artifact, but a retransmit would arrive too late.

## Context:

Today the audio streaming system (challenges 9 and 10) only detects and reorders — when a packet is truly lost, it stays lost. Today we add a layer that **recovers** part of those losses without asking permission, by sending a parity packet for every group of N data packets.

## What to build:

1. **Grouping into FEC blocks:**: the sender groups data packets into fixed groups (e.g. 4 data packets), calculates the XOR of all of them, and sends that result as the **5th packet** (the parity one), identified by a field indicating "this is parity for group X"
2. **Continuous loss simulation**: (reuses `rand.Float64() < lossRate` from earlier challenges)
3. **Reconstruction on the receiver side**: if exactly 1 packet out of the group of 5 (4 data + 1 parity) is lost, the receiver reconstructs it by XOR-ing all the others it received — if 2 or more are lost in the same group, simple XOR recovers nothing (a real limitation, must be reported as definitive loss)
4. **Report**: how many packets were recovered via FEC vs. how many were definitive loss (group with 2+ losses) — and the real bandwidth overhead (how many parity packets were sent in total)

## Required work

- Configurable group size (e.g. 4 data + 1 parity)
- XOR correctly calculated over the payload of the group's packets
- Reconstruction working when exactly 1 packet from the group is missing
- Honest detection of "not recoverable" when 2+ are missing in the same group
- Benchmark: effective recovery rate vs. configured loss rate, across different group sizes (bigger group = less bandwidth overhead, but more vulnerable to double loss in the same group)

## Bonus (if time allows):

Compare real bandwidth overhead (% of extra packets sent) between FEC and the retransmit cost from challenge 08, under the same loss scenario — which strategy "costs more" under each loss regime.

**What will be evaluated**:

whether you understand simple XOR's structural limit (recovers 1 loss per group, no more) and design the group size as an explicit trade-off (bandwidth vs. robustness), not as an arbitrary number.

## Benchmarks

Machine:

goos: linux
goarch: amd64
pkg: feq_parity_packets/receptor
cpu: 12th Gen Intel(R) Core(TM) i5-1235U

| Group | Loss | Overhead | Definitive Loss | Recovered |
|---|---|---|---|---|
| 4 | 5% | 25.00% | 3.00% | 14.67% |
| 4 | 10% | 25.00% | 6.67% | 30.33% |
| 4 | 20% | 25.00% | 21.33% | 40.33% |
| 8 | 5% | 12.50% | 9.33% | 28.67% |
| 8 | 10% | 12.50% | 27.00% | 31.67% |
| 8 | 20% | 12.50% | 57.00% | 28.67% |
| 16 | 5% | 6.25% | 26.33% | 33.67% |
| 16 | 10% | 6.25% | 57.67% | 26.00% |
| 16 | 20% | 6.25% | 90.00% | 7.33% |

### The intuition behind why "recovered %" peaks and then drops

`Recovered` only happens in the very specific case of **exactly one packet going missing**. With a small group (few packets sent), the chance of exactly 1 going missing is low simply because there are few members involved, but as the group grows, there are more **chances** for exactly one to go missing (more "attempts"), so the recovery rate goes up. At the same time, though, a bigger group also increases the chance of **two or more** going missing, and that effect grows faster. At some point, the chance of 2+ overtakes the chance of "exactly 1," and from there on growing the group only makes things worse: more definitive loss, less recovery.

You can see this in the numbers: at 20% loss, `recovered` goes from 40.33% (group 4) to 28.67% (group 8) and collapses to 7.33% (group 16) — the peak already happened between 4 and 8, and it's downhill from there.

### There isn't just one direction to the trade-off

The trade-off depends on several factors, like the network's expected loss rate.

- **Good network (5% loss):** group 16 already loses 26.33% definitively — that's surprisingly bad even on a "good" network, because the denominator (16 packets) is too large. Group 4 still only loses 3% definitively, and pays more overhead (25%) than group 16 (6.25%). So even on a good network, group 4 is strictly safer, just more expensive in bandwidth (expected given the approach taken).
- **Bad network (20% loss):** the difference becomes stark — group 4 loses 21.33% definitively, group 16 loses 90%. Under that condition, a large group is nearly useless.

### Conclusion

Small groups are always more robust. Large groups are always more bandwidth-efficient. No group size wins on both fronts at once with the approach taken here. The right choice depends on which resource is scarcer in the real scenario (bandwidth or loss tolerance), and this benchmark is the tool to decide that.

### Possible Improvements

- **Reed-Solomon instead of single-parity XOR.** XOR is really the special case of Reed-Solomon with just 1 parity packet, which is why it can only recover exactly 1 loss per group. Using `k` data packets + `m` parity packets (Reed-Solomon) would tolerate any `m` losses per group, at the cost of Galois Field arithmetic, which is meaningfully more expensive than a plain XOR operation on the CPU.
- **Interleaving.** This benchmark assumes independent, random packet loss. Real network loss is often bursty (a congested router drops several consecutive packets at once). If a FEC group is made of consecutive packets, a single burst can wipe out more than one packet from the same group. Interleaving spreads consecutive packets across different groups, so a burst hits multiple groups lightly instead of one group heavily.