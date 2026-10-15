# Static Single Assignment

## 1. What SSA Is

SSA stands for Static Single Assignment.

In SSA form, each logical value is assigned exactly once.

This is not how Go source must be written.

It is an internal representation that makes data flow easier for the compiler to reason about.

Example source:

```go
x := 1

if cond {
    x = 2
}

return x + 1
```

Conceptually becomes:

```text
x0 = 1

if cond:
    x1 = 2

x2 = φ(x0, x1)

return x2 + 1
```

The `φ` value represents the value selected from incoming control-flow paths.

---

## 2. Why SSA Helps Optimization

In ordinary source code, a variable name can be assigned multiple times.

The compiler must determine:

- which assignment reaches a use;
- whether another branch overwrites it;
- whether the value is constant;
- whether the value is still live.

SSA makes those relationships explicit.

A use refers directly to one SSA value.

This simplifies:

- constant propagation;
- dead-code elimination;
- redundancy removal;
- bounds proofs;
- branch simplification.

---

## 3. Values and Blocks

SSA programs can be thought of as:

```text
basic blocks
+
SSA values
+
control-flow edges
```

A basic block contains operations executed together before control branches elsewhere.

Example:

```text
entry
  ↓
check condition
 ↙        ↘
then      else
  ↘        ↙
    merge
```

The compiler reasons about which blocks dominate others and which values are available on each path.

---

## 4. Dominance

A block A dominates block B if every path to B passes through A.

This concept is fundamental to optimization.

Example:

```go
if len(b) < 8 {
    return
}

x := b[7]
```

The successful length check dominates the indexed access.

Therefore the compiler knows:

```text
on every path reaching b[7],
len(b) >= 8
```

This creates the proof required for bounds-check elimination.

---

## 5. Phi Values

Consider:

```go
x := 10

if cond {
    x = 20
}

use(x)
```

At the merge point, `x` can come from two definitions.

SSA creates a merged value:

```text
x2 = φ(x0, x1)
```

The compiler can then reason about the possible values separately.

---

## 6. Constant Propagation

Suppose inlining exposes:

```go
const flags = 0

if flags&1 != 0 {
    expensive()
}
```

SSA can determine:

```text
flags&1 == 0
```

The branch is always false.

The compiler can remove it.

Then the body becomes unreachable and dead-code elimination removes `expensive()`.

This illustrates how SSA enables chained optimization.

---

## 7. Dead-Code Elimination

If an SSA value has no observable effect and no required use, it can disappear.

Example:

```go
x := computePureValue()
_ = x
```

If the compiler proves the computation has no side effects and the result is unused, parts or all of the computation may be removed.

This is why badly written microbenchmarks can accidentally benchmark nothing.

---

## 8. Branch Simplification

SSA can simplify branches when conditions become known.

For example:

```text
if true
```

does not require a runtime branch.

Branches may become known through:

- constants;
- previous comparisons;
- type information;
- inlining.

---

## 9. Bounds Checks as SSA Operations

Bounds checks are represented internally as operations that can participate in proof and rewriting.

If a dominating condition already proves safety, the redundant check can be eliminated.

This is why bounds-check elimination is not merely a local peephole optimization.

It depends on control flow.

---

## 10. Nil-Check Elimination

The compiler can also remove redundant nil checks when previous operations already prove non-nil behavior.

As with BCE, the source-level existence of a dereference does not mean a separate runtime check survives.

---

## 11. SSA and Architecture Lowering

Machine-independent SSA operations eventually become architecture-specific operations.

Example:

```text
generic integer add
```

becomes the appropriate CPU instruction.

An atomic operation may lower to different instruction sequences on different architectures.

Thus SSA is the bridge between Go semantics and machine code.

---

## 12. Viewing SSA

For selected functions, Go can emit an SSA visualization.

Typical usage:

```bash
GOSSAFUNC=FunctionName go build ./...
```

The generated HTML lets you inspect transformation stages.

This is useful when a compiler decision is surprising.

Examples:

- why a bounds check remains;
- why a branch was not simplified;
- why a value is still live;
- how a high-level operation lowered.

---

## 13. SSA Is Not a First-Line Profiling Tool

SSA tells you:

> What the compiler did.

It does not tell you:

> Whether this function matters to the application.

Therefore a healthy workflow is:

```text
profile / benchmark
      ↓
identify hot function
      ↓
compiler diagnostics
      ↓
SSA if needed
```

Opening SSA for arbitrary cold code is rarely useful.

---

## 14. Source Structure and SSA Quality

Code that exposes facts clearly can produce simpler SSA.

Examples:

- explicit length guards;
- simple control flow;
- concrete hot-path types;
- avoiding unnecessary interface boundaries.

But source clarity remains the primary goal.

Do not contort normal application code around speculative SSA behavior.

---

## 15. SSA and Benchmark DCE

Suppose a benchmark repeatedly calls a pure function and ignores the result.

Inlining and SSA can reveal that the operation has no observable effect.

The compiler may eliminate it.

This is one reason benchmark design must ensure that the measured operation remains semantically relevant.

Modern Go benchmark APIs reduce some of this risk, but the underlying principle remains important.

---

## 16. Engineering Perspective

SSA is useful because it exposes a central truth:

> Compiler optimization is largely about data-flow proof.

If the compiler can prove:

- a value is constant;
- a branch is impossible;
- an index is safe;
- a result is unused;

runtime work can disappear.

Understanding SSA therefore helps explain why seemingly small source changes can produce meaningful machine-code changes.
