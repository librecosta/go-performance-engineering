# Evidence Model

## 1. Why an Evidence Model Is Needed

Performance claims vary dramatically in strength.

Compare:

```text
"This loop looks allocation-heavy."
```

with:

```text
"Production CPU profiles show this path accounts for 23% of CPU,
compiler diagnostics show one heap escape per request,
and a representative A/B removes the allocation and reduces CPU/request by 8%."
```

Both may be observations about the same code.

They should not have equal decision weight.

An evidence model makes the required proof proportional to the risk and scope of the optimization.

---

## 2. Claims Need Scope

A performance claim should specify what it actually claims.

Weak:

> This is faster.

Better:

> On Go 1.27/amd64, for 4 KiB payloads, implementation B reduces allocations from 3/op to 1/op and improves the microbenchmark median by approximately 12%.

Better still for production decisions:

> The same change reduces CPU/request by 4% in the component workload with no P99 or RSS regression.

Scope prevents local facts from becoming universal myths.

---

## 3. The Evidence Ladder

This handbook uses six evidence levels:

```text
L0 — Source Inspection
L1 — Diagnostics / Profiles
L2 — Reproducible Microbenchmark
L3 — Component / Service Benchmark
L4 — Production / Canary Evidence
L5 — Hardware PMU / Low-Level Evidence
```

The levels are not strictly "higher is always better".

They answer different questions.

A strong investigation often combines several.

---

## 4. L0 — Source Inspection

L0 identifies hypotheses from code.

Examples:

- obvious repeated allocation;
- one global mutex;
- large slice retained by small view;
- interface call in a hot-looking loop;
- `[]*T` pointer-heavy layout.

L0 can answer:

> What might be expensive?

It cannot prove:

> What is expensive?

---

## 5. Appropriate Use of L0

Source inspection is useful for:

- building initial hypotheses;
- spotting correctness/performance smells;
- deciding which tools to use;
- reviewing known hot code.

It is insufficient for optimization approval when the claim is performance-only.

---

## 6. L1 — Compiler/Runtime Diagnostics and Profiles

L1 observes implementation/runtime behavior.

Examples:

- CPU profile;
- heap/allocs profile;
- mutex/block profile;
- execution trace;
- escape diagnostics;
- BCE diagnostics;
- assembly/SSA inspection;
- runtime metrics.

This can answer:

> Does the suspected mechanism actually exist?

---

## 7. L1 Examples

### Escape

```text
compiler: moved to heap
```

confirms an escape decision.

### BCE

Residual check diagnostics confirm that a bounds check remains.

### CPU

pprof confirms a function is a meaningful CPU consumer.

### Contention

mutex profile confirms waiting is occurring around a critical section.

---

## 8. L1 Limit

Diagnostics often prove mechanism but not benefit.

Example:

```text
one bounds check remains
```

does not tell you whether removing it matters.

The check may execute once per hour.

Therefore L1 often leads to L2 or L3.

---

## 9. L2 — Reproducible Microbenchmark

L2 compares candidate implementations in isolation.

It is useful for:

- mechanism validation;
- local cost comparison;
- allocation measurement;
- scaling primitives.

A good L2 artifact includes:

- code;
- workload;
- command;
- environment;
- repeated results.

---

## 10. L2 Strength

Microbenchmarks are strong at causal isolation.

Example:

```text
same parser
same bytes
one version BCE-friendly
one version not
```

If diagnostics confirm mechanism and benchmark shows timing difference, the local claim is strong.

---

## 11. L2 Limit

Microbenchmarks are weak at system importance.

A 30% local speedup may improve the service by 0.02%.

L2 should not automatically authorize high-complexity changes.

---

## 12. L3 — Component / Service Benchmark

L3 evaluates the change in a realistic subsystem or service workload.

Examples:

- HTTP handler benchmark;
- storage-engine workload;
- parser + allocator + downstream processing;
- concurrency load test.

It answers:

> Does the local mechanism matter in realistic context?

---

## 13. L3 Inputs

A strong component benchmark tries to preserve:

- realistic payload sizes;
- key distribution;
- concurrency;
- cache state;
- allocation lifetime;
- relevant dependencies.

Not every external dependency must be included.

The goal is representative cost structure.

---

## 14. L4 — Production / Canary Evidence

L4 validates behavior under real workloads.

Possible signals:

- CPU/request;
- host CPU;
- RSS;
- throughput;
- P50/P99;
- GC CPU;
- error rate;
- infrastructure cost.

This is the strongest evidence for user/system value.

---

## 15. Production Evidence Is Noisy

Production includes uncontrolled factors:

- traffic changes;
- deploy timing;
- dependency variance;
- background jobs;
- regional mix.

Use:

- canaries;
- matched cohorts;
- before/after windows;
- controlled rollout;
- workload normalization.

Do not overinterpret one dashboard screenshot.

---

## 16. L5 — Hardware PMU / Low-Level Evidence

L5 uses hardware-level measurements such as:

- cycles;
- instructions;
- IPC;
- cache misses;
- branch misses;
- bandwidth;
- NUMA events.

This is useful when the claim itself is hardware-mechanism specific.

Examples:

- false sharing;
- cache locality;
- branch misprediction;
- memory-bandwidth saturation.

---

## 17. L5 Is Not Automatically Required

Do not use PMU counters for every optimization.

If:

```text
CPU profile
+
benchmark
+
system validation
```

already answer the question, PMU evidence may add little value.

The evidence cost should match the uncertainty.

---

## 18. Evidence by Risk

A simple safe optimization may require less evidence.

Example:

```go
make([]Item, 0, len(src))
```

in a measured allocation hotspot.

A high-risk optimization needs stronger proof.

Example:

```text
unsafe zero-copy
runtime linkname
custom lock-free queue
```

Higher maintenance/correctness risk raises the required evidence threshold.

---

## 19. Risk-Adjusted Evidence

Conceptually:

```text
required evidence
↑
as
correctness risk
maintenance cost
portability risk
scope
↑
```

This is one of the central governance rules of the project.

---

## 20. Mechanism Evidence vs Outcome Evidence

Two categories should be separated.

### Mechanism

> Did the intended low-level effect occur?

Examples:

- allocation disappeared;
- bounds check disappeared;
- cache misses decreased;
- lock wait decreased.

### Outcome

> Did the system improve?

Examples:

- CPU/request reduced;
- P99 reduced;
- throughput increased;
- RSS stayed within guardrail.

A strong optimization often has both.

---

## 21. Negative Evidence

Evidence can disprove an optimization hypothesis.

Example:

```text
source inspection:
global atomic looks suspicious

scaling benchmark:
no meaningful contention

profile:
atomic <0.2% CPU
```

Conclusion:

> Do not optimize this path.

This is a successful performance investigation.

---

## 22. Absence of Evidence

Not finding a hotspot is not proof that no performance problem exists.

You may be using the wrong tool.

Example:

```text
CPU profile looks normal
P99 terrible
```

Possible next step:

```text
execution trace / block profile
```

Evidence models guide escalation, not premature closure.

---

## 23. Representative Evidence

A benchmark is only evidence for the workload it represents.

Every result should record key dimensions:

- payload;
- concurrency;
- distribution;
- Go version;
- architecture.

Without these, the scope becomes ambiguous.

---

## 24. Version Evidence

Compiler/runtime claims should record Go version.

Examples:

```text
escape decision
BCE
inlining
allocator path
GC behavior
```

These can change across releases.

A proof should be re-runnable.

---

## 25. Hardware Evidence

Claims involving:

- cache lines;
- branch predictor;
- instruction lowering;
- atomic scaling;

should record CPU/architecture.

Do not turn one x86 benchmark into a universal Go rule.

---

## 26. Statistical Evidence

Repeated benchmarks reduce noise.

A/B comparison should use enough samples to characterize variance.

Statistical significance is useful, but not sufficient.

---

## 27. Engineering Significance

A performance change must matter enough.

Conceptually:

```text
measured gain
×
hotness
×
deployment scale
```

should justify:

```text
complexity
risk
maintenance
```

This is the optimization value equation.

---

## 28. Confidence Language

Use wording that matches evidence.

Examples:

### L0 only

> This pattern may cause heap allocation.

### L1 diagnostics

> The Go 1.27 compiler moves this value to the heap in this build.

### L2 benchmark

> Under this benchmark, removing the escape reduces one allocation/op and improves median runtime.

### L4

> The canary shows lower CPU/request under comparable traffic with no P99 regression.

Precise language prevents overclaiming.

---

## 29. Evidence Provenance

For important claims, preserve:

- source commit;
- benchmark code;
- Go version;
- command line;
- hardware;
- result files;
- profile timestamp/context.

Future maintainers should be able to reconstruct the reasoning.

---

## 30. Proof Structure

A proof in this repository should generally contain:

```text
Claim
Cost Model
Baseline
Experiment
Verification
Benchmark
Environment
Caveats
Recommendation
```

This keeps proof artifacts focused.

---

## 31. Claim

State one narrow mechanism.

Example:

> A dominating length check allows the Go compiler to remove redundant bounds checks from fixed-index reads.

Avoid:

> This makes parsers much faster.

The second is too broad.

---

## 32. Cost Model

Explain why the mechanism could affect performance.

Example:

```text
redundant bounds checks
→ additional compare/branch/panic-edge machinery
```

This connects evidence to theory.

---

## 33. Baseline

Define the unoptimized/reference implementation.

The baseline should be reasonable, not deliberately pathological.

---

## 34. Experiment

Change only what is necessary to test the claim.

For mechanism proofs, minimize confounders.

---

## 35. Verification

Use direct evidence that the mechanism changed.

Examples:

- compiler output;
- assembly;
- profile;
- PMU counter.

Benchmark time alone is often indirect.

---

## 36. Benchmark

Measure local cost.

Repeat enough times.

Include allocation metrics when relevant.

---

## 37. Environment

Record:

```text
Go version
GOOS
GOARCH
CPU
GOMAXPROCS
commands
```

Version-sensitive proofs are incomplete without this.

---

## 38. Caveats

State where the result does not apply.

Examples:

- compiler-version sensitive;
- low-contention only;
- amd64 only;
- small payload only;
- memory retention excluded.

---

## 39. Recommendation

Recommendation levels can distinguish:

```text
safe/default
conditional
advanced
diagnostic only
avoid/private runtime
```

A proof demonstrating a mechanism does not imply automatic recommendation.

---

## 40. Evidence Matrix Example

| Claim | Minimum Useful Evidence | Stronger Evidence |
|---|---|---|
| Heap escape exists | compiler `-m` | alloc benchmark |
| Bounds check remains | BCE diagnostic | assembly |
| Mutex is contended | mutex profile | scaling benchmark |
| False sharing exists | scaling anomaly | PMU/cache evidence |
| Zero-copy helps | allocation/copy benchmark | system CPU + retention |
| PGO helps | A/B build benchmark | canary |

---

## 41. Causality

Performance engineering benefits from causal chains.

Weak:

```text
changed code
benchmark faster
```

Stronger:

```text
changed layout
↓
cache misses decrease
↓
cycles/op decrease
↓
throughput increases
```

Not every optimization requires every link, but mechanism clarity increases confidence.

---

## 42. Confounders

Common confounders include:

- Go version changes;
- CPU frequency;
- workload distribution;
- different data sets;
- GC settings;
- PGO profile changes;
- dependency versions.

A/B experiments should control these.

---

## 43. Correlation Trap

Example:

```text
after deploying pool:
GC CPU decreased
```

But traffic also dropped 20%.

Without normalization/control, causality is uncertain.

This is why canaries and matched workloads are valuable.

---

## 44. Amdahl's Law as Evidence Filter

If profiling shows the target path is 1% of total CPU, a complex optimization promising a 2× local speedup has a hard upper bound around 0.5% total CPU improvement.

This can stop low-value work before implementation.

---

## 45. Evidence Budget

Evidence itself costs engineering time.

Use an evidence budget.

For a tiny reversible safe change:

```text
profile + benchmark
```

may be enough.

For a runtime-private hack:

```text
profile
benchmark
system test
correctness stress
version CI
documented fallback
```

may be required.

---

## 46. Reproducibility

A result that cannot be reproduced becomes folklore.

The goal is not perfect scientific reproducibility across every machine.

The goal is enough information for another engineer to run the same experiment and understand why results may differ.

---

## 47. Evidence Decay

Evidence can age.

Causes:

- compiler improvements;
- runtime redesign;
- hardware changes;
- workload drift.

A proof should therefore be treated as:

```text
reproducible evidence
```

not:

```text
permanent law
```

---

## 48. Revalidation Triggers

Revalidate when:

- major Go version changes;
- architecture changes;
- workload changes substantially;
- optimization implementation changes;
- benchmark results become suspicious.

---

## 49. Evidence and Maintainability

Documentation is evidence preservation.

A comment such as:

```go
// Keep this bounds proof...
```

should ideally point to a benchmark/proof when the optimization is non-obvious.

This prevents future code cleanup from accidentally discarding the mechanism.

---

## 50. Evidence and Review

A reviewer should be able to answer:

- What problem is measured?
- What mechanism is proposed?
- What evidence confirms it?
- What trade-off is introduced?
- How will regression be detected?

If not, the optimization is not review-ready.

---

## 51. Related Sources

- Go diagnostics: https://go.dev/doc/diagnostics
- `testing`: https://pkg.go.dev/testing
- `runtime/pprof`: https://pkg.go.dev/runtime/pprof
- `benchstat`: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat

---

## 52. Engineering Perspective

The purpose of an evidence model is not bureaucracy.

It is to make optimization confidence proportional to optimization risk.

Performance engineering becomes reliable when every important recommendation can answer:

```text
What do we know?
How do we know it?
Under what conditions is it true?
```
