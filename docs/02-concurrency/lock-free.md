# Lock-Free Algorithms

## 1. Lock-Free Is a Progress Property

"Lock-free" is often misunderstood as:

> no locks, therefore faster.

That is incorrect.

Lock-free describes how an algorithm makes progress under concurrency.

It does not guarantee low latency or low CPU use.

---

## 2. Progress Classes

A simplified hierarchy:

### Blocking

One stalled participant can prevent others from progressing.

### Obstruction-Free

An operation completes if it eventually runs without interference.

### Lock-Free

The system as a whole guarantees that some operation continues to complete.

### Wait-Free

Every operation completes in a bounded number of steps.

These are correctness/progress properties, not benchmark rankings.

---

## 3. CAS Loop

A common lock-free pattern:

```go
for {
    old := state.Load()
    next := update(old)

    if state.CompareAndSwap(old, next) {
        return
    }
}
```

The linearization point is often the successful CAS.

Before that moment, the operation has not logically committed.

---

## 4. Retry Cost

When CAS fails:

- previous computation may be discarded;
- state must be reloaded;
- update may need recomputation;
- another RMW attempt hits the same shared line.

Under high contention, retries can dominate.

---

## 5. Retry Amplification

Suppose 64 workers update the same state.

One succeeds.

Many fail.

If failures retry immediately, the system may spend far more CPU on unsuccessful attempts than on useful commits.

This makes a lock-free implementation slower than a mutex that parks losers.

---

## 6. Backoff

Backoff intentionally delays retries.

The purpose is not "sleep for performance".

The purpose is to reduce the rate at which contenders attack the same shared location.

Possible strategies include:

- CPU pause/yield;
- increasing delay;
- adaptive retry.

The correct parameters depend on hardware and workload.

There is no universal backoff constant.

---

## 7. Linearization Point

A concurrent operation should appear to take effect at a single logical instant.

Example stack push:

```text
prepare node
↓
successful CAS(head)
↓
operation is now visible
```

Being able to identify the linearization point is central to reasoning about correctness.

If it is unclear, the algorithm is difficult to verify.

---

## 8. Helping

Some lock-free algorithms let one goroutine complete part of another goroutine's interrupted operation.

This can preserve system progress.

For example, queue algorithms may help advance a lagging tail pointer.

This is a major conceptual difference from simple critical-section reasoning.

---

## 9. Scheduler Interaction

A goroutine can be preempted in the middle of a lock-free operation.

Other goroutines may modify the structure many times before it resumes.

The algorithm must remain correct.

Lock-free theory does not imply good P99 latency in a real Go scheduler.

---

## 10. Garbage Collection Helps, But Not Completely

Go's GC removes much of the manual memory reclamation burden found in C/C++ lock-free structures.

If a goroutine holds a normal Go pointer to an object, that pointer participates in GC reachability.

However GC does not solve:

- ABA;
- linearizability;
- retries;
- starvation;
- incorrect invariants.

---

## 11. Pooling and Reuse

Object reuse can interact with lock-free correctness.

A removed node may later return to the structure with the same identity/address.

This can increase ABA concerns.

Reducing allocations with a pool is therefore not automatically safe for lock-free code.

---

## 12. Race-Free Is Not Correct

Atomic operations can eliminate data races while the algorithm remains logically wrong.

Possible bugs include:

- lost nodes;
- ABA;
- broken ordering;
- non-linearizable histories.

The race detector is necessary but not sufficient.

---

## 13. Benchmarking Lock-Free Structures

A useful benchmark should include:

- low contention;
- moderate contention;
- high contention;
- scaling across cores;
- retry counts if possible;
- CPU consumption;
- latency;
- allocations.

A lock-free implementation that improves throughput by 5% while using 5× CPU may not be a system improvement.

---

## 14. When Lock-Free Makes Sense

Candidates include specialized low-level structures where:

- operations are very small;
- blocking is problematic;
- contention pattern is understood;
- correctness can be rigorously tested;
- the expected benefit is significant.

Ordinary application state usually does not need a custom lock-free algorithm.

---

## 15. Engineering Principle

Do not reach for lock-free because it sounds advanced.

Reach for it only when the progress model solves a measured problem that simpler ownership or locking cannot solve satisfactorily.
