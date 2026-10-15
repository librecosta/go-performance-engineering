# Ownership-Oriented Design

## 1. The Best Synchronization Is Often No Synchronization

Synchronization exists because mutable state is shared.

If ownership becomes exclusive, synchronization can disappear.

This is one of the most powerful concurrency transformations available.

---

## 2. Shared Mutable State

A common design:

```text
G1 ─┐
G2 ─┼→ mutable structure
G3 ─┘
```

requires:

- mutex;
- atomics;
- channel serialization;
- another coordination mechanism.

As concurrency grows, the shared state can become the bottleneck.

---

## 3. Partition Ownership

Instead of sharing one structure:

```text
all keys
  ↓
one map
```

partition:

```text
hash(key)
  ↓
shard 0
shard 1
shard 2
...
```

Each shard has fewer contenders.

This is lock striping/sharding.

The trade-offs include:

- more structures;
- more complex iteration;
- rebalancing;
- shard-count decisions.

---

## 4. Local Accumulation

Global counters are common hot spots.

Instead:

```text
worker
  ↓
local count
```

then periodically:

```text
merge
```

This dramatically reduces synchronization frequency.

The trade-off is delayed aggregation.

This is usually acceptable for metrics and statistics.

---

## 5. Single Writer

A single-writer architecture gives one goroutine exclusive ownership of mutable state.

```text
workers
   ↓ messages
aggregator
   ↓
owns state
```

Within the aggregator:

```go
stats[key]++
```

may need no mutex.

The cost becomes:

- queueing;
- channel communication;
- potential serialization.

This is attractive when mutation is cheap and coordination dominates.

---

## 6. Immutable Snapshot

Read-mostly data can be represented as immutable versions.

Writer:

```text
build new snapshot
↓
publish pointer
```

Readers:

```text
load snapshot
↓
read without mutation
```

This can eliminate reader locking.

The cost moves to:

- copying/building new snapshots;
- memory for old/new versions during transition.

---

## 7. Batching

Instead of synchronizing each operation:

```text
lock
update one
unlock
```

batch:

```text
prepare many updates
↓
lock
apply batch
unlock
```

Benefits:

- fewer lock operations;
- less handoff;
- better amortization.

Costs:

- temporary buffers;
- delayed visibility;
- potentially higher individual latency.

---

## 8. Message Passing

Message passing can express ownership transfer explicitly.

This helps reduce ambiguous shared mutation.

However, message passing is not free.

Potential costs:

- allocation;
- channel contention;
- copying;
- queueing.

The design should be chosen for ownership clarity first, then benchmarked.

---

## 9. Read Replication

Some workloads can replicate read-only state across workers.

This reduces shared reads and synchronization.

The trade-offs include:

- replication memory;
- update propagation;
- consistency delay.

This is another example of:

```text
memory
→ lower coordination
```

---

## 10. Ownership and Cache Locality

Ownership design also affects hardware.

If one worker repeatedly owns the same state, its cache lines may remain local to one core longer.

If many workers constantly write the same state, the lines bounce between cores.

Thus ownership reduces both:

- logical synchronization;
- physical coherence traffic.

---

## 11. Ownership Boundaries

A useful design question is:

> Which goroutine is responsible for mutating this state?

If the answer is:

> everyone,

the structure deserves scrutiny.

Better answers often include:

- one worker;
- one shard owner;
- one stage;
- one immutable version.

---

## 12. When Sharing Is Necessary

Some states are inherently shared.

Examples:

- coordination flags;
- global limits;
- connection registries.

The goal is not to eliminate all sharing.

The goal is to minimize the frequency and scope of coordination.

---

## 13. Ownership as an Optimization Strategy

Primitive-level thinking:

```text
Mutex vs Atomic vs Channel?
```

Ownership-level thinking:

```text
Why do these goroutines need to mutate the same state?
```

The second question often leads to larger and more durable performance improvements.

---

## 14. Engineering Principle

When contention is severe, stop optimizing the synchronization primitive and redesign the ownership topology.
