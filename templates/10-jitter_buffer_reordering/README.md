# JITTER BUFFER: REORDERING

**Category**: Streaming / Network programming, raw
**Time**: 4h (15min theory + ~3h45 challenge)
**Builds on**: challenge 26 (RTP-style packetizer with seq + timestamp)

## Study before starting (15 min)

Core focus: jitter is the variation in packet arrival timing. Even if the sender transmits at perfectly regular intervals (20ms, 20ms, 20ms...), the network does NOT deliver with that same regularity packets can take different paths and arrive at different times. One packet might arrive in 18ms, the next in 35ms, the next in 15ms. The network "stutters and speeds up" (congestion and decongestion, among other factors). If the receiver played each packet as soon as it arrived, audio/video would sound robotic and choppy, because playback timing would be hostage to network variation instead of the original capture timing.

Solution: **jitter buffer**.

Instead of playing a packet the moment it arrives, you hold it for a fixed amount of time (the "buffer delay," typically 20-200ms depending on the application) before releasing it for playback. This creates a safety margin: if a packet arrives a bit late, there's still time to reorder it before the moment it would actually need to play. The tradeoff is direct: bigger buffer = more jitter tolerance, but more total latency (you're always deliberately "delaying" playback). This is why real-time voice calls use small buffers (20-60ms, latency matters more than perfect smoothness) while pre-recorded video streaming uses buffers of several seconds (smoothness matters more than latency).

Review this mentally: the Timestamp you already have on the RTP-style packet is exactly what lets you calculate "when this packet should play." The jitter buffer uses that to decide release order, not network arrival order.

## Context

You already have the receiver detecting gaps and out-of-order packets (challenge 26), but today it only logs that, without doing anything useful with the information though that groundwork is what makes today possible. Now you're building the component that actually reorders what arrives, releasing packets into a playback queue in the correct order (by seq/timestamp), even when the network delivers them out of order, with a limit on how long to wait before giving up on a packet that's too late.

## What to build

A buffer with a configurable time window: a structure that receives packets as they arrive (possibly out of order) and holds them for a period before releasing them
Release logic based on timestamp, not arrival order: the buffer needs to know "what's the next packet that should play" based on the expected Timestamp/SequenceNumber, not on who arrived first on the network.

Timeout for a late packet: if the expected sequence number doesn't show up within the buffer's window, the player can't stall waiting forever after a deadline, it skips that packet (marks it as lost) and releases the next one it already has
Integration with the challenge 26 receiver: instead of just logging gaps/out-of-order arrivals, the receiver now feeds this buffer, and a simulated "player" (can just be a log.Println saying "playing frame N") consumes from the buffer in the correct order

##  Required work

Buffer with a configurable time window (e.g. 100ms); packets are only released after having "waited" that long since original capture (based on Timestamp)
Reordering: if packets 5, 7, 6 arrive on the network in that order, the buffer releases 5, 6, 7 (correct order), not arrival order
Per-packet timeout: if the expected sequence number doesn't show up within the window's deadline, mark it as lost and move on without stalling the player
Final report: how many packets were successfully reordered, how many were definitively lost (never arrived within the window), and effective playback latency (time between capture and "playback")

## Bonus (if time allows)

Adaptive buffer: instead of a fixed window, dynamically adjust the window size based on recently observed jitter (shrink it when the network is stable to reduce latency; grow it when it's unstable to tolerate more)
A real jitter metric, in the format RTCP uses (RFC 3550): a moving average of the difference between consecutive arrival intervals
What will be evaluated

How you structure the waiting queue (a heap by timestamp? an ordered list? something else?) so that inserting an out-of-order packet and figuring out "what's next to release" are both efficient and how cleanly you decide when to give up waiting on a packet without stalling the player indefinitely.