# Zero-Copy

## 1. What "Zero-Copy" Actually Means

Zero-copy is often described as:

> Avoid copying bytes.

That description is incomplete.

A copy normally creates independent ownership.

Without a copy, two layers often share the same storage.

Therefore the deeper transformation is:

```text
independent buffers
        ↓
shared representation
```

Zero-copy is fundamentally an **ownership and lifetime optimization**.

---

## 2. Ordinary Copy Semantics

Consider:

```go
s := string(b)
```

At the language level, the resulting string must behave as an immutable string independently of later mutable slice operations.

A copy is a straightforward way to preserve that contract.

Conceptually:

```text
[]byte storage A
      ↓ copy
string storage B
```

The two lifetimes can now evolve independently.

---

## 3. Zero-Copy `[]byte` → `string`

Modern Go exposes:

```go
unsafe.String(
    unsafe.SliceData(b),
    len(b),
)
```

to construct a string view over existing bytes.

Conceptually:

```text
        backing bytes
        ↑          ↑
     []byte      string
```

No second byte array is created.

Potential savings:

- allocation;
- memcpy;
- memory bandwidth.

---

## 4. Immutability Contract

The returned string is semantically immutable.

Therefore the backing bytes must remain unchanged for as long as the string is reachable.

Invalid pattern:

```go
b := getBuffer()
s := unsafe.String(unsafe.SliceData(b), len(b))

b[0] = 'X'
use(s)
```

The string now observes mutation that ordinary string semantics would not suggest.

---

## 5. Pool Reuse Hazard

A particularly dangerous pattern:

```go
buf := pool.Get().([]byte)
fill(buf)

s := unsafe.String(unsafe.SliceData(buf), len(buf))

pool.Put(buf[:0])

return s
```

The pool may hand `buf` to another goroutine.

That goroutine mutates the backing bytes.

The supposedly immutable string changes underneath the caller.

Possible consequences:

- corrupted output;
- races;
- inconsistent map-key behavior;
- security bugs.

---

## 6. Ownership Transfer

A safe zero-copy design may use ownership transfer.

For example:

```text
buffer created
↓
no future writer retains access
↓
string view published
```

This works because mutation rights are surrendered.

The key is not merely keeping memory alive.

The key is controlling who may write it.

---

## 7. Borrowed Views

Another model is borrowing:

```text
view is valid only during callback
```

Example conceptually:

```go
func WithStringView(b []byte, fn func(string)) {
    // view cannot escape the controlled lifetime
}
```

Restricting view lifetime can make ownership easier to reason about.

API design matters as much as conversion mechanics.

---

## 8. `string` → `[]byte`

The reverse direction is more dangerous.

A slice type implies writable bytes.

Constructing a slice view over string storage can let code mutate memory that should be immutable.

Possible problems include:

- violation of string semantics;
- modifying shared/interned/literal-like storage assumptions;
- races;
- unexpected crashes on protected storage depending on origin/platform.

A writable zero-copy string→bytes conversion should be treated as high risk.

---

## 9. Read-Only Byte Views

Sometimes native or low-level code only needs read access to string bytes.

Using `unsafe.StringData` to obtain a pointer for a bounded native operation may be reasonable under a strict lifetime contract.

The operation must not write through that pointer.

---

## 10. Retention Is the Opposite Side of Zero-Copy

Suppose a parser receives:

```text
16 MiB input
```

and needs to keep one:

```text
100-byte token
```

A zero-copy substring/view may keep the full 16 MiB backing storage alive.

A copy costs 100 bytes and may release megabytes.

Thus:

```text
zero-copy
```

can use **more memory** than copying.

---

## 11. Short-Lived vs Long-Lived Views

Zero-copy is strongest when:

```text
source and view have nearly identical lifetime
```

Example:

```text
parse one request buffer
use token immediately
discard both
```

It becomes less attractive when:

```text
source is huge
view is tiny
view is long-lived
```

Lifetime ratio is an important design variable.

---

## 12. Large vs Small Copy

Copy cost scales with bytes.

A 16-byte copy is usually not comparable to a 16-MiB copy.

Before introducing unsafe ownership, measure the actual distribution:

```text
p50 size
p90 size
p99 size
```

A benchmark using only maximum-size payloads can exaggerate the expected system benefit.

---

## 13. Memory Bandwidth

Large copies consume memory bandwidth.

For streaming/high-throughput workloads, memory bandwidth can become the bottleneck even when CPU instruction cost is small.

Zero-copy may therefore help more than allocation metrics alone suggest.

Possible evidence:

- CPU profile;
- throughput;
- perf/PMU bandwidth counters;
- end-to-end service benchmark.

---

## 14. Cache Effects

Copying data can populate cache.

This is not always bad.

If the copied representation is immediately consumed, the copy may produce a compact hot buffer with good locality.

Conversely, sharing a large source mapping may cause scattered accesses and page faults.

"Zero-copy" does not mean "zero memory-system cost".

---

## 15. Operating-System Zero-Copy

The term also appears in OS/networking contexts.

Examples can include avoiding userspace copies through kernel-supported mechanisms.

Those techniques have different ownership models from `unsafe.String`.

Do not treat all zero-copy mechanisms as one performance category.

---

## 16. mmap as Zero-Copy

A memory-mapped file lets a process access file-backed pages through virtual memory rather than explicitly reading into a separately allocated Go buffer.

This can remove explicit copy/read buffers.

But costs remain:

- page faults;
- TLB;
- cache;
- mapping lifetime;
- IO.

See:

- [mmap](./mmap.md)

---

## 17. Native Memory Views

C libraries may provide pointers to native buffers.

`unsafe.Slice` can create Go views.

Again, the view does not own the memory.

The native owner must keep storage valid.

If C reallocates/frees it, the Go slice becomes invalid.

---

## 18. Zero-Copy Serialization

Some serializers can write directly into caller-provided destination buffers.

This is often safer than unsafe aliasing.

Example:

```go
func AppendEncode(dst []byte, v Value) []byte
```

This reduces intermediate copies/allocations while keeping ordinary Go ownership.

Safe representation/API changes should generally be considered before unsafe views.

---

## 19. Scatter/Gather

Some IO APIs accept multiple buffers.

This can avoid concatenating:

```text
header + payload
```

into one temporary buffer.

Conceptually:

```text
writev([header, payload])
```

can reduce userspace copying while retaining clear ownership.

This is another example where zero-copy thinking does not require unsafe conversion.

---

## 20. Copy-on-Write

Copy-on-write shares storage until mutation.

This can reduce copying in read-heavy paths.

But it adds ownership/state complexity:

```text
shared?
↓
mutation requested
↓
copy before write
```

Go does not provide a general automatic COW container, so applications implementing this pattern must define synchronization and lifetime carefully.

---

## 21. Map Keys

A zero-copy string used as a map key is particularly sensitive to mutation.

Map hashing assumes key value semantics remain stable.

If backing bytes mutate after insertion, the logical string value changes while the map's internal assumptions were based on the old contents.

This can create extremely confusing behavior.

Never publish a mutable-backed string key.

---

## 22. Concurrent Readers

Multiple readers can safely share immutable backing bytes.

The main danger is hidden writers.

A useful ownership invariant is:

```text
many readers
zero writers
```

If that cannot be guaranteed, copy.

---

## 23. API Boundary Risk

A private helper can rely on a local invariant more safely than a public API whose callers are unconstrained.

Public zero-copy APIs should make lifetime semantics explicit.

Examples of possible designs:

- callback-scoped borrow;
- documented immutable ownership transfer;
- object type that owns the source buffer.

---

## 24. Benchmark Design

A zero-copy benchmark should compare:

```text
safe copy
vs
zero-copy
```

under realistic:

- byte sizes;
- lifetime;
- concurrency;
- reuse;
- retention.

Measure:

```text
ns/op
B/op
allocs/op
RSS/live heap
end-to-end throughput
```

---

## 25. Microbenchmark Trap

A conversion benchmark may show:

```text
safe: 100 ns
unsafe: 1 ns
```

This is locally dramatic.

But if the conversion occurs once inside a 1-ms operation:

```text
maximum system improvement is tiny
```

Do not accept unsafe complexity based only on local ratio.

---

## 26. Retention Benchmark

When views may outlive source operations, include a memory-retention experiment.

A correct benchmark may intentionally keep tokens live and compare retained heap.

This can reveal cases where copying wins despite worse local `ns/op`.

---

## 27. Correctness Tests

Zero-copy tests should exercise:

- empty input;
- small input;
- large input;
- mutation attempts;
- pool reuse;
- lifetime boundaries;
- concurrent access where relevant.

The benchmark proves speed.

Tests prove the ownership model.

---

## 28. Comments

Unsafe zero-copy code needs stronger comments than ordinary performance code.

Bad:

```go
// Faster zero-copy.
```

Good:

```go
// Returns a zero-copy string view over b.
// b must remain immutable and must not be reused while the string is reachable.
```

The comment must preserve the safety contract.

---

## 29. Version Sensitivity

Public unsafe functions are documented APIs, but compiler/runtime optimization behavior around ordinary conversions can evolve.

A conversion that allocates in one Go version may be optimized differently in another context/version.

Revalidate before retaining a workaround forever.

---

## 30. Decision Model

A useful decision tree:

```text
copy measured hot?
      │
     no → keep safe copy
      │
     yes
      ↓
safe API restructuring possible?
      │
     yes → prefer it
      │
     no
      ↓
ownership/lifetime fully controlled?
      │
     no → copy
      │
     yes
      ↓
zero-copy candidate
      ↓
benchmark + retention + correctness validation
```

---

## 31. Related Official Sources

- `unsafe`: https://pkg.go.dev/unsafe
- `runtime.KeepAlive`: https://pkg.go.dev/runtime

---

## 32. Engineering Perspective

The best way to think about zero-copy is:

> A copy buys ownership independence.

When the copy is removed, the program must pay for that independence with explicit lifetime and mutation rules.

Zero-copy is worthwhile when those rules are simple and the eliminated data movement is materially expensive.
