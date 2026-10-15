# sync.Pool

## 1. Purpose

`sync.Pool` is designed for temporary reusable objects.

It can reduce pressure from high-frequency allocations by allowing objects to be reused across operations.

Typical candidates include:

- temporary buffers;
- encoders/decoders;
- scratch objects;
- formatting state.

It is not a general-purpose persistent cache.

---

## 2. Why Pooling Can Help

Without reuse:

```text
request
→ allocate scratch
→ use
→ scratch dies
→ repeat
```

With pooling:

```text
Get
→ use
→ reset
→ Put
```

Potential benefits:

- fewer allocations;
- lower allocation bytes/sec;
- fewer GC cycles;
- lower allocator CPU.

---

## 3. Pool Semantics Are Intentionally Weak

A `sync.Pool` entry can disappear at any time according to pool semantics/runtime behavior.

Code must therefore remain correct if:

```text
Get
→ pool empty
→ allocate new object
```

A pool must never be the only owner of essential state.

---

## 4. Pool Is Not a Cache

A cache promises some notion of retention.

A pool does not.

Do not use `sync.Pool` for:

- session storage;
- required object lookup;
- durable reuse guarantees.

Use it only when losing pooled objects is semantically harmless.

---

## 5. Reset Before Reuse

Pooled objects often contain old state.

Example:

```go
buf := pool.Get().(*bytes.Buffer)
buf.Reset()
```

Reset semantics must be correct.

For custom objects, make sure reused state does not accidentally retain:

- previous request data;
- large slices;
- pointers to large graphs.

---

## 6. Capacity Retention

A reused slice can have:

```text
len = 0
cap = huge
```

Resetting length does not shrink capacity.

Therefore a pool can retain oversized backing arrays.

This is one of the most important practical pool hazards.

---

## 7. Drop Oversized Objects

A common pattern is:

```go
if cap(buf) <= maxPooled {
    pool.Put(buf[:0])
}
```

This creates an explicit retention policy:

```text
common-size buffers
→ reuse

outlier buffers
→ let GC reclaim
```

The threshold should be derived from workload distribution, not copied blindly.

---

## 8. Pooling Small Cheap Objects

Pooling is not automatically useful for tiny objects.

Modern Go's small-allocation fast path is efficient.

Pool operations also have overhead and can complicate ownership.

A tiny object allocated rarely may be cheaper and clearer than pooling.

---

## 9. Pooling Large Scratch Buffers

Large temporary buffers are stronger candidates when:

- allocation is frequent;
- capacity is stable;
- reset is cheap;
- retaining normal-size buffers fits memory budget.

The gain can come from reducing both allocation and zeroing/growth.

---

## 10. Pool and GC

`sync.Pool` is deliberately integrated with GC behavior.

The pool does not provide permanent retention.

This is why it is suitable for opportunistic reuse.

Application code should depend only on documented semantics, not private per-P implementation details.

---

## 11. Pool and Contention

A global pool is internally optimized, but heavy use can still interact with concurrency and cache behavior.

Do not assume pooling always improves scalability.

Benchmark under realistic worker counts.

---

## 12. Pool and False Sharing

Custom pools or freelists may create shared hot metadata.

If a hand-written pool replaces `sync.Pool`, it can accidentally introduce:

- global atomics;
- lock contention;
- false sharing.

The custom allocator can become worse than the allocation it was designed to avoid.

---

## 13. Pool and Lock-Free Reuse

In lock-free data structures, node reuse can interact with ABA.

A pooled node can reappear with the same identity/address.

Therefore:

```text
allocation optimization
```

can change:

```text
concurrency correctness assumptions
```

This must be analyzed explicitly.

---

## 14. Pooling Pointer-Heavy Objects

Reusing pointer-rich objects may reduce allocation churn, but retained pointers must be cleared when appropriate.

Otherwise pooled objects can keep large graphs reachable longer than intended.

Example reset may need:

```go
obj.Big = nil
obj.Items = obj.Items[:0]
```

depending on desired capacity retention.

---

## 15. Pooling and Sensitive Data

Reused buffers may contain previous contents.

If objects cross security or tenant boundaries, reset semantics may need explicit clearing rather than only changing length.

Performance reuse must not violate confidentiality requirements.

---

## 16. Benchmarking a Pool

Compare:

```text
without pool
vs
with pool
```

under realistic:

- object sizes;
- concurrency;
- burst behavior;
- outlier capacities.

Measure:

```text
ns/op
B/op
allocs/op
GC CPU
RSS
```

A reduction in `allocs/op` alone does not prove the pool is beneficial.

---

## 17. Pool Lifecycle

The best pooled objects usually have:

```text
short logical lifetime
repeated shape
cheap reset
```

Objects with irregular ownership or long lifetime are poor pool candidates.

---

## 18. Common Misconceptions

### "sync.Pool is an object cache"

False.

### "Pooling always reduces memory"

False; it may retain memory.

### "0 alloc/op is always better"

False.

### "If one benchmark improves, pool everything"

False.

---

## 19. Related Official Sources

- `sync.Pool`: https://pkg.go.dev/sync#Pool
- Go GC guide: https://go.dev/doc/gc-guide
- Runtime allocator: https://go.dev/src/runtime/malloc.go

---

## 20. Engineering Perspective

Pooling is a churn-control mechanism.

The right question is:

> Is repeated temporary allocation measurably expensive, and can this object be safely reused without retaining too much state or memory?
