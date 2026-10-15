# Devirtualization

## 1. Interface Dispatch

Consider:

```go
func consume(r io.Reader, buf []byte) {
    _, _ = r.Read(buf)
}
```

The static type is `io.Reader`.

At runtime the concrete value may be:

- `*os.File`;
- `*bytes.Reader`;
- a network type;
- a custom implementation.

A generic interface call therefore uses dynamic dispatch.

---

## 2. Direct vs Indirect Calls

A direct call has a known target:

```text
call bytes.Reader.Read
```

An interface call may be indirect:

```text
receiver method resolved dynamically
```

The indirect call itself has some cost.

But the larger performance effect is often loss of optimization context.

---

## 3. Why Indirect Calls Block Optimization

Inlining generally requires a known callee.

If the compiler cannot identify the target:

```text
cannot inline target
```

That can also weaken:

- escape analysis;
- constant propagation;
- BCE;
- dead-code elimination.

Therefore interface cost can be much larger than one dispatch instruction.

---

## 4. Static Devirtualization

Sometimes source code uses an interface but the compiler can still prove the concrete receiver.

Example:

```go
f, _ := os.Open(name)

var r io.Reader = f

use(r)
```

If the dynamic type is statically known at the relevant call site, the compiler may replace the interface call with a direct call.

This is static devirtualization.

---

## 5. Devirtualization Enables Inlining

The key chain is:

```text
interface call
   ↓
concrete receiver discovered
   ↓
direct call
   ↓
inline candidate
```

After inlining, caller-specific facts become visible.

Thus devirtualization often matters because of what comes next.

---

## 6. Abstraction Is Not Automatically Expensive

A common anti-pattern is:

> Interfaces are slow, so remove them from all hot code.

This can damage design unnecessarily.

If the compiler devirtualizes the hot call, the abstraction may have little runtime cost.

The correct approach is to inspect measured hot paths.

---

## 7. Profile-Guided Devirtualization

Static analysis cannot always determine one concrete receiver.

Production profiles can still reveal:

```text
95% of calls → concrete type A
5%            → other types
```

PGO can use this information to create a hot direct-call path while retaining a generic fallback.

Conceptually:

```text
if receiver is hot concrete type:
    direct call
else:
    original interface dispatch
```

The exact generated form is compiler-dependent.

---

## 8. Conditional Fast Path

Profile-guided devirtualization is interesting because it preserves abstraction semantics.

Cold receiver types still work.

The compiler simply specializes the common path.

This is often preferable to manually encoding type switches solely for performance.

---

## 9. Devirtualization and Escape

Suppose a hot interface method accepts a pointer to temporary data.

If the call remains opaque, escape analysis may be conservative.

After devirtualization and inlining, the compiler can see what the concrete method actually does.

This may eliminate allocations that appeared unrelated to interface dispatch.

---

## 10. Devirtualization and Generics

Generics and interfaces solve different abstraction problems.

Generic code may still involve shape/dictionary implementation details.

Interface code may be devirtualized.

Therefore simplistic rules such as:

```text
generics = zero-cost
interfaces = slow
```

are not reliable.

Generated code and profiles matter.

---

## 11. Diagnosing Dynamic Dispatch

Useful evidence includes:

- CPU profile showing hot interface method calls;
- compiler diagnostics;
- assembly showing indirect calls;
- PGO comparison.

The goal is to answer:

> Is dynamic dispatch actually surviving on this hot path, and is it preventing valuable downstream optimization?

---

## 12. Manual Type Switches

A manual type switch can create specialized direct paths:

```go
switch x := r.(type) {
case *bytes.Reader:
    ...
default:
    ...
}
```

But this has costs:

- source complexity;
- coupling to implementations;
- maintenance burden.

Do not write it merely because devirtualization exists as a concept.

Prefer compiler/PGO specialization when available.

---

## 13. API Boundaries

Interfaces remain valuable at architecture boundaries.

A useful design pattern is often:

```text
generic abstraction at boundary
        ↓
concrete specialized hot internals
```

This preserves maintainability without forcing every low-level loop through dynamic dispatch.

---

## 14. Version Sensitivity

Devirtualization capabilities evolve.

A call that remains indirect in one Go release may become direct in another.

Therefore performance-driven interface refactors should be revalidated during toolchain upgrades.

---

## 15. Common Misconceptions

### "Interface call means allocation"

Not necessarily.

### "Interface call is always indirect at machine code"

Not necessarily.

### "Removing interfaces always improves hot code"

Not necessarily.

### "PGO requires removing abstractions manually"

No; one major value of PGO is helping the compiler specialize real hot behavior.

---

## 16. Engineering Perspective

The real cost of dynamic dispatch is often:

```text
lost compiler knowledge
```

rather than the dispatch instruction alone.

Devirtualization restores that knowledge and can reconnect the optimization chain.
