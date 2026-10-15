# Garbage Collector

## 1. Why Go Uses Garbage Collection

Go allows programs to allocate heap objects without manually freeing each object.

The garbage collector determines when heap objects are no longer reachable and makes their memory reusable.

This greatly simplifies ownership compared with manual memory management.

The trade-off is runtime work.

Understanding GC performance therefore means understanding:

```text
what must be traced
how often tracing occurs
how much work application allocation creates
```

---

## 2. Tracing Garbage Collection

Go uses tracing GC.

Conceptually:

```text
roots
 ↓
follow pointers
 ↓
mark reachable objects
 ↓
unmarked objects are dead
```

The collector does not primarily ask:

> Which object was explicitly freed?

It asks:

> Which objects are still reachable?

---

## 3. Roots

GC traversal begins from roots such as:

- global variables;
- goroutine stacks;
- runtime-maintained roots.

From these roots, the collector follows Go pointers through the object graph.

This makes pointer density and graph shape important performance variables.

---

## 4. Marking

Marking discovers reachable heap objects.

A simplified graph:

```text
root
 ↓
A → B → C
    ↓
    D
```

If A is reachable, the collector follows its pointers to find B and D, then continues.

This work is proportional not merely to allocated bytes, but to reachable pointer-bearing structure.

---

## 5. Pointer-Free Objects

A byte array contains no Go pointers.

Once the collector knows the object is reachable, it does not need to treat every byte as a possible reference.

This is why:

```go
[]byte
```

and:

```go
[]*Node
```

have different scan costs even at similar byte size.

---

## 6. Object Graph Shape

A flat dense representation can be easier on the collector than millions of scattered linked nodes.

Reasons include:

- fewer objects;
- fewer pointers;
- better memory locality;
- less metadata traversal.

This is one reason pointer→index transformations can improve both CPU-cache and GC behavior.

---

## 7. Concurrent Collection

Modern Go GC performs much of its work concurrently with application goroutines.

This keeps stop-the-world pauses relatively small.

But concurrent GC still consumes CPU.

Therefore:

```text
small STW
```

does not imply:

```text
zero GC latency cost
```

The application and collector compete for CPU while marking.

---

## 8. Write Barriers

The application can mutate pointers while concurrent marking is running.

Example:

```go
a.Next = b
```

The collector must maintain correctness even if the object graph changes during tracing.

Write barriers cooperate with the GC to preserve required invariants.

Thus pointer mutation can become more expensive during mark phases.

---

## 9. Allocation During GC

The application continues allocating while collection is running.

This complicates pacing.

The collector must finish enough work before heap growth exceeds the current target.

When application allocation is too fast, goroutines may be required to assist the collector.

---

## 10. GC Assist

GC assist transfers some marking work to allocating goroutines.

Conceptually:

```text
goroutine allocates quickly
      ↓
accumulates GC work debt
      ↓
must perform marking work
```

This helps keep GC progress aligned with heap growth.

But it can increase request latency.

---

## 11. Sweeping

After marking determines which objects are unreachable, memory from dead objects becomes available for reuse.

Sweeping turns dead object slots back into allocator-free space.

Go performs sweeping largely concurrently.

In many workloads, marking/scanning is more important than sweep CPU.

---

## 12. Reclamation vs OS Release

When an object dies:

```text
GC reclaims object slot
```

This means Go can reuse the memory.

It does not automatically mean:

```text
OS RSS decreases immediately
```

Returning pages to the operating system is a separate scavenging process.

---

## 13. Green Tea GC

Green Tea became the default garbage collector in Go 1.26.

It redesigned important parts of marking/scanning to improve locality and scalability, especially for small-object-heavy heaps.

The stable engineering conclusion is not:

> Green Tea makes pointer-heavy structures free.

It is:

> Collector implementation has improved, so old GC microbenchmarks must be revalidated on modern Go.

Pointer count, live heap, allocation rate, and graph structure remain important.

---

## 14. Small-Object Scanning

Historically, tracing many small objects can create poor locality because the collector repeatedly follows pointers among scattered heap objects.

Green Tea improves how scanning work is organized to process memory with better locality.

This reduces GC overhead for many workloads.

However results remain workload-dependent.

---

## 15. GC CPU vs Pause Time

Two separate metrics matter:

### Pause

How long application execution is globally stopped.

### CPU

How much processor time the collector uses overall.

A service can have excellent pause times but still spend significant CPU in concurrent GC.

---

## 16. GC and Tail Latency

Potential sources of GC-related latency include:

- STW phases;
- mutator assists;
- CPU competition with mark workers;
- write-barrier work;
- root scanning interactions.

Therefore tail-latency analysis should not focus only on pause duration.

---

## 17. Allocation Rate

GC frequency depends strongly on how quickly the heap grows between cycles.

A program can have:

```text
small stable live heap
```

but:

```text
huge allocation churn
```

and spend substantial CPU on frequent GC.

---

## 18. Live Heap

If a large portion of heap remains reachable after collection, the collector cannot reclaim it.

Running GC more often cannot fix a genuinely large live set.

The solutions are architectural:

- retain less;
- compact representation;
- increase memory budget.

---

## 19. Pointer Density

A 10 GiB pointer-free cache and a 10 GiB pointer graph create different GC work.

This is why heap byte count alone is not enough.

Useful analysis includes:

```text
live bytes
+
scannable bytes
+
object count
+
pointer topology
```

---

## 20. Historical Heap Ballast

A heap ballast was a large long-lived pointer-free allocation used to artificially increase live heap.

Under older GOGC-based pacing:

```text
larger apparent live heap
→ larger heap target
→ less frequent GC
```

Because the ballast was often an untouched byte slice, virtual-memory behavior could make the physical cost smaller than the apparent heap size.

The technique was clever but unsupported.

Modern Go provides memory-limit controls that solve the tuning problem more directly.

Heap ballast should therefore be treated as a historical mechanism, not a current default recommendation.

---

## 21. Finalizers and Cleanup

GC-driven finalization is nondeterministic.

External resources such as:

- files;
- sockets;
- transactions;

should normally use explicit lifetime management.

Garbage collection is for memory reachability, not deterministic resource release.

Modern cleanup mechanisms can provide fallback cleanup for certain resource wrappers, but they do not turn Go GC into deterministic RAII.

---

## 22. What GC Cannot Fix

GC cannot solve:

- excessive live data;
- bad cache locality;
- memory bandwidth saturation;
- mutex contention;
- external mmap/cgo memory;
- incorrect ownership design.

GC tuning is not a substitute for application architecture.

---

## 23. Related Official Sources

- GC guide: https://go.dev/doc/gc-guide
- Green Tea GC: https://go.dev/blog/greenteagc
- Go 1.26 release notes: https://go.dev/doc/go1.26
- Runtime GC source: https://go.dev/src/runtime/mgc.go

---

## 24. Engineering Perspective

The garbage collector is best viewed as a service the application continuously pays for.

Application design determines the bill through:

```text
allocation rate
live heap
pointer density
object graph
```

The strongest GC optimizations often reduce those inputs rather than micro-tuning the collector itself.
