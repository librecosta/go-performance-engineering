# Memory Limits, RSS, and Scavenging

## 1. Why a Memory Limit Exists

GOGC controls proportional heap growth.

But production systems often operate under an external memory budget:

- container limit;
- VM size;
- process budget.

Go therefore provides a soft runtime memory limit through:

- `GOMEMLIMIT`;
- `debug.SetMemoryLimit`.

The goal is to let the runtime trade GC CPU for memory while respecting an approximate upper budget.

---

## 2. GOMEMLIMIT Is Not GOGC

### GOGC

Controls normal heap-growth ratio.

### GOMEMLIMIT

Controls memory pressure against an absolute soft budget.

They complement each other.

---

## 3. Soft Runtime Memory Budget

The memory limit applies to memory managed/understood by the Go runtime according to its documented accounting model.

It should not be interpreted as:

```text
hard process RSS cap
```

because a process can also consume memory through:

- mmap;
- cgo/native allocators;
- external libraries;
- OS/kernel buffers.

---

## 4. Container Headroom

Suppose a container has:

```text
8 GiB hard limit
```

Setting:

```text
GOMEMLIMIT = 8 GiB
```

leaves little room for memory not included in Go runtime accounting.

A safer operational policy usually leaves headroom.

The exact margin depends on:

- cgo;
- mmap;
- workload;
- stack growth;
- native libraries;
- observability agents.

No universal percentage fits every process.

---

## 5. Memory Limit Becomes a GC Pressure Signal

When normal GOGC pacing would exceed the soft limit, the runtime increases GC pressure.

Conceptually:

```text
heap grows
↓
approaches memory limit
↓
collector runs more aggressively
```

This trades CPU for lower runtime memory.

---

## 6. Too-Low Limit

If the live set itself is close to the limit:

```text
GC
↓
almost nothing dies
↓
allocation
↓
limit pressure
↓
GC again
```

The program can enter GC thrashing.

This manifests as:

- high GC CPU;
- frequent cycles;
- assist work;
- low application progress.

The fix is not "more GC".

The fix is:

- reduce live set;
- increase memory budget;
- redesign representation.

---

## 7. Memory Limit and GOGC

Far below the limit, GOGC may determine normal pacing.

Near the limit, the memory budget becomes dominant.

This gives the runtime two dimensions:

```text
preferred growth policy
+
maximum soft budget
```

---

## 8. Disabling Ordinary GOGC

Using:

```text
GOGC=off
```

or equivalent runtime setting while keeping a memory limit can make the memory limit the main GC trigger.

This can be useful when:

- memory budget is fixed;
- using available memory to reduce GC CPU is desirable.

But this behavior should be evaluated under real traffic and tail-latency conditions.

---

## 9. RSS

RSS is an operating-system view of resident process memory.

It includes more than Go heap live objects.

Possible contributors:

```text
Go heap
Go stacks
runtime metadata
binary/code
mmap
cgo/native memory
```

Therefore:

```text
RSS high
```

does not directly imply:

```text
Go heap leak
```

---

## 10. Heap Reclaim vs RSS

GC can reclaim dead objects and make their heap slots reusable.

But pages may stay mapped/resident because:

- runtime expects reuse;
- spans are partially occupied;
- scavenger has not released pages yet.

This creates a gap between:

```text
heap live
```

and:

```text
RSS
```

---

## 11. Scavenger

The runtime scavenger returns unused physical pages to the operating system.

Conceptually:

```text
objects die
↓
sweep frees slots
↓
pages become fully/partially reclaimable
↓
scavenger releases physical backing
↓
RSS may fall
```

Scavenging is intentionally separate from object tracing.

---

## 12. Why Runtime Retains Memory

Keeping free heap memory can improve performance because the runtime can reuse it without reacquiring pages from the OS.

A steady-state server often benefits from this reuse.

Therefore immediately returning every free page is not necessarily optimal.

---

## 13. `debug.FreeOSMemory`

`debug.FreeOSMemory` forces a GC and attempts to return as much free memory as possible to the OS.

This can make sense after a major phase transition:

```text
startup / indexing phase
↓
large temporary heap
↓
temporary objects die
↓
steady state
```

It is usually not appropriate as a periodic steady-state "memory cleanup" timer.

That fights normal GC/scavenger policy.

---

## 14. Fragmentation

Suppose pages contain:

```text
live
free
live
free
```

Memory is reusable by Go but may not be releasable as whole pages.

This is fragmentation.

If:

```text
HeapAlloc ↓
RSS stays high
```

fragmentation is one possible explanation.

---

## 15. Large Object Release

Large allocations can sometimes produce page regions that are easier to return once completely unused.

However workload shape and reuse matter.

Large-object churn can still create RSS volatility.

---

## 16. mmap

mmap memory belongs to the process address space but not ordinary Go heap allocation.

It can significantly affect:

- RSS;
- virtual memory;
- page faults.

GOMEMLIMIT is not a complete accounting system for such memory.

Applications using large mappings should monitor them separately.

---

## 17. cgo and Native Allocators

Native libraries may allocate memory unknown to Go's heap accounting.

Examples:

- C libraries;
- databases;
- image codecs;
- GPU/native runtimes.

A Go process with significant native memory needs a process-level memory budget in addition to GOMEMLIMIT.

---

## 18. Observability

Useful memory dashboards should distinguish:

```text
process RSS
Go heap live
Go heap goal
released heap
allocation rate
GC CPU
external/native memory if available
```

One graph cannot explain all memory behavior.

---

## 19. OOM Prevention

GOMEMLIMIT can reduce OOM risk by encouraging earlier/more aggressive collection.

But it cannot prevent OOM when:

- live Go heap exceeds budget;
- native memory grows independently;
- mmap dominates;
- kernel/container accounting differs.

It is a control mechanism, not an absolute safety guarantee.

---

## 20. Memory Budget Design

A practical memory budget separates:

```text
Go runtime budget
+
external/native budget
+
operational safety margin
```

The exact split should be derived from observed process behavior.

---

## 21. Historical Ballast Connection

Heap ballast historically solved:

> The machine has lots of memory, but GOGC triggers too frequently because live heap is small.

It did this by artificially inflating live heap.

GOMEMLIMIT provides a supported way to express the more useful operational intention:

> Use available memory to reduce GC pressure, but stay near this budget.

This is why ballast is now best understood as historical context.

---

## 22. Related Official Sources

- GC guide: https://go.dev/doc/gc-guide
- `runtime/debug`: https://pkg.go.dev/runtime/debug
- Runtime scavenger: https://go.dev/src/runtime/mgcscavenge.go

---

## 23. Engineering Perspective

Memory limits should be treated as part of capacity planning.

The correct question is not:

> What number should I set GOMEMLIMIT to?

It is:

> How much memory belongs to the Go runtime, how much belongs elsewhere, how large is the true live set, and how much GC CPU is acceptable under pressure?
