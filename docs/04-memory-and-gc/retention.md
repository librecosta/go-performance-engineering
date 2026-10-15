# Memory Retention

## 1. Allocation and Retention Are Different Problems

Allocation answers:

> How much memory is created?

Retention answers:

> How long does memory remain reachable?

A program can have low allocation rate and still consume large memory because it retains too much.

Conversely, a program can allocate huge amounts over time while keeping a small steady live heap.

These require different diagnostics.

---

## 2. Backing Arrays

Slices are descriptors over backing arrays.

Conceptually:

```text
slice:
pointer
length
capacity
```

A small slice can keep a large backing array reachable.

Example:

```go
func first100(buf []byte) []byte {
    return buf[:100]
}
```

If `buf` is 100 MiB, the returned 100-byte slice may retain that entire backing allocation.

---

## 3. Intentional Copy to Release Storage

A deliberate copy can break the ownership link:

```go
func first100(buf []byte) []byte {
    return append([]byte(nil), buf[:100]...)
}
```

Now only the copied small allocation remains reachable.

This is an important counterexample to:

> Fewer allocations are always better.

---

## 4. Slice Capacity Retention

A slice with:

```text
len = 0
cap = 32 MiB
```

still owns a large backing array.

Resetting:

```go
buf = buf[:0]
```

does not release the capacity.

This is useful for reuse but can create long-lived memory.

---

## 5. Reusable Buffers

Worker-local buffers often use:

```go
buf = buf[:0]
```

to avoid allocation.

This works well when capacity remains near normal workload size.

But an outlier request can grow the buffer permanently.

A retention policy may be needed:

```text
if capacity unusually large
→ drop buffer
```

---

## 6. `sync.Pool` Retention

Pools can retain large objects temporarily.

Although pool contents are not guaranteed to persist indefinitely, placing unusually large buffers into a pool can raise memory footprint.

A common policy is:

```go
if cap(buf) <= maxPooled {
    pool.Put(buf[:0])
}
```

The exact threshold is workload-specific.

---

## 7. Strings and Substrings

Strings also reference backing data.

When parsing a very large string/buffer, small retained substrings may keep larger storage alive depending on representation and conversion path.

Parser design should consider whether tokens should:

- alias source storage;
- copy into compact storage.

The correct choice depends on token lifetime.

---

## 8. Caches

Caches intentionally retain memory.

A cache leak and a cache policy may look similar in heap profiles.

Important questions:

- Is the cache bounded?
- Does it have TTL/LRU/size policy?
- Are values larger than expected?
- Does key cardinality grow indefinitely?
- Is memory pressure fed back into eviction?

A cache without an explicit ownership policy can become accidental permanent retention.

---

## 9. Maps

Deleting entries from a map removes logical references, but internal capacity behavior does not necessarily shrink immediately in proportion to entry count.

Long-lived maps that grow to peak size and later become sparse can retain significant memory.

Sometimes rebuilding a map is justified after large phase changes.

This should be measured rather than done periodically by habit.

---

## 10. Object Graph Retention

A single root can retain a large graph.

Example:

```text
cache entry
   ↓
session
   ↓
request history
   ↓
large buffers
```

Heap profiling should follow retention paths conceptually, not just blame the largest leaf allocation.

---

## 11. Closures and Goroutines

Long-lived goroutines or closures can retain references unintentionally.

Example:

```go
large := loadLargeData()

go func() {
    useSomething(large)
    ...
}()
```

If the goroutine lives indefinitely, `large` may remain reachable.

Goroutine leaks can therefore become memory-retention problems.

---

## 12. Channels

Buffered channels retain queued values.

A large-capacity channel can therefore hold substantial memory.

If consumers stall, the queue itself becomes a retention mechanism.

Backpressure design and queue size are memory-design decisions.

---

## 13. Finalizers / Cleanup and Lifetime

Objects awaiting cleanup/finalization can have longer effective lifetime than expected.

Resource cleanup should normally be explicit where deterministic release matters.

GC-driven cleanup is not an immediate memory-release mechanism.

---

## 14. Retention vs Fragmentation

Even after references are removed, RSS may remain high because freed objects occupy spans/pages that are not fully releasable.

This is fragmentation rather than reachability retention.

Heap inuse metrics and RSS therefore need to be compared carefully.

---

## 15. Heap Profile vs Allocs Profile

### Heap / inuse

Useful for:

> What is still live?

### Allocs / alloc_space

Useful for:

> What created allocation traffic over time?

For retention problems, heap/inuse is usually the more direct starting point.

---

## 16. Copy vs Zero-Copy

Zero-copy can reduce:

- allocation;
- memcpy;
- bandwidth.

But it can increase retention by keeping a large source buffer alive.

The real comparison is:

```text
copy cost
vs
retained-memory cost
vs
ownership complexity
```

---

## 17. Phase-Oriented Programs

Some programs have clear phases:

```text
load data
↓
build index
↓
discard source
↓
steady state
```

If temporary data is no longer referenced but RSS remains high, scavenging/page release may be the next issue.

This is different from a true retention leak.

---

## 18. What to Measure

Useful retention evidence includes:

```text
heap inuse bytes
heap object count
RSS
large object paths
slice capacities
cache sizes
goroutine count
mmap/native memory
```

Compare before/after phase transitions when relevant.

---

## 19. Engineering Perspective

Memory retention is fundamentally an ownership problem.

Ask:

> Which live reference is responsible for keeping this storage reachable?

That question is often more useful than:

> Which line allocated it?
