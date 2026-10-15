# TLB and Pages

## 1. Virtual Memory

Go programs use virtual addresses.

The operating system maps virtual address ranges to physical memory through page tables.

Conceptually:

```text
virtual address
    ↓
page table translation
    ↓
physical memory
```

Performing a full page-table walk for every memory access would be expensive.

Processors therefore cache translations.

---

## 2. Translation Lookaside Buffer

The TLB caches mappings from virtual pages to physical pages.

A TLB hit makes address translation cheap.

A TLB miss may require a page-table walk.

This means memory locality affects both:

- cache behavior;
- address translation.

---

## 3. Page Locality

Suppose a program scans a contiguous slice.

Many adjacent elements share the same memory pages.

Once a page translation is cached, many accesses can reuse it.

Random pointer-heavy structures may touch many unrelated pages and create more translation pressure.

---

## 4. Object Density Per Page

A compact representation places more useful objects on each page.

For example, reducing an object from 32 bytes to 16 bytes can roughly double how many objects fit into the same amount of page space.

This may improve:

- cache density;
- TLB coverage;
- memory bandwidth efficiency.

Therefore object size is more than an RSS concern.

---

## 5. Pointer Chasing and TLB

Pointer chasing can combine several costs:

```text
dependent load
+
cache miss
+
TLB miss
```

Large graphs with scattered allocations are especially vulnerable.

This is one reason flat index-based representations can outperform pointer graphs even when both contain the same logical information.

---

## 6. Page Faults

Virtual address space does not imply all pages are physically resident.

Pages may be populated on demand.

This distinction is relevant to:

- large allocations;
- mmap;
- sparse access;
- heap ballast history.

A large virtual reservation may have a smaller RSS until pages are touched.

---

## 7. Huge Pages

Larger pages can increase the amount of memory covered by a small number of TLB entries.

This can help very large memory workloads.

But huge pages introduce system-level trade-offs:

- fragmentation;
- allocation behavior;
- latency;
- deployment complexity;
- kernel configuration.

They should not be a default Go optimization.

---

## 8. Transparent Huge Pages

Operating systems may automatically promote memory to large pages.

This behavior can vary between environments.

Benchmarks that depend on page behavior should record relevant system configuration.

---

## 9. mmap and Page Behavior

mmap maps file-backed or anonymous pages into the process address space.

The program may access data without copying it into Go-managed heap buffers.

This can reduce explicit copies, but performance depends on:

- page faults;
- access pattern;
- page cache;
- mapping lifetime;
- working set.

mmap is therefore an OS memory design choice, not merely a Go API optimization.

---

## 10. TLB Optimization Is Usually Data-Layout Optimization

Application code rarely manipulates the TLB directly.

The usual levers are:

- compact data;
- contiguous storage;
- fewer scattered allocations;
- predictable traversal.

Only specialized workloads should progress toward explicit huge-page or NUMA tuning.

---

## 11. Engineering Principle

When a large memory workload behaves unexpectedly, ask not only:

> How many bytes are used?

also ask:

- How many objects?
- How many pages?
- Are accesses sequential or random?
- Are objects compact?
- Does the program repeatedly cross unrelated pages?

Memory size and memory topology are different performance dimensions.
