# Synchronization Model

## 1. Concurrency Performance Is About Shared State

Concurrency primitives are often compared as if they were interchangeable:

```text
atomic
mutex
channel
```

They are not.

The deeper question is:

> What state is shared, who is allowed to modify it, and how often do participants need to coordinate?

A useful model is:

```text
shared mutable state
        ↓
coordination
        ↓
cache coherence / waiting / retry
        ↓
throughput and latency
```

---

## 2. Synchronization Has Multiple Costs

Synchronization can cost CPU in several ways:

- atomic instructions;
- cache-line ownership transfers;
- spinning;
- retries;
- goroutine parking/wakeup;
- scheduler work.

It can also cost latency:

- waiting behind a lock;
- queueing behind a single writer;
- batching delay.

The visible primitive is only one part of the cost.

---

## 3. Contention

Contention occurs when multiple participants need the same resource at overlapping times.

A mutex may be extremely cheap when uncontended.

The same mutex may dominate performance when many goroutines repeatedly enter a long critical section.

A useful mental approximation:

```text
contention pressure
≈ arrival rate × lock hold time
```

This is not a formal formula, but it explains why reducing hold time is often more effective than replacing the lock.

---

## 4. Blocking, Spinning, and Retrying

When synchronization fails immediately, the program can respond in different ways.

### Block / Park

Stop using CPU until progress is possible.

### Spin

Remain on CPU and repeatedly check.

### Retry CAS

Recompute and attempt a lock-free update again.

Each strategy is appropriate under different assumptions.

Short waits may justify limited spinning.

Long waits favor parking.

High retry rates can make lock-free code expensive.

---

## 5. Ownership

Synchronization exists because multiple actors share mutable state.

Reducing shared ownership can eliminate synchronization entirely.

Examples:

```text
global mutable map
→ sharded maps

shared counter
→ local counters + merge

mutable config
→ immutable snapshot + atomic pointer

many writers
→ single writer
```

Ownership design is therefore often a stronger optimization than primitive selection.

---

## 6. Communication vs Shared Memory

A mutex protects shared state.

A channel transfers values and can transfer ownership.

These are different models.

### Shared State

```text
G1 ─┐
G2 ─┼→ same object
G3 ─┘
```

requires synchronization.

### Ownership Transfer

```text
G1 owns object
   ↓ send
G2 owns object
```

can reduce shared mutation.

---

## 7. Fairness vs Throughput

Synchronization algorithms often trade fairness for throughput.

Strict handoff can reduce starvation but increase scheduling overhead.

Allowing an already-running goroutine to acquire a resource can improve throughput but make older waiters less favored.

Go's synchronization primitives contain such trade-offs internally.

This is why replacing standard primitives with custom logic is rarely trivial.

---

## 8. Tail Latency

Average synchronization cost can look small while tail latency grows.

For example:

```text
most Lock calls: immediate
rare Lock calls: wait milliseconds
```

Averages may hide this.

Concurrency performance should therefore consider:

- throughput;
- CPU;
- P50;
- P99;
- fairness.

---

## 9. Scheduler Interaction

Goroutines are scheduled on Go's runtime scheduler.

When a goroutine parks:

```text
G blocked
↓
P may run another G
```

This differs from the simplistic model:

```text
mutex wait
→ OS thread sleeps immediately
```

Runtime scheduling is part of synchronization behavior.

---

## 10. Scaling Curves

A synchronization primitive should be evaluated across concurrency levels.

Example:

```text
1 worker
2 workers
4 workers
8 workers
16 workers
32 workers
```

The important result is often the curve, not a single ns/op number.

---

## 11. Engineering Principle

Do not ask:

> Which primitive is fastest?

Ask:

> What coordination does this state require, and can the design reduce coordination before optimizing the primitive?
