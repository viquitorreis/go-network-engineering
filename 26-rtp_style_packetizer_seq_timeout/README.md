# RTP-STYLE PACKETIZER: SEQ + TIMESTAMP

**Category:** Streaming / Network programming, raw
**Time:** 2h (15min theory + ~1h45 challenge)
**Builds on:** challenges 23/24 (raw UDP + reliable UDP), but pay attention here:
this challenge does not use reliability. It's an important shift in mental model,
see below.

## Study before starting (15 min)

Core focus: **RTP (Real-time Transport Protocol) is not reliable, and that's
deliberate.** In the reliable UDP challenge, losing a message was unacceptable,
you retransmitted until delivery was guaranteed. In real-time media streaming
(audio, video, game state), the scenario is the opposite: if an audio packet gets
lost, retransmitting it doesn't help at all, by the time the retransmission would
arrive, the moment to play that audio has already passed. It's better to drop the
packet and move on than to stall waiting for something that's useless by the time
it arrives late. This is the fundamental difference between **reliability**
(guaranteeing everything arrives, no matter when) and **timeliness** (guaranteeing
things arrive on time, accepting that some won't arrive at all).

RTP (used by WebRTC, VoIP, live streaming) is designed around that philosophy:
every packet carries a **sequence number** (to detect loss and reorder anything
that arrives out of order) and a **timestamp** (to know when that data was
supposed to be "played," not just when it arrived).

Think this through: a 20ms audio packet captured at timestamp X needs to play 20ms
after the previous packet, even if it arrived on the network out of order or with
variable delay (jitter). The timestamp is what lets the receiver know "this should
play here," independent of network arrival order.

## Context

You're building the transport layer for a simple audio streaming system (think a
simplified voice call).

- The sender captures "frames" of audio periodically and sends them over raw UDP
- The receiver needs to reconstruct the correct sequence to play frames in the
  right order, even when the network delivers them out of order or drops some

This is different from the reliable UDP challenge: here **there is no
retransmission**. The goal isn't to guarantee everything arrives, it's to give the
receiver the **information it needs** (seq + timestamp) to decide what to do with
whatever did arrive.

## What to build

**1. Simplified RTP-style packet format**
- Every datagram carries at minimum a `SequenceNumber` (uint16 or uint32,
  increments on every packet sent, regardless of loss)
- A `Timestamp` (when that frame was "captured," in consistent units, can be an
  incrementing counter simulating audio samples, doesn't need to be real wall
  clock time)
- A `Payload` (the data itself, can be simulated, like a number identifying which
  "frame" it is, doesn't need to be real audio)

**2. Sender**
- Generates frames at regular intervals (e.g. every 20ms, simulating audio
  capture)
- Sends each one over raw UDP, with no confirmation whatsoever, pure
  fire-and-forget, unlike the reliable client built earlier

**3. Receiver**
- Receives packets as they arrive (which can be out of order, since UDP doesn't
  guarantee ordering), and needs to **detect** two things from the
  `SequenceNumber`: packets that arrived out of order (a sequence number lower
  than the last one already processed) and packets that never arrived (gaps in
  the sequence)

**4. Session report**
At the end (or periodically), the receiver reports: how many packets it received,
how many gaps it detected (sequence numbers that never arrived), how many arrived
out of order, and the timing variation between arrival and what the timestamp
expected. This is a preview of the **jitter** concept, which gets covered in more
depth later.

## Required work

- A packet with `SequenceNumber`, `Timestamp`, and `Payload`, simple binary
  format (can reuse the framing pattern from earlier challenges, or simplify it
  for this case since there's no need for a command like MSG/ACK)
- Sender sends at regular intervals (ticker), without waiting for any response
- Receiver detects gaps in the sequence (skipped sequence numbers), doesn't try
  to recover them, just detects and reports
- Receiver detects out-of-order packets (a sequence number lower than the highest
  one seen so far)
- No retransmission, no ACK, this is deliberate, don't forget to leave it out
- Final report with: total sent (known by the sender), total received, observed
  loss rate, how many arrived out of order

## Bonus (if time allows)

- Simulate packet loss along the way (reuse the `rand.Float64() < lossRate`
  logic from earlier challenges), to see gap detection working for real, not just
  in theory
- Calculate a simple jitter metric: for each packet, compare the expected
  interval between consecutive timestamps against the actual arrival interval,
  this is the seed of what a later jitter buffer challenge will need

## What will be evaluated

How you design gap/out-of-order detection using only the `SequenceNumber`
(without any heavy state structure, this should be much simpler than the
`pending map` built for reliable UDP, since there's no retry waiting on a
response here), and whether you resist the temptation to sneak reliability back
in. This is a common mistake: people who just implemented ACK/retry tend to want
to reuse that pattern here, but that would defeat the entire point of RTP.

---

First step: think about how the `SequenceNumber` alone already gives you gap
detection for free. If the receiver only stores **one number** (the last sequence
number seen), what simple operation on that number already reveals "how many
packets did I lose between the last one and the one that just arrived"?