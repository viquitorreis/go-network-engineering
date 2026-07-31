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