# Optimization Review

## 1. Why Performance Changes Need a Different Review Lens

A normal code review asks:

- Is it correct?
- Is it readable?
- Is it maintainable?
- Is the API appropriate?

A performance optimization adds new questions:

- Is there a measured problem?
- Does the proposed mechanism target that problem?
- Is the benefit large enough?
- Does the change move cost elsewhere?
- Can future maintainers recover the optimization intent?

A change can be faster and still be the wrong change.

---

## 2. Review the Problem Before the Solution

A performance PR should begin with the problem statement.

Weak:

> Replace Mutex with atomics for speed.

Strong:

> Under 32-way load, this global metrics lock contributes 19% of blocked time and throughput stops scaling above 8 workers.

The second statement gives reviewers something testable.

---

## 3. Define the Objective

Specify the primary objective:

```text
CPU/request
throughput
P99 latency
RSS
allocation rate
GC CPU
```

Then specify guardrails.

Example:

```text
Goal:
- CPU/request -5% or better

Guardrails:
- P99 must not regress >2%
- RSS must not increase >5%
- no unsafe/runtime-private dependency
```

This prevents a change from optimizing one metric at any cost.

---

## 4. Baseline

A review needs a baseline.

Possible baselines:

- current main;
- current production;
- previous implementation;
- idiomatic safe version.

Record exact benchmark/profile conditions.

Without a baseline, "faster" is undefined.

---

## 5. Bottleneck Evidence

The change should target a measured bottleneck.

Evidence may include:

- CPU profile;
- heap/alloc profile;
- mutex profile;
- trace;
- runtime metrics;
- service benchmark.

A source-level smell is insufficient for high-complexity changes.

---

## 6. Cost Model

The author should explain the expected mechanism.

Example:

```text
one shared atomic counter
↓
cache-line ownership transfers
↓
poor multicore scaling

candidate:
per-shard counters
↓
independent write lines
↓
aggregation on read
```

This lets reviewers challenge the mechanism rather than debate style.

---

## 7. Candidate Simplicity

Before approving a complex technique, ask whether a simpler change addresses the same cost.

Examples:

```text
custom lock-free queue
vs
larger batch + channel

unsafe zero-copy
vs
append-style API

runtime procPin
vs
explicit sharding

sync.Pool
vs
stack/local buffer
```

Prefer the lowest-risk mechanism that achieves the objective.

---

## 8. Remove Work First

Review whether the operation can disappear.

Before:

```text
make atomic increment faster
```

ask:

```text
do we need one global increment per event?
```

Before:

```text
pool temporary object
```

ask:

```text
can it stay on stack?
```

Architecture can outperform micro-tuning.

---

## 9. A/B Evidence

A performance PR should normally provide representative before/after measurement.

For benchmarkable local changes:

```text
old
new
benchstat
```

For large service changes:

```text
load test / canary
```

One number without repeated samples should not support a subtle optimization.

---

## 10. Statistical vs Practical Gain

A statistically significant 0.3% microbenchmark win may not justify:

- unsafe;
- custom assembly;
- additional abstraction.

Review engineering significance separately.

---

## 11. Amdahl Bound

Use profile hotness to estimate upper-bound value.

Example:

```text
target = 4% total CPU
candidate local speedup = 25%

approx system upper bound ≈ 1%
```

If implementation complexity is high, stop early.

---

## 12. Correctness First

Performance optimizations cannot weaken semantic correctness unless the API explicitly changes semantics.

Particularly dangerous areas:

- lock-free algorithms;
- unsafe aliasing;
- cgo lifetime;
- mmap lifetime;
- runtime-private tricks.

A benchmark cannot prove correctness.

---

## 13. Maintainability Is a Hard Gate

An optimization that cannot be safely maintained is unfinished.

This project treats maintainability as a hard acceptance condition, not a soft preference.

---

## 14. Which Optimizations Need Comments?

Non-obvious performance code should normally have adjacent explanation.

Examples:

- `_ = b[7]` BCE proofs;
- cache-line padding;
- unusual field ordering;
- intentional copy to avoid retention;
- pool capacity cutoffs;
- pointer→index representation;
- AoS→SoA;
- hot/cold split;
- unusual sharding/batching;
- CAS/backoff;
- lock-free algorithms;
- unsafe zero-copy;
- mmap lifetime;
- `runtime.KeepAlive`;
- cgo directives;
- assembly;
- runtime-private dependencies.

---

## 15. Which Optimizations Usually Do Not Need Comments?

Idiomatic code whose intent is already clear:

```go
items := make([]Item, 0, len(src))
```

does not need:

```go
// Preallocate for performance.
```

Redundant comments create noise.

Document the unusual invariant, not every efficient line.

---

## 16. Performance Comment Structure

A useful comment may answer four questions.

### WHY

Why is this unusual code intentional?

### MECHANISM

What measured cost does it reduce?

### PRESERVATION

What tempting simplification would reintroduce the cost?

### SAFETY CONTRACT

For unsafe/lifetime code, what makes it correct?

Not every comment needs all four, but the relevant ones should be recoverable.

---

## 17. BCE Example

Good:

```go
// Keep this bounds check: it proves len(b) >= 8 to the compiler,
// allowing the fixed indexed reads below to eliminate redundant checks.
_ = b[7]
```

Bad:

```go
// Optimization.
_ = b[7]
```

The first comment preserves the reason.

---

## 18. Retention Example

```go
// Copy the small result instead of retaining the full input buffer.
// Returning input[:n] here may keep a multi-megabyte backing array alive.
out := append([]byte(nil), input[:n]...)
```

The code looks slower locally.

The comment explains the system-level goal.

---

## 19. False-Sharing Example

```go
type shard struct {
    counter atomic.Uint64

    // Keep independently written shard counters on separate cache lines.
    // Removing this padding can reintroduce false sharing under contention.
    _ cpu.CacheLinePad
}
```

The comment protects a physical-layout invariant invisible in ordinary logic.

---

## 20. Unsafe Example

```go
// bytesToString returns a zero-copy view of b.
// The backing bytes must remain immutable and must not be reused while
// the returned string is reachable.
func bytesToString(b []byte) string {
    if len(b) == 0 {
        return ""
    }
    return unsafe.String(unsafe.SliceData(b), len(b))
}
```

Unsafe comments must describe correctness, not merely speed.

---

## 21. Benchmark as Documentation

A benchmark can preserve why an optimization exists.

Example:

```text
BenchmarkFalseSharing
BenchmarkShardedCounter
BenchmarkRetentionCopy
```

Future maintainers can verify the mechanism after compiler/runtime upgrades.

---

## 22. Proof Link

For particularly non-obvious optimizations, comments or design docs can link to a proof directory.

Example:

```text
proofs/compiler/bounds-check-elimination/
```

This separates detailed evidence from source comments.

---

## 23. Unsafe Escalation Model

Risk should escalate deliberately:

```text
safe Go
↓
public unsafe API
↓
OS / FFI boundary
↓
compiler directives / assembly
↓
runtime-private dependency
```

Do not skip directly to the lowest layer.

---

## 24. Safe Go First

Before unsafe zero-copy, ask:

- can caller provide destination?
- can data be batched?
- can copy be removed by architecture?
- is the copy actually hot?

Safe redesign often captures most of the benefit.

---

## 25. Public Unsafe API

Public `unsafe` operations have documented contracts.

They are still high risk, but preferable to private runtime hacks when the mechanism genuinely requires aliasing or raw memory.

---

## 26. OS / FFI Boundary

mmap and cgo introduce external lifetime/ownership.

Review must include:

- who allocates;
- who releases;
- whether pointers escape;
- whether views outlive storage.

---

## 27. Compiler Contracts

Directives such as noescape assert facts to the compiler.

A false assertion can break memory safety.

Require:

- implementation proof;
- tests;
- strong comments;
- toolchain compatibility plan.

---

## 28. Runtime-Private Red Zone

`//go:linkname` into private runtime symbols or scheduler internals should be exceptional.

Review questions:

- Is there a public alternative?
- What exact measured gain requires this?
- What Go versions are supported?
- How will upgrades detect breakage?
- Is there a fallback?

---

## 29. Lock-Free Review

Custom lock-free structures need a higher correctness bar.

Review:

- invariant;
- linearization point;
- progress guarantee;
- ABA;
- memory lifetime;
- retry behavior;
- contention scaling;
- race/stress testing.

"Uses atomics" is not a correctness argument.

---

## 30. Atomic Review

Ask:

- Is state truly single-word?
- Are compound invariants being split?
- What is write frequency?
- Is there true sharing?
- Would sharding/ownership be simpler?

Atomics are not a replacement for state design.

---

## 31. Mutex Review

Do not reject a mutex because it exists.

Ask:

- Is it contended?
- How long is the critical section?
- Can work move outside?
- Can state be partitioned?

A short uncontended mutex is often excellent engineering.

---

## 32. RWMutex Review

Require workload evidence for `RWMutex` in hot code.

Measure:

- reader duration;
- writer frequency;
- core count;
- Mutex baseline.

"Mostly reads" is not enough.

---

## 33. Pool Review

For `sync.Pool`, ask:

- Is allocation churn measured?
- Is reset complete?
- Can large capacities poison the pool?
- Are retained pointers cleared?
- Is the object truly temporary?
- Does pooling improve RSS as well as allocs/op?

---

## 34. Data Layout Review

For layout transformations, ask:

- what fields are hot?
- what is access pattern?
- object count?
- cache/TLB impact?
- GC pointer density?
- readability cost?

Do not reorder one rarely allocated config struct for theoretical packing.

---

## 35. PGO Review

If a change relies on PGO behavior:

- identify profile origin;
- ensure workload representativeness;
- compare PGO on/off where relevant;
- preserve build reproducibility.

Do not manually specialize source if the compiler already does so unless evidence supports it.

---

## 36. Compiler-Sensitive Review

For escape/BCE/inlining changes:

- record Go version;
- include diagnostics;
- benchmark;
- avoid hard-coding compiler heuristics.

A workaround should be removable when the compiler improves.

---

## 37. Version Sensitivity

Mark optimizations that depend on:

- current allocator thresholds;
- current GC implementation;
- current inlining heuristic;
- current runtime private layout;
- architecture-specific instructions.

These need explicit revalidation triggers.

---

## 38. Portability

A 10% amd64 improvement may not exist on arm64.

If the package supports multiple architectures, review:

- generated code;
- benchmark coverage;
- correctness;
- fallback.

---

## 39. Memory Guardrails

CPU optimizations can spend memory.

Review:

```text
CPU ↓
RSS ?
live heap ?
retention ?
```

A cache or pool optimization without memory guardrails is incomplete.

---

## 40. Latency Guardrails

Throughput optimization can harm P99.

Examples:

- batching;
- long queue buffers;
- aggressive spin;
- large GC targets.

Measure latency distribution when user-facing latency matters.

---

## 41. CPU Guardrails

A lock-free algorithm may increase throughput while burning more CPU.

If infrastructure efficiency matters, include:

```text
CPU/request
```

or equivalent.

---

## 42. Complexity Budget

Every optimization spends a complexity budget.

A useful rule:

```text
complexity must be proportional to
measured system value
```

Do not turn an ordinary service into a runtime research project for a tiny gain.

---

## 43. Reversibility

Prefer reversible changes when evidence is uncertain.

Examples:

- feature flag;
- build option;
- isolated implementation;
- fallback path.

This is especially useful for PGO/configuration-level changes.

---

## 44. Failure Mode

Review what happens if the optimization assumption stops being true.

Examples:

```text
pool receives 100 MiB buffer
shard count too low
input distribution changes
PGO profile goes stale
mmap closes early
```

Optimization robustness matters.

---

## 45. Observability

For system-level changes, ensure the relevant metrics remain observable after deployment.

Example:

```text
change GC tuning
→ keep GC CPU/live heap/RSS dashboards
```

An optimization that removes the ability to detect regressions is risky.

---

## 46. Regression Guard

Before merging, decide how the gain will be preserved.

Options:

- benchmark;
- allocation assertion;
- system load test;
- CI comparison;
- canary metric;
- proof.

See:

- [Regression Strategy](./regression-strategy.md)

---

## 47. Stop Condition

A review can legitimately conclude:

> Do not optimize.

Reasons:

- bottleneck too small;
- gain below noise;
- complexity too high;
- system result neutral;
- guardrail regression;
- evidence weak.

Rejecting an optimization is a successful outcome.

---

## 48. Suggested Review Narrative

A performance change should be explainable as:

```text
Problem
↓
Baseline evidence
↓
Cost model
↓
Candidate
↓
A/B result
↓
Trade-offs
↓
Maintainability
↓
Regression guard
```

If this narrative is coherent, the code review becomes much easier.

---

## 49. Human Review Template

### Goal

What system metric should improve?

### Baseline

What measurements show the current behavior?

### Cost Model

Why does the current implementation spend this resource?

### Candidate Change

What mechanism changes?

### Evidence

What A/B results support the change?

### Guardrails

What must not regress?

### Maintainability

What non-obvious invariant must be preserved?

### Version Sensitivity

Does this depend on compiler/runtime/hardware behavior?

### Regression Strategy

How will future changes detect loss of the gain?

---

## 50. Decision Categories

A review can classify an optimization as:

```text
ACCEPT
benefit and risk are justified

ACCEPT WITH GUARD
valid but requires benchmark/comment/version test

EXPERIMENTAL
promising but evidence not sufficient for default

REJECT
cost/complexity exceeds value

NO OPTIMIZATION NEEDED
hypothesis disproven
```

This is more informative than "fast" vs "slow".

---

## 51. Engineering Perspective

The purpose of optimization review is not to make performance code harder to merge.

It is to ensure that speed improvements remain:

- correct;
- measurable;
- understandable;
- maintainable;
- reproducible.

A performance optimization is finished only when the next engineer can safely preserve or remove it for the right reason.
