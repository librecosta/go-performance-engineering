# Go Compiler Pipeline

## 1. Why the Compiler Matters for Performance

Go source code does not map one-to-one to machine instructions.

Between:

```go
result := process(input)
```

and the final CPU instructions, the compiler performs several stages of analysis and transformation.

A useful high-level model is:

```text
Go source
   ↓
parsing and type checking
   ↓
compiler IR
   ↓
devirtualization / inlining
   ↓
escape analysis
   ↓
SSA construction
   ↓
optimization passes
   ↓
architecture-specific lowering
   ↓
register allocation
   ↓
machine code
```

For performance engineering, this matters because many costs visible at runtime exist only when the compiler cannot prove that they are unnecessary.

Examples:

```text
cannot prove index is safe
→ keep bounds check

cannot prove object lifetime
→ allocate on heap

cannot determine concrete receiver
→ keep indirect interface dispatch

cannot inline
→ lose caller context for later optimization
```

Compiler-aware performance engineering is therefore often a problem of **making useful facts provable**.

---

## 2. Source Code Is a Contract, Not a Machine-Code Description

A Go expression tells the compiler what behavior must be preserved.

It usually does not prescribe how that behavior is implemented.

For example:

```go
x := new(Foo)
```

does not mean:

> Allocate Foo on the heap.

It means:

> Produce a pointer to a zero-valued Foo with the required lifetime.

If the compiler proves the lifetime is local, storage may remain on the stack.

Similarly:

```go
v := s[i]
```

requires:

> Panic if `i` is outside the slice bounds.

It does not require:

> Emit a separate runtime bounds check at this exact source location.

If safety is already proven, the check can disappear.

---

## 3. Front End

The compiler front end is responsible for understanding the program as Go.

Important stages include:

- parsing;
- name resolution;
- type checking;
- generic instantiation/lowering;
- building internal representations.

At this stage the compiler gains facts such as:

- concrete types;
- interface relationships;
- constant values;
- control-flow structure.

Those facts are later used by optimization passes.

---

## 4. Intermediate Representation

Compilers rarely optimize directly on source syntax.

They convert programs into an internal representation that makes semantic relationships easier to analyze.

This representation separates:

```text
what the program means
```

from:

```text
how the source happened to be written
```

This is important because many different source forms may become equivalent after lowering.

---

## 5. Devirtualization Before Deeper Optimization

Interface calls can hide the concrete callee.

Conceptually:

```go
func consume(r io.Reader, b []byte) {
    r.Read(b)
}
```

contains an indirect method call.

If the compiler can prove that `r` is always a specific concrete type at a call site, it may replace the interface dispatch with a direct call.

This is devirtualization.

Why do it early?

Because a direct call can become an inlining candidate.

And inlining may expose more information for:

- escape analysis;
- constant propagation;
- bounds-check elimination;
- dead-code elimination.

The important relationship is:

```text
devirtualization
      ↓
direct call
      ↓
inlining
      ↓
more optimization context
```

---

## 6. Inlining Before Escape Analysis

Inlining substitutes the body of a callee into the caller.

Example:

```go
func value(p *Point) int {
    return p.X
}

func work() int {
    p := &Point{X: 10}
    return value(p)
}
```

After conceptual inlining:

```go
func work() int {
    p := &Point{X: 10}
    return p.X
}
```

The compiler now sees the lifetime of `p` in the caller.

That additional context can change escape decisions.

This is why inlining is more than a call-overhead optimization.

It is an **enabling optimization**.

---

## 7. Escape Analysis

Escape analysis asks whether a value can safely remain in stack-managed lifetime.

Conceptually, the compiler tracks how references flow:

```text
local variable
   ↓ address taken
parameter
   ↓ returned/stored/captured?
```

If a pointer can outlive the stack frame that owns the object, heap allocation is required.

Escape analysis is discussed in detail in:

- [Escape Analysis](./escape-analysis.md)

---

## 8. SSA Construction

After high-level transformations, the compiler converts code into Static Single Assignment form.

SSA makes data flow explicit.

Conceptually:

```go
x := 1

if cond {
    x = 2
}

return x
```

becomes something like:

```text
x0 = 1

if cond:
    x1 = 2

x2 = φ(x0, x1)

return x2
```

Each SSA value has one definition.

This makes optimizations easier because the compiler can precisely reason about:

- where values come from;
- whether a value is constant;
- which branch dominates another;
- whether a result is unused;
- whether a check is redundant.

SSA is covered in:

- [Static Single Assignment](./ssa.md)

---

## 9. Machine-Independent Optimizations

Once code is in SSA form, the compiler can perform transformations that are not tied to one CPU architecture.

Examples include:

- constant propagation;
- dead-code elimination;
- branch simplification;
- nil-check elimination;
- bounds-check elimination;
- value rewriting.

The common theme is:

> Remove work whose result or safety is already known.

---

## 10. Dominance and Proof

Control flow provides proof information.

Example:

```go
if len(b) < 8 {
    return
}

use(b[7])
```

The successful path is dominated by the condition:

```text
len(b) >= 8
```

Therefore the compiler may not need another independent bounds check for `b[7]`.

This is an example of how source structure affects what the optimizer can prove.

---

## 11. Architecture-Specific Lowering

SSA operations are still abstract.

Eventually they must become instructions for:

- amd64;
- arm64;
- riscv64;
- other supported architectures.

An abstract operation such as:

```text
integer add
atomic operation
memory copy
```

may lower differently on different CPUs.

Therefore:

> Source-level performance conclusions can be architecture-sensitive even when Go semantics are identical.

---

## 12. Register Allocation

The CPU has a limited number of registers.

The compiler decides which values remain in registers and which must be spilled to memory.

Register pressure can therefore affect performance.

Large inlined functions or many simultaneously live values can increase pressure.

This is one reason:

```text
more inlining
```

is not automatically always better.

Inlining must balance:

- call removal;
- optimization opportunity;
- code size;
- register pressure;
- instruction-cache effects.

---

## 13. Machine Code Is the Final Evidence

When compiler diagnostics are insufficient, machine code answers concrete questions:

- Did the call disappear?
- Is interface dispatch still indirect?
- Is the bounds panic path still present?
- Which atomic instruction was emitted?
- Did the compiler fold the branch?

Common tools include:

```bash
go build -gcflags='-S' ./...
```

and:

```bash
go tool objdump -s 'package\.Function' ./binary
```

Assembly inspection is powerful, but should normally be the end of a diagnostic chain rather than the beginning.

---

## 14. Compiler Optimizations Form a Graph

It is tempting to think of compiler optimizations as independent tricks:

```text
inlining
escape analysis
BCE
devirtualization
```

In practice they enable each other.

A common chain is:

```text
concrete type discovered
        ↓
devirtualization
        ↓
direct call
        ↓
inlining
        ↓
caller facts exposed
        ↓
escape improves
BCE improves
constant propagation improves
        ↓
dead code disappears
```

This is one of the most important ideas in compiler-aware performance engineering.

---

## 15. Why "Readable Code" Can Also Be Optimizable Code

Compiler-friendly Go often resembles good engineering style:

- clear control flow;
- explicit guards;
- simple hot helpers;
- concrete invariants;
- limited hidden mutation.

This does not mean "write code for the compiler" everywhere.

It means avoiding unnecessary opacity in genuinely hot paths.

---

## 16. Version Sensitivity

Compiler heuristics change.

The following are not stable language guarantees:

- whether one function is inlined;
- whether one allocation escapes;
- whether one bounds check is removed;
- how generics are instantiated internally;
- whether one interface call is devirtualized;
- how one operation lowers to machine instructions.

Therefore compiler-sensitive performance work must be validated on the target Go version.

---

## 17. Engineering Perspective

The compiler is not a black box that merely translates syntax.

It is an active participant in the performance model.

A useful question is not:

> How many operations are written in the source?

It is:

> After the compiler has proven everything it can, what work still reaches the CPU?
