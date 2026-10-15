# Inlining

## 1. What Inlining Does

Inlining replaces a function call with the callee body at the call site.

Source:

```go
func add(a, b int) int {
    return a + b
}

func work() int {
    return add(1, 2)
}
```

Conceptually after inlining:

```go
func work() int {
    return 1 + 2
}
```

The obvious benefit is eliminating call overhead.

But that is usually not the most important benefit.

---

## 2. Inlining as an Enabling Optimization

Inlining exposes callee code to caller context.

After inlining, the compiler may discover:

- constant arguments;
- caller-side length guarantees;
- concrete receiver types;
- shorter lifetimes;
- dead branches.

This can enable:

- escape improvements;
- BCE;
- constant propagation;
- dead-code elimination.

Therefore:

> The main value of inlining is often the optimization it unlocks.

---

## 3. Constant Propagation Example

```go
func enabled(flags uint32) bool {
    return flags&1 != 0
}

func process() {
    const flags = 0

    if enabled(flags) {
        expensive()
    }
}
```

After inlining:

```text
flags&1 == 0
```

The branch is known false.

The compiler can remove:

```text
expensive()
```

The gain is much larger than one removed call.

---

## 4. Inlining and Escape Analysis

A helper may return a pointer:

```go
func makePoint() *Point {
    return &Point{X: 1}
}
```

At the helper boundary, the pointer appears to escape.

If inlined into a caller that immediately reads the value and discards it, the compiler may prove the storage can remain local.

This is a key interaction between inlining and escape analysis.

---

## 5. Inlining and Bounds Checks

Example:

```go
func byte7(b []byte) byte {
    return b[7]
}

func parse(b []byte) byte {
    if len(b) < 8 {
        return 0
    }

    return byte7(b)
}
```

Without enough context, the helper must preserve safe semantics.

After inlining, the caller's dominating length check becomes visible around `b[7]`.

This can enable BCE.

---

## 6. Inlining Heuristics

Inlining increases code size.

The compiler therefore uses heuristics rather than inlining every function.

Considerations include:

- function complexity;
- call-site context;
- profile information;
- expected benefit;
- code growth.

These heuristics evolve across Go releases.

Do not write application code around a hard-coded assumption such as:

> Functions below exactly N nodes always inline.

That is an implementation detail.

---

## 7. Diagnostics

Compiler diagnostics can report whether a function is considered inlineable.

Typical command:

```bash
go build -gcflags='-m=2' ./...
```

Useful messages include:

```text
can inline
cannot inline
```

The reason may reveal excessive complexity or other constraints.

---

## 8. Inlining Can Increase Code Size

Inlining duplicates callee code at call sites.

Excessive code growth can hurt:

- instruction-cache locality;
- binary size;
- register pressure.

Therefore more inlining is not automatically always better.

The compiler's job is to balance those trade-offs.

---

## 9. Hot Helpers

Small helpers in a hot path can often preserve clean source structure without paying runtime call overhead.

This is valuable because developers do not need to manually flatten all code for performance.

Good abstraction and good optimization are often compatible.

---

## 10. Large Functions

A giant function is not automatically faster than several helpers.

Possible problems include:

- larger instruction footprint;
- harder compiler heuristics;
- more register pressure;
- less maintainability.

Performance engineering should not turn into manual source-code flattening.

---

## 11. Interfaces and Inlining

An indirect interface call cannot be inlined in the same way as a known direct call.

Devirtualization can restore a direct call.

Then inlining can proceed.

This creates the chain:

```text
interface call
   ↓
devirtualization
   ↓
direct call
   ↓
inlining
```

---

## 12. PGO and Inlining

Profile-guided optimization tells the compiler which call sites are hot.

The compiler can spend more optimization budget where production profiles show real execution frequency.

This can improve inlining decisions without manually removing abstractions from the source.

---

## 13. Inlining and Benchmark Interpretation

A microbenchmark may measure very different machine code from the same function used in a larger application.

Why?

Because caller context changes:

- constants;
- inlining;
- escape;
- devirtualization.

Therefore microbenchmarks should reflect realistic call patterns where possible.

---

## 14. Disabling Inlining for Experiments

Temporarily disabling inlining can help determine how much a benchmark depends on it.

This is a diagnostic experiment, not a production configuration.

The comparison can reveal whether an observed gain is really caused by inlining or something else.

---

## 15. Common Misconceptions

### "Inlining just saves a CALL"

Incomplete.

### "Every tiny function will always inline"

Not a language guarantee.

### "Manual function merging is always faster"

False.

### "Interfaces always prevent optimization"

Not necessarily; devirtualization and PGO may resolve them.

---

## 16. Engineering Perspective

Inlining is best understood as **context propagation**.

It lets facts known at the caller flow into the callee body.

That context can remove allocations, checks, branches, and dispatch.

The performance question is therefore not:

> Can I eliminate this function call manually?

It is:

> Does the compiler already have enough context to optimize this hot call path?
