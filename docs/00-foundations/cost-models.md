# Cost Models

## 1. Why Cost Models Matter

A performance optimization is useful only if we know which cost it changes.

Without a cost model, developers easily confuse correlated observations with causes.

Examples:

```text
CPU profile shows atomic.Add
```

This does not prove:

```text
atomic implementation is inefficient
```

The real cost may be cache-line contention.

Similarly:

```text
heap allocations increased
```

does not automatically mean allocation CPU is the problem.

The real cost may be increased GC frequency.

A cost model turns a vague observation into a mechanism.

---

## 2. CPU Cost Model

A simplified CPU cost model includes:

```text
instructions
+
dependency chains
+
pipeline stalls
+
cache misses
+
branch misprediction
+
memory latency
```

Source-level operation count alone is insufficient.

Two implementations can execute similar Go code but differ greatly because one has better cache locality or fewer dependency chains.

### Instructions

More instructions generally mean more work, but instruction count is not the only factor.

Modern CPUs can execute multiple independent instructions per cycle.

### Cycles

Cycles reflect how long the processor spends executing and waiting.

A high cycle count with similar instruction count may indicate:

- cache miss;
- branch miss;
- data dependency;
- synchronization;
- memory stalls.

### Memory Latency

A load that hits L1 is dramatically cheaper than one that reaches DRAM.

Therefore the same logical access can have different physical cost depending on locality.

---

## 3. Cache Cost Model

The CPU moves data in cache-line units.

A useful mental model is:

```text
address
  ↓
cache line
  ↓
core-local cache
```

Performance depends on:

- whether the line is already present;
- whether nearby data is useful;
- whether another core is writing the same line;
- whether the line is repeatedly evicted.

This creates two important concepts:

### Locality

Good locality reduces expensive memory traffic.

### Coherence

Shared writes cause cores to coordinate ownership of cache lines.

This is the basis for atomic and false-sharing costs.

---

## 4. Memory Cost Model

Memory-related cost includes more than the number of allocated bytes.

Important dimensions:

```text
allocation count
allocation bytes
working-set size
retained memory
fragmentation
memory bandwidth
locality
```

### Allocation Count

Many tiny allocations can create allocator overhead and GC object metadata work.

### Allocation Bytes

High bytes/sec consumes heap headroom and can increase GC cycle frequency.

### Retention

An object may logically contain little useful data while retaining much more backing storage.

### Bandwidth

Some workloads are limited by how fast data can be moved, not how fast instructions can execute.

---

## 5. Garbage Collection Cost Model

A useful Go GC model separates:

```text
allocation rate
live heap
scannable heap
pointer density
roots
memory limit pressure
```

### Allocation Rate

High allocation rate consumes the space between GC targets quickly.

Even short-lived objects can cause frequent GC cycles.

### Live Heap

Long-lived memory must remain available across collections.

Larger live sets generally require more memory and can increase GC work.

### Scannable Heap

Pointer-free memory is cheaper to trace than pointer-rich object graphs.

A large byte buffer and an equally large pointer graph can have very different GC costs.

### Pointer Density

More pointers mean more potential edges to inspect.

### Object Graph Shape

Flat structures are often easier to scan and parallelize than deeply linked structures.

---

## 6. Concurrency Cost Model

Concurrency introduces costs not visible in sequential code.

```text
shared state
   ↓
synchronization
   ↓
coherence / waiting / retry
```

Important dimensions:

- lock hold time;
- arrival rate;
- number of contenders;
- atomic RMW frequency;
- retry rate;
- goroutine parking;
- scheduling;
- ownership topology.

### Contention

A lock is not expensive because it exists.

It becomes expensive when multiple workers need it at the same time.

A useful approximation is:

```text
contention pressure
≈ arrival rate × critical-section duration
```

This is not an exact formula, but it explains why shortening critical sections is often powerful.

---

## 7. Atomic Cost Model

An atomic load, store, and read-modify-write operation have different costs.

The important distinction is not merely:

```text
atomic vs mutex
```

but:

```text
read-only shared access
vs
shared write ownership
```

A high-frequency atomic increment on one shared counter can cause repeated cache-line ownership transfers across cores.

The code may contain no lock, yet still serialize through hardware coherence.

---

## 8. Compiler Cost Model

The compiler can remove work only when it can prove that removal is safe.

Examples:

```text
cannot prove bounds
→ retain bounds check

cannot prove lifetime
→ heap allocation

cannot determine dynamic receiver
→ indirect interface dispatch

cannot inline
→ less context for later optimization
```

Many compiler optimizations are therefore **proof problems**.

The source code communicates not only behavior but also information that the optimizer can or cannot derive.

---

## 9. Abstraction Cost Model

Abstraction has multiple possible costs:

- indirect calls;
- interface conversion;
- boxing;
- reduced inlining;
- weaker escape analysis;
- additional allocation.

But abstraction can also be free if the compiler successfully removes it.

Therefore:

> abstraction cost must be measured at generated-code level, not guessed from syntax.

---

## 10. Operating-System Cost Model

The runtime interacts with:

- virtual memory;
- pages;
- system calls;
- thread scheduling;
- file descriptors;
- page cache;
- mmap;
- network stack.

Some application-level memory does not belong to the Go heap.

Therefore:

```text
RSS
≠
Go heap
```

and:

```text
GOMEMLIMIT
≠
process hard RSS limit
```

in programs that use significant native memory, mmap, or other external allocations.

---

## 11. Latency Cost Model

Latency includes more than CPU execution.

A request can spend time:

```text
running
waiting for CPU
waiting for lock
waiting for channel
helping GC
waiting for syscall
waiting for network
```

This explains why CPU profiles alone cannot diagnose every latency problem.

---

## 12. Cost Interaction

Costs often interact.

Example:

```text
pointer-heavy representation
```

may simultaneously increase:

- allocation count;
- GC scan work;
- cache miss;
- TLB pressure;
- memory footprint.

Conversely, a compact representation may improve multiple layers at once.

The highest-value optimizations often change representation or ownership in a way that removes several costs together.

---

## 13. The Right Question

Instead of asking:

> Which technique is faster?

Ask:

> Which cost dominates this workload, and which representation or execution model removes that cost with acceptable trade-offs?

That question is the foundation of performance engineering.
