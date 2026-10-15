# Unsafe

## 1. Why `unsafe` Exists

Go deliberately provides a strong memory-safety model:

- pointer types are checked;
- slice bounds are checked;
- object lifetime is managed by the runtime;
- the garbage collector understands ordinary Go references;
- package boundaries constrain access to implementation details.

`unsafe` exists for the places where those abstractions are not sufficient.

Typical examples include:

- operating-system interfaces;
- memory-mapped data;
- serialization and parsing internals;
- foreign-function interfaces;
- runtime implementation;
- carefully controlled zero-copy views.

The package is named `unsafe` because using it transfers part of the compiler/runtime correctness burden to the programmer.

The important performance lesson is:

> `unsafe` is not an optimization category. It is an escape hatch that may be useful after a specific safe abstraction cost has been measured.

---

## 2. What Safety Is Being Bypassed?

Depending on the operation, `unsafe` may bypass or weaken assumptions about:

- type identity;
- alignment;
- object bounds;
- immutability;
- ownership;
- lifetime;
- GC visibility.

A piece of unsafe code can be correct only if the programmer reconstructs the relevant invariant manually.

For example, a zero-copy `[]byte` → `string` conversion must preserve the invariant that the bytes cannot be mutated while the string is reachable.

The type system would normally enforce that independence through copying.

The unsafe version replaces the copy with a lifetime/ownership contract.

---

## 3. `unsafe.Pointer`

`unsafe.Pointer` is the bridge between typed pointers.

Conceptually, it permits conversions that ordinary Go typing would reject.

Examples include interpreting memory as another representation or passing an address to low-level APIs.

But `unsafe.Pointer` still participates in Go's pointer/lifetime model.

It is not equivalent to an arbitrary integer address.

---

## 4. `uintptr` Is Not a GC Reference

One of the most important rules in low-level Go is:

> `uintptr` is an integer, not a Go pointer.

Consider:

```go
p := &obj
u := uintptr(unsafe.Pointer(p))
```

The programmer may know that `u` contains an address.

The garbage collector does not treat that integer as a live reference.

If all ordinary Go references disappear:

```text
object
 ↓
only integer address remains
```

the integer does not keep the object alive.

This is why long-lived pointer storage through `uintptr` is dangerous.

---

## 5. Pointer → Integer → Pointer

Unsafe pointer arithmetic historically used patterns such as:

```go
unsafe.Pointer(uintptr(p) + offset)
```

The validity rules are strict.

A pointer cannot generally be converted to `uintptr`, stored for an arbitrary time, and later assumed to identify a still-live Go object.

The problem is not only whether the numeric address changed.

The problem is that the GC may no longer see a reference.

---

## 6. `unsafe.Add`

Modern Go provides:

```go
unsafe.Add(ptr, offset)
```

for pointer arithmetic.

This makes intent clearer and avoids many historical `uintptr` arithmetic patterns.

Example:

```go
next := unsafe.Add(ptr, fieldOffset)
```

It is still unsafe.

The caller remains responsible for:

- valid allocation bounds;
- alignment;
- target representation;
- lifetime.

The benefit is a more structured primitive.

---

## 7. `unsafe.Sizeof`

`unsafe.Sizeof` reports the size of a value's representation.

It is useful for studying:

- struct size;
- padding;
- compact representations;
- layout experiments.

Example:

```go
fmt.Println(unsafe.Sizeof(MyStruct{}))
```

But `Sizeof` alone does not tell the full memory footprint.

It does not directly account for:

- separately allocated pointed-to objects;
- slice backing arrays;
- map storage;
- runtime metadata.

---

## 8. `unsafe.Alignof`

`unsafe.Alignof` exposes the alignment requirement for a value.

Alignment matters because processors and compiler-generated instructions may require or prefer aligned addresses.

A valid low-level reinterpretation must respect the target type's alignment requirements.

---

## 9. `unsafe.Offsetof`

`unsafe.Offsetof` returns a struct field's offset.

It can help validate representation assumptions.

Example:

```go
type Header struct {
    A uint32
    B uint64
}

off := unsafe.Offsetof(Header{}.B)
```

This is useful in interoperability and layout experiments.

Do not replace normal field access with pointer arithmetic simply because offsets can be computed.

---

## 10. `unsafe.Slice`

`unsafe.Slice(ptr, len)` constructs a slice view from a pointer and length.

This is useful when memory originates outside ordinary Go slice creation.

Examples:

- native buffers;
- mmap regions;
- low-level system APIs.

Conceptually:

```text
pointer + length
      ↓
Go slice view
```

The view does not acquire independent ownership.

The underlying memory must remain valid for the entire slice lifetime.

---

## 11. Historical Giant-Array Tricks

Before structured APIs such as `unsafe.Slice`, low-level Go code often used patterns resembling:

```go
(*[1 << 30]T)(unsafe.Pointer(ptr))[:n:n]
```

These techniques should generally be treated as historical.

Modern public unsafe APIs communicate intent more clearly and reduce the number of implementation assumptions in application/library code.

---

## 12. `unsafe.SliceData`

`unsafe.SliceData` provides access to the underlying data pointer for a slice.

This can be useful when crossing a low-level boundary.

For example:

```go
ptr := unsafe.SliceData(buf)
```

The pointer's validity is still coupled to the slice/backing allocation.

A raw data pointer is not an ownership transfer.

---

## 13. `unsafe.String`

`unsafe.String(ptr, len)` constructs a Go string whose bytes begin at `ptr`.

This is the basis for a modern zero-copy `[]byte` → `string` view.

The crucial contract is that the underlying bytes must not be modified while the resulting string is reachable.

Strings are semantically immutable.

Unsafe code must preserve that semantic manually.

---

## 14. `unsafe.StringData`

`unsafe.StringData` exposes a pointer to string data.

This is useful for:

- native calls;
- read-only zero-copy access;
- low-level inspection.

It does not make the string storage writable.

Treating string memory as mutable violates the semantic contract.

---

## 15. Reflect Header Tricks

Older code may manipulate:

- `reflect.StringHeader`;
- `reflect.SliceHeader`.

to reinterpret memory.

Modern code should prefer the dedicated `unsafe` APIs.

Header tricks are especially fragile because they can:

- obscure lifetime relationships;
- create non-GC-visible pointers;
- rely on representation details;
- produce invalid aliasing.

They are useful historical context, not a default modern technique.

---

## 16. Alignment

A pointer that is valid as `*byte` is not automatically valid as `*uint64`.

Low-level reinterpretation must account for alignment.

Misalignment may:

- be slower;
- require special instructions;
- fault on some architectures;
- violate compiler assumptions.

Never extrapolate amd64 tolerance into a cross-platform Go invariant.

---

## 17. Bounds

Unsafe pointer arithmetic can bypass ordinary slice bounds checks.

That means:

```text
compiler no longer protects allocation boundary
```

A pointer can accidentally refer to:

- another object;
- unmapped memory;
- runtime metadata.

Bounds must therefore be derived from trusted representation information.

---

## 18. Aliasing

Unsafe conversion can make two Go values refer to the same memory.

Example:

```text
[]byte
  └────┐
       ↓
same backing bytes
       ↑
string ┘
```

The type system normally lets programmers reason about mutation independently.

Aliasing changes that assumption.

This can create bugs that look impossible from ordinary Go syntax.

---

## 19. Lifetime

An unsafe pointer or view must not outlive the underlying memory.

This matters especially for:

- pooled buffers;
- mmap;
- C memory;
- stack/local storage;
- manually managed regions.

A correct address is useless after the storage has been released or reused.

---

## 20. Ownership

The most useful question when reviewing unsafe code is:

> Who owns the bytes?

Possible owners include:

- Go heap object;
- caller;
- pool;
- mmap;
- C allocator;
- OS resource.

Then ask:

> When may that owner mutate or release them?

If those answers are unclear, the unsafe optimization is not ready.

---

## 21. GC Visibility

Ordinary typed Go pointers communicate reachability to the garbage collector.

Integer representations do not.

Some runtime-internal structures deliberately encode pointers in ways that are invisible to GC, but they operate under specialized runtime invariants.

Third-party code should not copy those techniques casually.

---

## 22. `runtime.KeepAlive`

`runtime.KeepAlive` can extend the observable reachability of a Go object to a particular point.

This is necessary in some low-level lifetime patterns involving:

- finalizers/cleanup;
- syscalls;
- native use.

But `KeepAlive` does not make an otherwise illegal unsafe pointer conversion legal.

It controls lifetime; it does not repair arbitrary pointer provenance.

---

## 23. Unsafe and Escape Analysis

Unsafe code can affect the compiler's ability to track pointer flow.

Low-level runtime/compiler directives can even assert no-escape behavior explicitly.

These mechanisms can influence stack/heap placement.

They are correctness contracts, not ordinary optimization hints.

See:

- [Runtime Internals](./runtime-internals.md)

---

## 24. Unsafe and the Race Detector

Unsafe aliasing can introduce ordinary data races.

Example:

```text
string view shares pooled []byte
↓
another goroutine reuses buffer
↓
read/write race
```

The race detector can catch some of these interactions.

It cannot prove the entire lifetime contract.

---

## 25. `checkptr`

Go can enable additional runtime checking for unsafe pointer misuse through compiler/runtime instrumentation.

This can help catch:

- invalid pointer arithmetic;
- invalid conversions;
- pointers outside expected allocations.

Such instrumentation is a correctness tool.

Its performance numbers are not production baseline numbers.

---

## 26. `go vet`

`go vet` recognizes some problematic unsafe patterns.

A clean vet run is useful, but it is not a proof of unsafe correctness.

Lifetime and ownership mistakes can remain invisible to static pattern checks.

---

## 27. Cross-Architecture Testing

Unsafe code should be more suspicious of architecture assumptions than ordinary Go code.

Questions include:

- pointer width;
- alignment;
- endianness;
- unaligned-load support;
- OS ABI;
- instruction behavior.

Where practical, test at least the architectures the package claims to support.

---

## 28. Encapsulation

Unsafe code should have a small surface.

A healthy structure may look like:

```text
internal/zerocopy/
internal/native/
internal/mmap/
```

rather than unsafe operations scattered through business logic.

This reduces:

- review surface;
- migration cost;
- runtime/compiler compatibility risk.

---

## 29. API Naming

An API that returns an aliased view should not silently look like an ordinary copying conversion when that difference matters to callers.

Example:

```go
func UnsafeBytesToString(b []byte) string
```

makes the contract visible.

An internal helper may avoid a public scary name if ownership is fully contained, but the implementation should still document the invariant.

---

## 30. Performance Evidence

Unsafe is justified only when the safe abstraction cost is meaningful.

Measure:

- copy bytes;
- allocations;
- CPU;
- bandwidth;
- end-to-end impact.

A microbenchmark showing a 100× faster conversion can still be irrelevant if conversion accounts for 0.1% of total request cost.

---

## 31. Maintainability Threshold

Unsafe code should face a higher evidence threshold because the maintenance cost is higher.

A useful decision model is:

```text
expected system gain
        ↓
must exceed
        ↓
correctness + maintenance + portability risk
```

---

## 32. What Not to Do

Do not use `unsafe` merely to:

- remove a bounds check that the compiler can eliminate safely;
- avoid a tiny cold-path copy;
- access private runtime state;
- force an allocation onto the stack;
- demonstrate cleverness.

---

## 33. Stable API vs Internal Runtime

Public `unsafe` APIs are dangerous but documented.

Private runtime internals are a different risk category.

Crossing into `//go:linkname` or undocumented runtime symbols adds compatibility risk on top of memory-safety risk.

---

## 34. Related Official Sources

- `unsafe`: https://pkg.go.dev/unsafe
- `runtime.KeepAlive`: https://pkg.go.dev/runtime#KeepAlive
- compiler directives: https://go.dev/src/cmd/compile/doc.go

---

## 35. Engineering Perspective

Safe Go makes memory ownership largely implicit and compiler/runtime enforced.

Unsafe code makes ownership explicit and programmer enforced.

The value of unsafe optimization comes from removing a measured abstraction cost.

The price is accepting responsibility for the invariant that abstraction used to protect.
