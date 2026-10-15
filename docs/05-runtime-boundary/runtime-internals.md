# Runtime and Compiler Boundaries

## 1. Why Runtime Internals Are Tempting

The Go runtime contains highly optimized mechanisms for:

- scheduling;
- allocation;
- garbage collection;
- synchronization;
- per-P state;
- lock-free structures;
- stack management.

Performance-oriented developers naturally inspect these mechanisms and ask:

> Can application code use the same trick?

Sometimes the answer is yes through a public API.

Often the answer is no because runtime code operates under stronger private invariants.

Understanding runtime internals is valuable.

Copying runtime internals is a separate decision.

---

## 2. Public Contract vs Private Mechanism

A public API provides a compatibility contract.

A private runtime symbol does not.

This distinction is essential.

Example:

```text
sync.Pool
```

may internally use per-P state.

That does not imply application code should reach into `runtime.procPin` to implement its own per-P cache.

The standard library/runtime can coordinate internal changes together.

Third-party code cannot assume those private contracts remain stable.

---

## 3. `runtime.KeepAlive`

Compiler liveness is not the same as lexical scope.

Example:

```go
p := &Resource{fd: fd}
syscall.Use(p.fd)
```

After extracting `p.fd`, the compiler may no longer need `p` as a live Go reference during the low-level call.

If `p` has a cleanup/finalizer that closes the descriptor, premature cleanup can become possible.

`runtime.KeepAlive(p)` marks the object as reachable until that program point.

---

## 4. Lexical Scope vs Liveness

A variable may still be visible in source scope:

```text
function has not returned
```

but compiler liveness may determine:

```text
there are no future semantic uses
```

These are different concepts.

Low-level lifetime-sensitive code must reason about actual liveness.

---

## 5. What `KeepAlive` Does Not Do

`KeepAlive` does not:

- make invalid pointer arithmetic valid;
- make a `uintptr` into a GC reference;
- repair use-after-unmap;
- synchronize mutable state.

It only extends object reachability to a point.

The ordinary unsafe rules still apply.

---

## 6. Cleanup and Finalization

Runtime cleanup/finalizer mechanisms are GC-triggered and nondeterministic.

They are useful for defensive resource cleanup in specific abstractions.

They are not a replacement for explicit:

```text
Close
Unmap
Release
```

when deterministic lifetime is required.

---

## 7. `runtime.Pinner`

`Pinner` is a public runtime mechanism for explicitly pinning Go objects.

Its existence demonstrates an important design principle:

> Do not rely on current GC implementation details when a public lifetime contract exists.

Even if current objects normally do not move, native retention should use the documented pinning model.

---

## 8. Compiler Directives

Go recognizes directives that change compiler/runtime behavior.

Performance-relevant examples include:

- `//go:noescape`;
- `//go:nosplit`;
- `//go:linkname`.

These directives have very different safety profiles.

They should not be grouped together as generic "compiler hints".

---

## 9. `//go:noescape`

`//go:noescape` applies to a function declaration without a Go body, typically implemented in assembly or another non-Go form.

It tells escape analysis that pointer arguments do not escape into:

- heap storage;
- returned values.

This can allow callers to keep objects on the stack.

---

## 10. noescape Is a Contract

Suppose:

```go
//go:noescape
func nativeUse(p *byte)
```

The compiler may assume `p` cannot remain usable after the call.

If the implementation stores `p` globally, the compiler's stack-placement decision can create a dangling pointer after the caller returns.

Therefore:

```text
noescape
```

means:

> The human guarantees a lifetime fact the compiler cannot inspect.

It is not a speculative optimization request.

---

## 11. `#cgo noescape`

cgo exposes a related directive for C functions.

The same principle applies.

If the native implementation violates the promise, correctness is lost.

This should be reviewed as an FFI safety contract.

---

## 12. `//go:nosplit`

Normal Go function entry includes stack-growth safety machinery.

`//go:nosplit` tells the compiler to omit the usual stack-overflow check for that function.

Runtime code uses this in places where stack growth/preemption machinery itself may be unavailable or unsafe.

---

## 13. `nosplit` Is Not `go:fast`

Removing a stack check may sound like a performance optimization.

For ordinary code, using `//go:nosplit` for speed is unsafe.

Compiler documentation explicitly warns that use outside low-level runtime code can allow stack overflow/corruption.

This is a correctness primitive for runtime implementation.

It is not a general hot-path hint.

---

## 14. `//go:linkname`

`//go:linkname` can bind a local declaration to another package's symbol, including an unexported symbol.

This can bypass:

- package encapsulation;
- type safety;
- compatibility boundaries.

The compiler restricts it to code importing `unsafe`.

That restriction reflects its power.

---

## 15. Private Runtime Symbols

Some high-performance libraries historically use `linkname` to access:

- `runtime.procPin`;
- scheduler/runtime helpers;
- private hashing/allocation internals.

These techniques may work for one Go version.

They create a maintenance obligation:

```text
runtime upgrade
↓
private contract may change
↓
library breaks or corrupts state
```

---

## 16. `runtime.procPin`

`procPin` pins the current goroutine to its P for a critical internal operation and exposes the P identity internally.

This makes per-P data structures possible.

Attractive use cases include:

- per-P counters;
- RNG state;
- caches.

But it is not a normal public application API.

---

## 17. Why Per-P Can Be Fast

Per-P state can reduce:

- global locks;
- shared cache lines;
- contention.

Conceptually:

```text
P0 → local state 0
P1 → local state 1
P2 → local state 2
```

This resembles sharding with scheduler-local ownership.

The idea is useful.

The private implementation hook is not necessarily appropriate.

---

## 18. Prefer Public Alternatives

Before private per-P tricks, consider:

- sharding;
- worker-local state;
- `sync.Pool`;
- batching;
- explicit ownership.

These approaches preserve compatibility and often capture most of the benefit.

---

## 19. Runtime Noescape Tricks

Runtime code has historically used techniques that obscure pointer relationships from escape analysis.

Conceptually:

```text
pointer
↓
integer-like transformation
↓
pointer
```

to prevent the compiler from seeing a relation.

This is extraordinarily dangerous in application code.

If the pointer truly escapes while the compiler believes it does not, stack lifetime may become invalid.

---

## 20. Lying to the Compiler

Compiler optimization facts are also correctness facts.

Forcing:

```text
does not escape
```

when the pointer does escape can produce:

```text
stack object
↓
function returns
↓
external reference remains
↓
memory corruption
```

The potential gain is an allocation.

The potential loss is memory safety.

This trade is unacceptable for ordinary application code.

---

## 21. Tagged Pointers

Runtime lock-free structures may pack:

```text
pointer identity
+
version/tag
```

into an atomic machine word.

This can help solve ABA.

But such code may depend on:

- pointer width;
- virtual-address range;
- alignment;
- non-GC-visible representations.

These are runtime-level assumptions.

---

## 22. GC-Visible vs Non-GC-Visible Pointer Representation

If an address is encoded inside:

```text
uintptr / uint64
```

the GC does not treat it as a normal reference.

Runtime code that does this must arrange lifetime through separate invariants.

Ordinary heap objects cannot safely depend on an encoded integer being their only reference.

---

## 23. Runtime Lock-Free Structures

The runtime contains specialized lock-free stacks and scheduler data structures.

They are useful teaching material for:

- CAS;
- ABA;
- tagging;
- backoff;
- per-P state.

They are not ready-made application templates.

The runtime controls:

- memory placement;
- GC interaction;
- scheduling assumptions.

Application code usually does not.

---

## 24. Backoff

Runtime lock-free algorithms may use architecture-specific spin/backoff behavior.

Those parameters reflect workload and architecture assumptions.

Copying one constant into a different application or CPU does not preserve the same performance model.

---

## 25. Intrinsics

Some functions/directives are recognized specially by the compiler and replaced with optimized machine code.

The standard library/runtime can use these internal compiler contracts.

Third-party code generally should rely on public functions whose optimization behavior the toolchain manages.

---

## 26. Assembly

Assembly is appropriate in specialized libraries when:

- hot loop is proven;
- compiler cannot generate required instruction sequence;
- architecture-specific implementation is justified.

Costs include:

- per-architecture maintenance;
- ABI/compiler changes;
- reduced portability;
- harder review/testing.

---

## 27. Assembly and `//go:noescape`

Assembly declarations often need noescape information because the Go compiler cannot inspect the assembly implementation deeply enough for ordinary escape analysis.

The directive restores a fact known by the human implementation.

This is a valid low-level use.

It still requires correctness review.

---

## 28. `//go:noinline`

`//go:noinline` can be useful for:

- compiler/runtime implementation constraints;
- benchmarks/experiments;
- debugging.

It should not normally be used to "improve performance".

Preventing inlining often removes optimization opportunities.

---

## 29. Link-Time / Compiler Heuristics

Avoid encoding assumptions such as:

```text
this function always inlines
this private symbol always exists
this allocation threshold never changes
```

unless the project intentionally tracks a specific Go toolchain.

Runtime/compiler internals evolve.

---

## 30. Standard Library as a Boundary Example

The standard library can rely on more runtime/compiler cooperation than external modules.

For example, `sync` and runtime internals may share private contracts.

This does not mean those contracts are part of Go's compatibility promise.

When studying source, always ask:

> Is this mechanism public or privileged?

---

## 31. `internal` Packages

Go's own `internal/...` packages are another explicit signal that an API is not intended as a general external contract.

Performance engineers should use them for understanding implementation, not as a dependency target.

---

## 32. Runtime Source Archaeology

Reading runtime source is extremely valuable.

It teaches:

- what the runtime optimizes;
- which costs matter;
- how primitives are implemented;
- where version-sensitive behavior lives.

The correct output of runtime archaeology is usually:

```text
a better cost model
```

not:

```text
copy this private code
```

---

## 33. Version Upgrade Risk

Private runtime dependencies should trigger special upgrade testing.

Potential breakage can include:

- build failure;
- semantic change;
- memory corruption;
- performance regression.

A project using private runtime symbols has accepted a toolchain-maintenance burden.

---

## 34. Encapsulation

If a low-level package truly needs runtime-specific mechanisms, isolate them.

Example:

```text
internal/runtimehack/
    go1.xx-specific implementation
```

Provide a safe higher-level API and explicit version tests.

Do not spread private runtime assumptions throughout the repository.

---

## 35. Comments and Documentation

A low-level directive needs documentation of the contract.

Example noescape comment should answer:

- why pointer does not escape;
- which implementation is covered;
- what future change would invalidate it.

A private runtime dependency should explain:

- why public alternatives are insufficient;
- which Go versions are supported;
- how upgrade compatibility is tested.

---

## 36. Evidence Threshold

The lower the abstraction level, the stronger the evidence required.

Conceptually:

```text
safe source optimization
→ modest proof threshold

unsafe public API
→ higher threshold

assembly / compiler directive
→ stronger benchmark + correctness tests

private runtime linkname
→ exceptional justification
```

---

## 37. Correctness Tooling

Low-level code should use appropriate combinations of:

- normal unit tests;
- race detector;
- checkptr;
- architecture CI;
- fuzzing;
- stress testing;
- assembly inspection.

No single tool proves correctness.

---

## 38. `KeepAlive` and Native Calls

A typical valid use is ensuring a Go wrapper object remains reachable until a low-level operation that only uses one of its raw fields completes.

Current runtime documentation explicitly uses this kind of example.

The key idea is:

```text
last lexical-looking use
≠
last compiler-visible lifetime
```

---

## 39. KeepAlive and Unsafe

Current runtime documentation also explicitly warns that `KeepAlive` does not replace the unsafe pointer validity rules.

This is worth repeating because it is a common misconception.

Lifetime extension and pointer validity are separate invariants.

---

## 40. Related Official Sources

- compiler directives: https://go.dev/src/cmd/compile/doc.go
- `runtime.KeepAlive`: https://pkg.go.dev/runtime#KeepAlive
- `runtime.Pinner`: https://pkg.go.dev/runtime#Pinner
- runtime source: https://go.dev/src/runtime/
- runtime lock-free stack: https://go.dev/src/runtime/lfstack.go

---

## 41. Engineering Perspective

Runtime internals are valuable because they reveal what Go itself considers expensive enough to optimize.

But runtime code is written inside a privileged environment.

The safest lesson to export is the **design principle**:

```text
reduce sharing
reduce allocation
preserve locality
make lifetime explicit
```

not the private hook used by the runtime to implement that principle.
