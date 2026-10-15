# Go Memory Allocator

## 1. Why Allocation Matters

Heap allocation is often treated as synonymous with "slow".

That model is too simple.

Modern Go is designed to make common small allocations cheap. The runtime uses per-P allocation state, size classes, spans, and caching so that most small allocations avoid both global locks and operating-system calls.

The important performance question is therefore not:

> Does this code allocate?

It is:

> How often does it allocate, how many bytes does it allocate, what lifetime do those objects have, and what downstream GC work do they create?

Allocation has several distinct costs:

```text
allocation CPU
+
zeroing / initialization
+
object metadata
+
heap growth
+
GC frequency
+
GC scan work
```

A single cheap allocation may be irrelevant.

Millions of allocations per second can become a system-level cost even if each one is individually inexpensive.

---

## 2. The Runtime Allocation Hierarchy

A simplified view of the Go allocator is:

```text
goroutine
   ↓
current P
   ↓
mcache
   ↓
mcentral
   ↓
mheap
   ↓
operating system
```

Each level exists to keep the common path local and amortize expensive global work.

### mcache

`mcache` is associated with a P.

It holds spans with free objects for size classes used by small allocations.

The common small-object fast path can usually allocate from the current P's cached span without taking a global lock.

### mcentral

`mcentral` manages spans for a size class.

When a P's cached span runs out of free slots, the allocator can obtain another span from the corresponding central structure.

### mheap

`mheap` manages the heap at page granularity.

It is responsible for obtaining and managing larger runs of pages.

### Operating System

When the runtime needs more virtual/physical memory than its current heap can provide, it obtains memory from the operating system.

This is a much less frequent path than individual application allocations.

---

## 3. Small Allocations and Size Classes

As of Go 1.27, the runtime treats allocations up to and including roughly 32 KiB as small allocations.

Small sizes are rounded to one of a set of size classes.

Conceptually:

```text
request N bytes
    ↓
choose size class
    ↓
allocate one slot from span
```

The size-class system trades a small amount of internal fragmentation for:

- fast allocation;
- fixed-size free-slot management;
- efficient reuse;
- reduced allocator metadata complexity.

The exact size-class table is a runtime implementation detail and can change.

Application code should not normally be written around specific size-class boundaries.

---

## 4. Internal Fragmentation

Suppose a program requests:

```text
65 bytes
```

but the nearest size class is larger.

The unused portion of the slot is internal fragmentation.

For one object, this rarely matters.

For millions of long-lived objects, a small per-object difference may accumulate into meaningful memory.

This is why struct-size changes sometimes have a larger real footprint effect than the raw `unsafe.Sizeof` difference suggests.

However:

> Do not reorder ordinary structs solely to chase allocator classes without heap-profile evidence.

---

## 5. mspan

An `mspan` represents a run of pages used to hold objects of one size class or a large allocation.

For small objects, one span is divided into multiple equal-sized slots.

Conceptually:

```text
span
┌────┬────┬────┬────┬────┐
│obj │obj │free│obj │free│
└────┴────┴────┴────┴────┘
```

Free-slot metadata lets the allocator find space quickly.

The span model is important because it connects object lifetime with fragmentation and later page reclamation.

---

## 6. Large Allocations

Objects above the small-allocation threshold follow a different path.

Large objects are allocated from the heap at page granularity rather than being packed into ordinary small-object size-class spans.

This changes the cost model.

Large frequent temporary allocations can:

- consume heap headroom rapidly;
- increase page-management work;
- increase RSS variability;
- interact strongly with pooling/reuse decisions.

Example:

```go
buf := make([]byte, 4<<20)
```

inside a hot request path deserves more scrutiny than one isolated small object.

---

## 7. Tiny Allocator

Go also has specialized handling for very small pointer-free objects.

The tiny allocator can pack multiple tiny noscan objects into a small shared block.

This reduces:

- allocation bookkeeping;
- object-slot overhead;
- metadata cost.

A key condition is that these objects must not contain pointers.

This is another example of how pointer-free representations can help both:

```text
allocator
+
garbage collector
```

The exact tiny-allocation thresholds are implementation details and should not become application-level constants.

---

## 8. Pointer-Free vs Pointer-Containing Allocation

The allocator and GC distinguish memory that contains pointers from memory that does not.

Consider:

```go
make([]byte, n)
```

The backing array is pointer-free.

Compare:

```go
make([]*Node, n)
```

The backing array contains Go pointers.

Both allocate bytes, but the second structure creates GC-visible references that must participate in tracing.

This means allocation size alone does not describe GC cost.

---

## 9. Zeroing

Go guarantees zero values for newly allocated memory.

The runtime/compiler must therefore ensure memory is appropriately initialized.

Some allocation paths can reuse already-cleared memory efficiently.

Others require explicit clearing.

Large objects can make zeroing itself measurable.

This is one reason reuse can help large scratch buffers:

```text
avoid allocation
+
avoid repeated zeroing/growth
```

But reuse can also retain too much memory, so it must be evaluated as a trade-off.

---

## 10. Go 1.27 Small-Allocation Specialization

Go 1.27 added compiler-generated calls to size-specialized allocation routines for some very small allocations.

The release notes describe reductions of up to roughly 30% for some allocations below about 80 bytes, with a much smaller overall improvement expected in real allocation-heavy programs.

This is important for performance engineering because it reinforces a general rule:

> The raw CPU cost of one small allocation is not stable across Go versions and should not be treated as a permanent constant.

Old microbenchmarks can become obsolete as the runtime/compiler improves.

---

## 11. Allocation Count vs Allocation Bytes

These are different dimensions.

Example A:

```text
10,000 allocations / second
1 KiB each
≈ 10 MiB/s
```

Example B:

```text
1,000,000 allocations / second
8 bytes each
≈ 8 MiB/s
```

The byte rates are similar.

But B creates far more object-allocation operations.

Potential costs differ:

```text
A:
higher per-object size

B:
higher object count
more allocator metadata activity
```

GC frequency is often strongly influenced by allocated bytes, while allocator CPU can also be sensitive to object count.

Both metrics matter.

---

## 12. Allocation Fast Path Does Not Make Allocation Free

A small allocation may be very cheap.

But it still creates a heap object.

That object may later participate in:

- reachability;
- marking;
- sweeping;
- span occupancy;
- fragmentation.

Therefore eliminating a hot allocation can create benefits larger than the allocator call itself.

This is why escape analysis can be valuable even when `mallocgc` is not an obvious CPU hotspot.

---

## 13. Preallocation

Preallocation reduces repeated growth.

Example:

```go
items := make([]Item, 0, expected)
```

can avoid:

```text
allocate
copy
grow
allocate
copy
grow
```

Benefits may include:

- fewer allocations;
- fewer copied bytes;
- lower GC churn.

But excessive preallocation can increase retained memory.

The correct capacity is workload-dependent.

---

## 14. Append-Style APIs

An API returning a fresh buffer:

```go
func Encode(v Value) []byte
```

may force allocation ownership into the callee.

An append-style API:

```go
func AppendEncode(dst []byte, v Value) []byte
```

lets the caller reuse storage.

This can reduce:

- allocation rate;
- temporary objects;
- copying.

The trade-off is a more explicit memory-ownership API.

---

## 15. Object Reuse

Reuse can reduce allocation churn.

Common mechanisms include:

- caller-owned buffers;
- reusable scratch state;
- `sync.Pool`;
- slab/index storage.

But reuse also changes lifetime and retention.

A reusable 32 MiB buffer can become worse than allocating 64 KiB buffers if it remains retained indefinitely.

Reuse must therefore include a retention policy.

---

## 16. Allocator Optimization Hierarchy

When allocation is proven hot, a healthy escalation is often:

```text
Can the value stay on stack?
        ↓
Can allocation be removed by API/lifetime design?
        ↓
Can capacity be preallocated?
        ↓
Can storage be caller-owned/reused?
        ↓
Would a pool/slab be justified?
```

Jumping immediately to pooling can hide a simpler lifetime problem.

---

## 17. What to Measure

Useful allocation evidence includes:

```text
allocs/op
B/op
allocation bytes/sec
object count
heap profile
allocs profile
GC CPU
```

Different measurements answer different questions.

### `allocs/op`

How many heap allocations does the operation perform?

### `B/op`

How many heap bytes are allocated per operation?

### allocs profile

Which call stacks create cumulative allocation traffic?

### heap profile

Which objects remain live?

---

## 18. Version Sensitivity

Allocator internals are implementation details.

The stable ideas are:

- allocation rate matters;
- object count matters;
- pointer-free objects differ from pointer-containing objects;
- reuse trades allocation for retention.

The exact thresholds, size classes, and fast paths must be revalidated against the target Go version.

---

## 19. Related Official Sources

- Go runtime allocator source: https://go.dev/src/runtime/malloc.go
- Go 1.27 release notes: https://go.dev/doc/go1.27
- Go GC guide: https://go.dev/doc/gc-guide

---

## 20. Engineering Perspective

The allocator should be understood as a highly optimized memory-management pipeline.

The right performance question is not:

> How do I avoid every allocation?

It is:

> Which heap allocations create meaningful total system cost, and what is the simplest way to remove or amortize them?
