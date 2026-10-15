# cgo

## 1. What cgo Changes

cgo allows Go code to call C code and C code to interact with Go.

This crosses several boundaries simultaneously:

```text
Go type system
Go garbage collector
Go scheduler
C ABI
native ownership
```

The performance cost is not only the call instruction.

The runtime must preserve Go's correctness model across a foreign environment that does not understand Go pointers or goroutines.

---

## 2. When cgo Is Appropriate

Typical reasons include:

- native system libraries;
- existing C/C++ SDKs;
- high-performance codecs;
- database libraries;
- cryptographic implementations;
- hardware/runtime integrations.

The first performance question should not be:

> How do I make cgo calls faster?

It should be:

> Is crossing this boundary necessary, and what granularity should the boundary have?

---

## 3. Call Boundary Overhead

A cgo call is generally more expensive than an ordinary Go function call.

The runtime may need to:

- transition execution state;
- maintain scheduler/runtime invariants;
- handle pointer rules;
- prepare for possible callbacks.

Go has improved cgo overhead across releases, so old benchmark numbers should not be treated as permanent constants.

---

## 4. Call Granularity

A common mistake is crossing the boundary for very small operations.

Example:

```text
for each byte:
    call C
```

Even if C performs the byte operation quickly, boundary overhead can dominate.

Prefer:

```text
batch many operations
↓
one native call
```

where API design permits.

---

## 5. Amortization

Suppose a native function performs 10 ms of heavy work.

A cgo boundary cost is likely negligible.

Suppose a native function performs 20 ns of trivial work.

The boundary may dominate.

Therefore cgo performance depends heavily on work-per-call ratio.

---

## 6. Go Pointers and C

C does not participate in Go GC reachability.

If C stores a Go pointer without following cgo rules, the runtime may:

- lose track of lifetime assumptions;
- be unable to guarantee future address stability;
- violate pointer-safety invariants.

Go therefore imposes explicit pointer-passing rules.

---

## 7. Temporary Pointer Passing

When a Go pointer is passed directly to C for the duration of a cgo call, the runtime/toolchain can enforce required behavior under the documented rules.

The critical condition is whether C keeps the pointer after the call returns.

A temporary borrow is simpler than a retained foreign reference.

---

## 8. Retained Go Pointers

If C must retain a Go pointer beyond one call, the pointed-to object needs an explicit pinning lifetime.

Modern Go provides `runtime.Pinner` for this use case.

Conceptually:

```text
Go object
 ↓
Pin
 ↓
C retains pointer
 ↓
C stops using pointer
 ↓
Unpin
```

The `Pinner` itself must remain alive for the required period.

---

## 9. `runtime.Pinner`

`Pinner.Pin` pins Go objects to a fixed location.

`Pinner.Unpin` releases the pins.

This is an explicit contract:

> Native code may retain this address while the object is pinned.

Even though current Go heap objects are generally non-moving, application correctness should depend on the documented pinning contract rather than an implementation accident.

---

## 10. Pointers Inside Pinned Objects

Pinning one outer object does not magically make arbitrary nested Go pointers safe for C retention.

If C needs to dereference stored Go pointers inside a structure, those referents must also satisfy cgo/pinning requirements.

Foreign memory graphs should remain simple.

---

## 11. `runtime/cgo.Handle`

Often C does not need the actual Go address.

It only needs to preserve an opaque identity and later pass it back.

`runtime/cgo.Handle` is designed for this.

Conceptually:

```text
Go value
 ↓
NewHandle
 ↓
integer-like opaque handle
 ↓
C stores handle
 ↓
Go receives handle
 ↓
Value()
 ↓
Delete()
```

This avoids exposing a Go pointer directly.

---

## 12. Handle Lifetime

A `cgo.Handle` consumes runtime resources until `Delete` is called.

Therefore it requires explicit lifecycle management.

It behaves more like a registry entry than a GC-transparent pointer.

Failure to delete handles creates leaks.

---

## 13. Go Value Containing Pointers

`cgo.Handle` is particularly useful when the Go value contains Go pointers.

Passing the value's raw memory to C may violate pointer rules.

Passing a handle keeps the actual Go object graph on the Go side.

---

## 14. C Memory Used by Go

The opposite direction is also common:

```text
C allocates buffer
↓
Go creates view
```

A Go slice can be constructed over native memory with unsafe mechanisms.

The key invariant is:

```text
C must not free/realloc memory while Go view exists
```

Go GC does not own that memory.

---

## 15. Native Allocation and RSS

Memory allocated by C libraries is not ordinary Go heap memory.

Therefore:

```text
Go heap metrics
```

may not explain process RSS.

Services using substantial cgo memory need process-level/native observability.

GOMEMLIMIT is not a hard cap on all C allocations.

---

## 16. Callback Direction

C may call back into exported Go functions.

This requires runtime preparation and has its own constraints.

Callbacks are much more expensive and complex than ordinary intra-Go calls.

If the native design can avoid per-item callbacks and return batches instead, performance and reasoning often improve.

---

## 17. `#cgo nocallback`

Current cgo supports a `#cgo nocallback` directive for C functions that are guaranteed not to call back into Go.

This allows the toolchain/runtime to skip callback-related preparation.

The contract is strict.

If the C function does call back despite the declaration, program behavior is invalid and current documentation states that such a callback will panic.

This is a low-level performance contract, not a casual hint.

---

## 18. `#cgo noescape`

Current cgo also supports `#cgo noescape`.

Normally, passing a Go pointer to C may force conservative heap lifetime.

If a specific C function:

- does not keep the Go pointer;
- does not pass it back into Go;

then `#cgo noescape` can tell the compiler that the pointer does not escape through that C function.

This can improve stack allocation.

If the declaration is false, the result can be memory corruption or crashes.

---

## 19. noescape Is a Correctness Assertion

The directive does not mean:

> Please optimize aggressively.

It means:

> I guarantee this foreign implementation obeys a specific pointer-lifetime contract.

The compiler may rely on that guarantee when deciding stack lifetime.

A false assertion can create dangling references.

---

## 20. Native Threading

C code may create or use native threads.

The Go runtime has specific rules for callbacks and thread interaction.

Performance behavior may involve:

- OS scheduling;
- runtime scheduler transitions;
- thread affinity.

Do not reason about cgo only at function-call level when native code has its own concurrency model.

---

## 21. Blocking Native Calls

A long-running C call can block the native thread executing it.

The Go runtime scheduler can compensate by running other goroutines on other threads as needed, but the boundary still has scheduling implications.

Large amounts of blocked cgo work can affect:

- OS thread count;
- scheduler behavior;
- resource usage.

---

## 22. cgo and GC

Go objects passed to C require lifetime guarantees.

Pinned objects can affect future GC implementation options and resource behavior.

Native memory itself is outside Go tracing.

This creates a split memory model:

```text
Go-owned
GC managed

C-owned
manual/native managed
```

Bridges between the two must be explicit.

---

## 23. Copy vs Native View

Sometimes the safest approach is copying C data into Go memory.

Costs:

- allocation;
- memcpy.

Benefits:

- independent lifetime;
- GC ownership;
- simpler concurrency;
- no dangling native view.

Zero-copy native views make sense when copy cost is material and lifetime is tightly controlled.

---

## 24. Strings

C strings and Go strings have different representation and lifetime semantics.

Converting between them may allocate/copy.

Avoid repeated per-item string conversion in hot native loops when batching or stable representations can reduce boundary traffic.

---

## 25. Error Handling

Native APIs may report:

- return codes;
- errno;
- callbacks;
- allocated error objects.

Performance-sensitive wrappers should avoid adding repeated conversions/allocations unnecessarily, but correctness and clear error semantics come first.

---

## 26. Batching

The most broadly useful cgo optimization is often batching.

Instead of:

```text
N Go→C transitions
```

perform:

```text
1 transition
N units of work inside C
```

This amortizes boundary overhead and often improves cache behavior.

---

## 27. Avoiding Chatty APIs

If a C library exposes many tiny getters:

```text
getX()
getY()
getZ()
```

a Go wrapper may consider one native function that fills a result structure, if the integration is under your control.

This reduces boundary crossings.

Do not change third-party ABI casually, but wrapper-level aggregation can help.

---

## 28. Benchmarking cgo

Benchmarks should separate:

```text
boundary overhead
native work
copy/conversion overhead
```

Useful tests include:

- empty/minimal C call;
- representative real C call;
- batched version;
- copy vs shared buffer.

Measure end-to-end behavior, not only raw transition ns/op.

---

## 29. Toolchain Version

cgo implementation and overhead evolve with Go releases.

Performance numbers must record:

- Go version;
- OS;
- architecture;
- C compiler;
- optimization flags;
- native library version.

---

## 30. Cross Compilation

cgo complicates cross compilation because it depends on a target C toolchain and ABI.

This is an engineering/maintenance cost of choosing native integration.

Performance gains must justify deployment complexity where alternatives exist.

---

## 31. Race and Sanitizer Boundaries

Concurrency bugs can cross the Go/C boundary.

Go's race detector does not automatically understand all native synchronization.

Native sanitizers and Go instrumentation may need separate testing strategies.

A passing Go race run is not a proof that the C side is race-free.

---

## 32. Encapsulation

Keep cgo inside a small package boundary.

A useful architecture:

```text
internal/native/
    cgo wrapper
    pointer/lifetime policy
        ↓
ordinary safe Go API
```

This prevents foreign lifetime semantics from leaking across the application.

---

## 33. Comments and Contracts

Low-level directives such as:

```text
#cgo noescape
#cgo nocallback
```

must have comments explaining why the C implementation satisfies the contract.

Future native-library changes can invalidate those assumptions.

The contract must remain reviewable.

---

## 34. Related Official Sources

- cgo command: https://pkg.go.dev/cmd/cgo
- `runtime.Pinner`: https://pkg.go.dev/runtime#Pinner
- `runtime/cgo.Handle`: https://pkg.go.dev/runtime/cgo

---

## 35. Engineering Perspective

The most effective cgo optimization is usually not a clever pointer trick.

It is choosing a coarse, explicit boundary:

```text
few crossings
clear ownership
simple pointer graph
batch work
```

The lower the crossing frequency and the clearer the lifetime contract, the easier the integration is to optimize and maintain.
