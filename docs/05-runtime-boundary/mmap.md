# mmap

## 1. What Memory Mapping Changes

A traditional file-read path often looks like:

```text
file
 ↓
kernel page cache
 ↓
read syscall
 ↓
Go buffer
```

Memory mapping creates a virtual-address range backed by file pages.

Conceptually:

```text
file-backed pages
       ↓
process virtual address space
       ↓
direct memory access
```

This can eliminate explicit userspace read buffers and copies in some workloads.

---

## 2. mmap Is an Operating-System Memory Mechanism

mmap is not a Go heap optimization.

The mapped pages are managed through the OS virtual-memory subsystem.

This means their behavior involves:

- page faults;
- page cache;
- TLB;
- virtual memory;
- file lifetime;
- mapping lifetime.

Go merely accesses the mapped bytes.

---

## 3. Mapping Does Not Mean All Data Is Resident

Mapping a 100-GiB file does not necessarily consume 100 GiB of RSS immediately.

Pages generally become resident as they are accessed.

This creates demand paging.

Thus:

```text
virtual mapping size
```

and:

```text
resident memory
```

are different.

---

## 4. Page Faults

The first access to a page may fault so the OS can establish/populate the mapping.

If the page is not already in memory, disk/storage IO may be required.

Therefore mmap latency can be highly workload-dependent.

Sequential warm scans and random cold scans behave very differently.

---

## 5. Page Cache

File-backed mappings typically interact with the OS page cache.

This means mmap does not bypass all copying in the storage stack.

"Zero-copy" in this context means avoiding an explicit application-level read into a second buffer.

The storage subsystem still moves data into physical memory.

---

## 6. Sequential Access

Sequential mapping access can work well because:

- page faults are predictable;
- OS readahead may help;
- CPU hardware prefetch can help;
- TLB/page locality is good.

Large read-mostly indexes often fit this model.

---

## 7. Random Access

Random access across a large mapping can create:

- frequent page faults;
- TLB misses;
- cache misses;
- storage IO.

A mapped file is not automatically faster than explicit buffered reads.

Access pattern remains the dominant factor.

---

## 8. Mapping Lifetime

A Go slice/view over mapped memory is valid only while the mapping exists.

Conceptually:

```text
mmap
 ↓
Go views created
 ↓
all views stop being used
 ↓
munmap
```

Reversing the order creates use-after-unmap.

---

## 9. Zero-Copy String Views

A mapping can be combined with:

```go
unsafe.String(...)
```

to expose string views without copying.

Now lifetime coupling becomes stronger:

```text
string lifetime
≤
mapping lifetime
```

If the mapping is unmapped first, the string points to invalid memory.

---

## 10. Ownership

Mapped memory is not owned by the Go heap.

The mapping object/resource must own the lifetime.

A safe API often wraps the mapping:

```go
type Mapping struct {
    data []byte
}
```

and prevents callers from using views after close/unmap.

Public raw slices can make lifetime harder to enforce.

---

## 11. Mutable Mappings

Some mappings permit writes.

Now shared state exists between:

- Go code;
- file/page cache;
- possibly other processes.

Concurrency and durability semantics become OS-specific concerns.

This is very different from an ordinary private Go byte slice.

---

## 12. Read-Only Mappings

Read-only mappings are easier to reason about.

They fit:

- immutable indexes;
- static assets;
- large binary datasets.

They also combine naturally with read-only zero-copy views.

---

## 13. File Truncation and External Mutation

A file backing a mapping can sometimes change independently of the mapping.

Behavior after truncation or incompatible external modification is platform-specific and may cause serious faults when accessing invalid regions.

Mapped-file designs should control file lifecycle carefully.

Do not assume that holding a Go slice makes the underlying file stable.

---

## 14. Offset Alignment

Mapping APIs generally require offsets aligned to OS page/granularity requirements.

Low-level wrappers must account for this.

If the desired logical region begins at an unaligned offset, code may need to:

```text
map aligned base
+
slice to logical start
```

This is an implementation detail that belongs inside a mapping abstraction.

---

## 15. RSS Accounting

Mapped resident pages contribute to process RSS/accounting depending on OS semantics.

They are not ordinary Go heap allocations.

Therefore:

```text
Go heap profile
```

may look small while:

```text
process RSS
```

is large.

This is expected for mmap-heavy applications.

---

## 16. GOMEMLIMIT

Go's runtime memory limit is not a hard limit for arbitrary mapped memory.

A process using large mappings must have process-level memory observability and capacity planning.

Do not expect GOMEMLIMIT alone to prevent an mmap-driven memory-pressure event.

---

## 17. GC Interaction

The GC sees the Go slice/header that references mapped bytes.

It does not trace the mapped byte region as ordinary Go heap objects.

This can make very large read-only datasets cheap from GC-scanning perspective.

But physical memory costs remain.

---

## 18. Mapping and Pointer Storage

Mapped files usually should not contain raw Go heap pointers.

Go pointers are process/runtime-managed addresses with GC/lifetime semantics.

Persistent/mapped data should use stable representations such as:

- offsets;
- indexes;
- encoded identifiers.

This can also improve portability.

---

## 19. Offset-Based Data Structures

mmap works well with data structures that use offsets:

```text
base address
+
offset
→ record
```

instead of in-memory Go pointers.

Benefits:

- mapping relocatability;
- persistence;
- pointer-free representation;
- lower GC interaction.

This resembles pointer→index techniques used in heap optimization.

---

## 20. Copying Can Still Be Better

A small hot working set copied from a huge mapping into compact Go memory may outperform repeated random access to the mapping.

Reasons:

- locality;
- cache;
- simpler lifetime;
- fewer page faults.

mmap and copy can coexist.

---

## 21. Warm vs Cold Benchmark

mmap benchmarks must distinguish:

```text
cold pages
vs
warm page cache
```

Otherwise results are difficult to interpret.

A benchmark run immediately after another run may mostly test cached memory rather than storage behavior.

---

## 22. OS Readahead

Sequential access may trigger readahead.

Random access may defeat it.

Therefore benchmark results depend on OS/kernel behavior and file layout.

Record environment details for serious comparisons.

---

## 23. Huge Mappings

Large mappings consume virtual address space.

On 64-bit processes this is often plentiful, but address-space and page-table costs still exist.

The amount of mapped virtual memory should not be confused with physical memory residency.

---

## 24. TLB

Large random mappings can place pressure on the TLB.

This links mmap directly to the CPU/memory chapters.

Possible mitigations are usually representation/access-pattern changes rather than Go-level pointer tricks.

---

## 25. Prefetch / Advice

Operating systems may provide hints/advice for expected mapping access patterns.

These are platform-specific optimizations.

Use only after measuring page-fault/access behavior.

Application correctness should not depend on the advice being honored.

---

## 26. Synchronization

If multiple goroutines read an immutable mapping, synchronization can be minimal.

If they write, ordinary Go synchronization may protect logical invariants, but file/mapping visibility and external-process semantics may add another layer.

Clearly define ownership and mutability.

---

## 27. Unmapping

Unmapping should occur only when no goroutine can access any derived view.

This is often easiest with an explicit lifecycle:

```text
Open
↓
Use
↓
Stop workers
↓
Close/Unmap
```

Reference-counted/manual borrowed APIs are possible but more complex.

---

## 28. Finalizers Are Not Lifetime Control

Do not rely on a finalizer to unmap at a precise time when correctness depends on mapping lifetime.

Explicit close/unmap should own deterministic cleanup.

Cleanup mechanisms can be defensive fallback, not primary ownership.

---

## 29. Go API Choice

Go's standard library does not expose one portable high-level mmap abstraction across all systems.

Unix-oriented projects often use platform APIs or `golang.org/x/sys/unix`.

This means mmap packages should isolate OS-specific code.

---

## 30. Cross-Platform Differences

Mapping semantics vary among:

- Linux;
- BSD/macOS;
- Windows.

A package claiming portability needs platform-specific implementation/testing.

Do not extrapolate Linux behavior into a universal contract.

---

## 31. Error Handling

Mapping can fail because of:

- invalid alignment;
- permissions;
- address-space limits;
- file state;
- resource limits.

Treat mmap as a resource acquisition operation with explicit errors and cleanup.

---

## 32. When mmap Fits

Good candidates often include:

- large immutable files;
- database/index pages;
- static lookup tables;
- assets with random read access where page cache behavior is useful.

Poor candidates may include:

- tiny files;
- short-lived data;
- workflows that immediately copy everything anyway;
- mutable data with unclear ownership.

---

## 33. Performance Evidence

Measure:

- throughput;
- latency;
- page faults;
- RSS;
- CPU;
- storage IO;
- warm/cold behavior.

Do not benchmark only `data[i]` after the entire mapping is warm and claim storage performance.

---

## 34. Related Sources

- `golang.org/x/sys/unix`: https://pkg.go.dev/golang.org/x/sys/unix
- Go `unsafe`: https://pkg.go.dev/unsafe
- OS-specific mmap documentation should be consulted for the target platform.

---

## 35. Engineering Perspective

mmap replaces explicit buffer ownership with virtual-memory ownership.

It can remove copies and reduce Go-heap pressure, but it moves performance responsibility into:

```text
page faults
page cache
TLB
mapping lifetime
OS behavior
```

It is most effective when the application's data representation and access pattern are designed for that model.
