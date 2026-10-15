# Atomic Operations

## 1. What Atomic Means

Atomic operations provide indivisible memory operations with synchronization semantics.

Examples:

- load;
- store;
- swap;
- add;
- compare-and-swap.

Atomic does not mean free.

It means concurrent access has a defined synchronization behavior.

---

## 2. Read vs Write

An atomic load can be relatively cheap when many cores only read shared state.

A write or read-modify-write operation is more expensive because cache-line ownership must be coordinated.

This distinction is critical.

```text
atomic.Load
```

and:

```text
atomic.Add
```

belong to different performance categories.

---

## 3. Read-Modify-Write

Operations such as:

```go
counter.Add(1)
```

must read the old value and publish a new value atomically.

On a shared counter, many cores compete for the same cache line.

This creates hardware serialization even though no mutex exists.

---

## 4. True Sharing

A global counter:

```go
var requests atomic.Uint64
```

updated by every worker creates true sharing.

```text
Core 0 writes
Core 1 writes
Core 2 writes
...
```

The line must repeatedly change ownership.

This can collapse scalability.

---

## 5. Sharded Counters

A common transformation is:

```text
one global counter
→ many local/sharded counters
```

Each worker updates a different location.

Aggregation occurs less frequently.

This trades:

```text
cheap writes
```

for:

```text
more expensive reads/aggregation
```

which is often favorable for write-heavy metrics.

---

## 6. Immutable Snapshot

Atomics are especially effective for read-mostly state.

Example:

```go
type Store struct {
    config atomic.Pointer[Config]
}
```

Readers:

```go
cfg := store.config.Load()
```

Writers build a complete new immutable configuration and publish it once.

This avoids reader locks and reader-side RMW operations.

---

## 7. Compare-and-Swap

CAS succeeds only if the current value still matches the expected value.

Conceptually:

```text
load old
compute new
CAS(old, new)
```

Under low contention this can be efficient.

Under high contention many goroutines may compute values that are immediately discarded.

---

## 8. Retry Amplification

Suppose 32 goroutines race on one CAS.

One succeeds.

31 fail.

If they retry immediately:

```text
1 useful update
31 failed attempts
```

The failure work can dominate CPU and increase cache traffic.

This is retry amplification.

---

## 9. Atomic Does Not Compose Automatically

Two atomic fields:

```go
balance atomic.Int64
version atomic.Uint64
```

do not form one atomic state transition.

A reader can observe:

```text
new balance
old version
```

If values belong to one invariant, use:

- mutex;
- immutable snapshot;
- carefully designed packed state.

---

## 10. Memory Ordering

Atomic operations also impose memory-ordering semantics.

The exact generated instructions differ across architectures.

Application code should reason from Go's memory model, not from one CPU's instruction sequence.

---

## 11. Atomic vs Mutex

Atomic is attractive when:

- state fits a simple atomic operation;
- invariant is small;
- contention is low or read-mostly.

Mutex is attractive when:

- multiple fields must change together;
- critical sections are complex;
- retries would duplicate expensive work.

The choice is semantic first, performance second.

---

## 12. Diagnosing Atomic Hotspots

A hot atomic in CPU profile may indicate:

- true sharing;
- false sharing;
- retry loops;
- overly frequent synchronization.

The solution may be:

- sharding;
- batching;
- local accumulation;
- ownership redesign.

Replacing one atomic primitive with another rarely solves the underlying topology.

---

## 13. Engineering Principle

Atomic operations remove software locking, not hardware coordination.

When many cores write one location, cache coherence becomes the lock.
