# Channels

## 1. Channels Are Communication Primitives

A channel is not simply a slower or faster mutex.

It combines several concepts:

- communication;
- synchronization;
- queueing;
- blocking;
- ownership transfer.

This makes channel performance inseparable from its semantics.

---

## 2. Shared-State Model

With a mutex:

```text
G1 ─┐
G2 ─┼→ shared mutable state
G3 ─┘
```

Synchronization protects the shared object.

---

## 3. Ownership-Transfer Model

With a channel:

```text
producer owns value
        ↓ send
consumer owns value
```

The program can reduce shared mutation by moving responsibility between goroutines.

This is often more important than the raw cost of send/receive.

---

## 4. Buffered Channels

A buffered channel creates a bounded queue.

This can decouple producer and consumer temporarily.

Benefits:

- absorb bursts;
- reduce direct handoff;
- provide backpressure.

Costs:

- queue memory;
- waiting when full;
- latency from batching/queueing.

---

## 5. Unbuffered Channels

An unbuffered send requires a matching receiver rendezvous.

This is useful when the synchronization itself is part of the design.

It is not equivalent to a lock-protected queue.

---

## 6. Backpressure

Channels can intentionally block producers.

This is often a feature.

Without backpressure, a system may move pressure into:

- unbounded memory;
- goroutine creation;
- downstream overload.

Performance engineering must therefore evaluate stability, not only throughput.

---

## 7. Channel vs Mutex

Use a mutex when the problem is:

> Several goroutines need coordinated access to shared state.

Use a channel when the problem is:

> Work or ownership should move between participants.

Forcing channel-based ownership around a simple protected map can create unnecessary complexity.

Forcing a mutex around a workflow that naturally transfers jobs can obscure the design.

---

## 8. Single-Writer Architecture

A channel often enables a single-writer model.

```text
workers
   ↓ events
aggregator goroutine
   ↓
owns mutable aggregate state
```

The aggregator can mutate its private state without locks.

The cost moves from shared synchronization to queueing and message passing.

---

## 9. Batching

A consumer can drain multiple channel items and process them as a batch.

This may improve throughput by reducing:

- per-item locking;
- syscall frequency;
- aggregation overhead.

But batching can increase individual latency.

---

## 10. Channel Contention

Channels can themselves become contended.

Symptoms include:

- many blocked senders;
- many blocked receivers;
- throughput collapse;
- scheduler activity.

A channel is not immune to shared coordination cost.

---

## 11. Diagnosing Channel Performance

Useful questions:

- Is the channel frequently full or empty?
- Are producers or consumers slower?
- Is the buffer hiding or solving the problem?
- Is one consumer a deliberate serialization point?
- Would sharding the queue improve throughput?
- Would batching reduce overhead?

---

## 12. Engineering Principle

Choose channels for communication and ownership semantics.

Do not choose them from a single-operation benchmark against mutexes.
