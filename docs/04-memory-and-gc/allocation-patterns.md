# Allocation Patterns

## 1. Allocation Behavior Matters More Than Allocation Existence

A program that allocates one long-lived 100 MiB structure and a program that allocates 100 MiB of temporary objects every second have very different performance characteristics.

The important dimensions are:

```text
size
frequency
lifetime
pointer content
reuse
```

These define the allocation pattern.

---

## 2. Allocation Churn

Allocation churn means repeatedly creating short-lived heap objects.

Example:

```go
func handle(items []Item) {
    for _, item := range items {
        tmp := make([]byte, 4096)
        process(tmp, item)
    }
}
```

The temporary buffers may all die quickly.

Live heap may remain small.

But allocation bytes/sec can be high.

This consumes heap headroom rapidly and increases GC frequency.

---

## 3. Short-Lived Does Not Mean Free

A temporary object still creates costs:

```text
allocation
+
heap growth
+
later GC accounting
```

Its short lifetime helps the collector reclaim it, but does not erase the cost of creating it.

This is why high-throughput serialization/parsing code often benefits from allocation reduction.

---

## 4. Allocation Rate

A useful mental model:

```text
available heap headroom
        ÷
allocation bytes/sec
        ≈
time until next GC pressure
```

This is only approximate, but explains an important phenomenon:

> GC frequency can increase dramatically while live heap stays nearly unchanged.

The cause may simply be more churn.

---

## 5. Burst Allocation

Some workloads allocate in bursts:

```text
steady state
↓
large batch arrives
↓
huge temporary allocation spike
↓
returns to normal
```

Burst behavior can create:

- temporary GC pressure;
- assist work;
- transient RSS increase;
- tail-latency spikes.

Average allocation rates may hide these events.

---

## 6. Steady Allocation

A stable stream of small allocations can create predictable GC behavior.

This is often easier for the runtime to pace.

Optimization decisions should consider whether the workload is:

- steady;
- bursty;
- phase-oriented.

---

## 7. Preallocation

Slices grow when capacity is exceeded.

Repeated append growth can cause:

```text
new backing allocation
+
copy old contents
```

If final size is approximately known:

```go
out := make([]Item, 0, expected)
```

can reduce both allocation and copying.

---

## 8. Over-Preallocation

Preallocation can be excessive.

Example:

```go
make([]Item, 0, 1_000_000)
```

for a workload that usually produces 100 items.

This trades allocation churn for retained memory.

Therefore preallocation should reflect realistic distributions rather than maximum theoretical size.

---

## 9. Map Capacity

Maps may also benefit from capacity hints when approximate size is known.

The objective is to reduce internal growth and reorganization.

As with slices, over-sizing can waste memory.

---

## 10. Append-Style APIs

Consider:

```go
func Marshal(v Value) []byte
```

The function controls the output allocation.

An append-style form:

```go
func AppendMarshal(dst []byte, v Value) []byte
```

allows:

- caller reuse;
- batching;
- stack/local buffer strategies.

This often appears in performance-oriented encoding APIs.

---

## 11. Caller-Owned Scratch Space

A caller can own temporary storage:

```go
buf := make([]byte, 0, 4096)

for ... {
    buf = buf[:0]
    buf = encode(buf, value)
}
```

This avoids repeated allocation.

But maximum observed capacity can become retained.

Long-lived scratch buffers therefore need size discipline.

---

## 12. Capacity Poisoning

Suppose normal requests need:

```text
64 KiB
```

but one request grows a reusable buffer to:

```text
32 MiB
```

If the buffer is retained forever:

```text
one outlier
→ permanent higher memory footprint
```

This is capacity poisoning.

It is especially relevant to pools and long-lived worker scratch buffers.

---

## 13. Copy vs Retain

Sometimes a deliberate copy reduces memory.

Example:

```go
small := append([]byte(nil), huge[:100]...)
```

This creates a new tiny allocation.

But it may allow a huge backing array to become unreachable.

Thus:

```text
allocation count ↑
live retained memory ↓↓↓
```

The total system result can be better.

---

## 14. `[]T` vs `[]*T`

A slice of pointers may imply:

- one allocation for the slice;
- many allocations for pointed-to objects.

A slice of values may store all objects in one backing array.

This can reduce:

- object count;
- pointer scanning;
- allocator traffic.

But value semantics may not fit all APIs.

---

## 15. Flat Index Storage

For large graphs or trees, index-based storage can turn:

```text
many heap nodes
```

into:

```text
one/few large backing arrays
```

This changes allocation shape dramatically.

Potential gains:

- fewer allocations;
- fewer objects;
- less GC metadata;
- better locality.

---

## 16. Slab-Like Storage

An application can manage logical objects inside a large slice or slab.

Example:

```go
type Arena struct {
    Nodes []Node
}
```

New logical object:

```go
idx := len(a.Nodes)
a.Nodes = append(a.Nodes, Node{})
```

A whole batch can be reset together.

This is useful when many objects share a lifetime.

---

## 17. Same-Lifetime Objects

Arena/slab-like patterns work best when objects:

```text
are born together
used together
die together
```

Examples:

- parser AST;
- query plan;
- request-scoped graph;
- batch state.

Mixed lifetimes reduce the usefulness of bulk reset.

---

## 18. `sync.Pool`

`sync.Pool` is one way to reuse temporary heap objects across operations.

It is covered separately because its semantics are intentionally weak and GC-aware.

See:

- [sync.Pool](./sync-pool.md)

---

## 19. Large Temporary Buffers

Large buffers deserve special attention because they can:

- use page-granular allocation;
- increase RSS rapidly;
- poison pools;
- create memory-limit pressure.

Often the best strategy is:

```text
reuse common sizes
drop unusual outliers
```

rather than pool every observed capacity.

---

## 20. Batch Allocation

Sometimes many objects can be represented in one allocation.

Example:

```go
items := make([]Item, n)
```

instead of:

```go
items := make([]*Item, n)
for i := range items {
    items[i] = new(Item)
}
```

This can drastically reduce allocation count.

---

## 21. Escape-Driven Patterns

Some allocation patterns are compiler artifacts.

A local temporary may heap-allocate because:

- it escapes through interface;
- it is captured by closure;
- a callee cannot be analyzed/inlined.

Before adding pooling, check whether the allocation can disappear through compiler-friendly lifetime design.

---

## 22. Allocation Patterns and Tail Latency

Burst allocation can force goroutines to perform GC assist work.

This can create request-latency spikes even when STW pauses are small.

Thus allocation optimization can be a latency optimization, not only a CPU optimization.

---

## 23. What to Compare

When evaluating an allocation change, compare:

```text
ns/op
B/op
allocs/op
GC CPU
live heap
RSS
P99 latency
```

No single metric is sufficient.

---

## 24. Engineering Perspective

A good allocation strategy matches object lifetime.

The best representation often makes the lifetime obvious:

```text
stack-local
caller-owned
batch-owned
pool-reusable
long-lived heap
```

Performance improves when ownership and lifetime align.
