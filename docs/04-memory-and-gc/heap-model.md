# Heap Model

## 1. The Heap Is Not One Number

When developers say:

> The program uses 4 GiB of memory.

they may be referring to very different things:

- currently live Go objects;
- total Go heap reserved;
- runtime metadata;
- goroutine stacks;
- pages retained for reuse;
- mmap;
- cgo/native allocations;
- process RSS.

Performance engineering requires separating these concepts.

---

## 2. Live Heap

The live heap is memory occupied by objects that remain reachable after garbage collection.

Conceptually:

```text
roots
 ↓
reachable objects
 ↓
live heap
```

The live heap strongly affects:

- minimum memory requirement;
- GC work;
- heap-growth target.

A program with a 20 GiB truly live working set cannot be fixed by simply running GC more often.

---

## 3. Heap Capacity vs Live Heap

The runtime usually owns more heap memory than the exact bytes currently occupied by live objects.

Reasons include:

- free slots inside spans;
- unused pages retained for future allocation;
- GC target headroom;
- fragmentation.

Therefore:

```text
live heap
<
runtime heap footprint
```

is normal.

---

## 4. Scannable Heap

A crucial distinction is:

```text
live heap
vs
scannable heap
```

Consider 1 GiB of:

```go
[]byte
```

versus 1 GiB of:

```go
[]*Node
```

The byte backing array contains no pointers.

The pointer slice contains references the GC must inspect.

The two heaps may have similar byte size but very different tracing cost.

---

## 5. Pointer Density

Pointer density describes how much of a representation consists of GC-visible references.

Pointer-bearing Go values include more than explicit `*T`.

Examples include:

- strings;
- slices;
- maps;
- channels;
- interfaces;
- function/closure state.

A struct with no explicit `*SomeType` can still contain pointer-bearing fields.

---

## 6. Object Graph Shape

Tracing cost also depends on graph structure.

A flat array:

```text
[objects objects objects]
```

has different locality from:

```text
node → node → node
       ↘ node
```

Pointer-heavy graphs can create:

- more scan edges;
- less memory locality;
- less efficient traversal.

This connects GC cost with CPU-cache behavior.

---

## 7. Span Occupancy

Heap objects occupy slots inside spans.

Suppose:

```text
span:
live dead live dead live
```

The dead slots can be reused.

But the pages backing the span cannot necessarily be released to the operating system while live objects remain.

This creates fragmentation.

---

## 8. Internal Fragmentation

Internal fragmentation is unused space inside allocated slots.

Example:

```text
requested object < size-class slot
```

The runtime accepts some waste in exchange for fast fixed-size allocation.

At scale, this can influence total footprint.

---

## 9. External / Page-Level Fragmentation

Even if total live bytes are small, they may be spread across many partially occupied spans/pages.

This can prevent large regions from becoming fully free.

Therefore:

```text
HeapAlloc falls
```

does not guarantee:

```text
RSS falls immediately
```

---

## 10. Runtime-Reusable Memory

After objects die, their memory may become free for future Go allocations.

This is valuable even if the operating system still considers the pages resident.

Reusing already-owned heap memory is often cheaper than returning it to the OS and reacquiring it later.

This is why aggressively forcing memory release can hurt steady-state workloads.

---

## 11. RSS

RSS represents resident process pages seen by the operating system.

It can include:

- Go heap;
- Go stacks;
- runtime structures;
- code/data;
- mmap;
- native/cgo memory.

Therefore:

```text
RSS ≠ heap live bytes
```

A high RSS does not automatically imply a Go heap leak.

---

## 12. Virtual Memory

The process may reserve virtual address space that is not fully backed by resident physical memory.

This is relevant to:

- large sparse mappings;
- mmap;
- historical heap-ballast techniques.

A large virtual reservation can coexist with a much smaller RSS until pages are touched.

---

## 13. Heap Metrics Need Context

Useful memory questions include:

### "Who is allocating?"

Use cumulative allocation information.

### "Who is retaining?"

Use live/inuse heap information.

### "Why is RSS still high?"

Investigate:

- retained heap pages;
- fragmentation;
- scavenging;
- stacks;
- mmap/cgo.

These are different questions.

---

## 14. Heap Objects and Stack Objects

Stack values generally do not become independent heap objects.

This can reduce:

- allocator traffic;
- GC object count;
- heap metadata.

Therefore escape analysis affects the heap model directly.

---

## 15. Pointer-Free Bulk Storage

A compact flat representation:

```go
type Node struct {
    Left  uint32
    Right uint32
    Value uint64
}

nodes []Node
```

can create a large pointer-free backing array.

Potential benefits:

- lower scan work;
- fewer heap objects;
- better cache locality;
- better page density.

This is one reason representation design often matters more than allocator micro-tuning.

---

## 16. Large Live Byte Buffers

A large pointer-free heap can still consume substantial physical memory.

GC may scan it cheaply, but the application still pays:

- RSS;
- memory bandwidth;
- cache pressure;
- paging risk.

Reducing GC scan does not make memory capacity irrelevant.

---

## 17. Heap Growth Headroom

Go intentionally allows heap growth between GC cycles.

This headroom lets the application allocate without continuously collecting.

The amount depends on GC pacing and memory-limit policy.

This creates the fundamental trade-off:

```text
more headroom
→ fewer GC cycles
→ more memory

less headroom
→ more GC cycles
→ less memory
```

---

## 18. Memory Ownership

A data structure may logically contain little data but own large backing storage.

Examples:

```text
slice len=100
cap=16 MiB
```

or:

```text
tiny substring/view
→ large original buffer remains reachable
```

Logical size is not ownership size.

This distinction is central to memory-retention analysis.

---

## 19. Version Sensitivity

The stable concepts are:

- live heap;
- reachability;
- pointer scanning;
- reusable heap memory;
- RSS separation.

Exact runtime metrics and internal span/GC organization can evolve.

---

## 20. Engineering Perspective

Do not optimize "memory" as one scalar.

First determine which memory is being discussed:

```text
live objects?
allocation churn?
runtime heap capacity?
retained backing storage?
fragmentation?
RSS?
external memory?
```

Only then choose the correct mechanism.
