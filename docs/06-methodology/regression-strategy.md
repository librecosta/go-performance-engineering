# Regression Strategy

## 1. Why Performance Regressions Are Different

Functional regressions often have binary symptoms:

```text
test passes
test fails
```

Performance is noisy.

A benchmark may fluctuate even when code is unchanged.

This means performance regression protection cannot always use a simple exact threshold.

A good strategy chooses the right guard for the magnitude and mechanism of the optimization.

---

## 2. The Goal of a Regression Guard

A regression guard should preserve one of two things:

### Mechanism

Example:

> This parser should remain allocation-free in this path.

### Outcome

Example:

> This endpoint should remain below a CPU/request budget.

Mechanism guards are often stable for local optimizations.

Outcome guards are necessary for system behavior.

---

## 3. Regression Pyramid

Useful layers include:

```text
source/compiler contract
        ↓
microbenchmark
        ↓
component benchmark
        ↓
service/load benchmark
        ↓
production monitoring
```

Not every optimization needs every layer.

Use enough layers to catch likely failure modes.

---

## 4. Functional Tests First

A fast implementation that is incorrect is not a valid baseline.

Performance guards supplement:

- unit tests;
- race tests;
- fuzzing;
- integration tests.

They do not replace them.

---

## 5. Allocation Contracts

Some local behaviors are stable enough for allocation assertions.

Example:

```go
allocs := testing.AllocsPerRun(100, func() {
    parse(input)
})

if allocs != 0 {
    t.Fatalf("unexpected allocations: %v", allocs)
}
```

This can protect a deliberate allocation-free API.

Use only when zero/specific allocation count is actually part of the intended contract.

---

## 6. Allocation Counts Can Be Version-Sensitive

Compiler/runtime upgrades can change allocation behavior.

A strict allocation assertion may fail after a Go upgrade for legitimate reasons.

That is not necessarily bad.

It forces revalidation.

But do not add allocation-count tests to every function.

---

## 7. Compiler Diagnostic Guard

Some projects may protect critical compiler-sensitive code by periodically checking:

- escape diagnostics;
- BCE diagnostics;
- generated assembly.

This is appropriate for specialized libraries.

It is usually too brittle for ordinary application code.

---

## 8. Proof as Regression Reference

A `proofs/` experiment can act as a living reference.

Example:

```text
proofs/compiler/bounds-check-elimination
```

When Go upgrades, rerun the proof.

If compiler behavior changes, update or remove the source workaround.

This is better than assuming the old optimization remains necessary forever.

---

## 9. Microbenchmark Guard

A stable microbenchmark preserves local behavior over time.

Examples:

- parser throughput;
- allocation count;
- atomic scaling;
- copy cost.

Keep the benchmark in the repository even if CI does not hard-fail on every run.

The benchmark itself is an executable historical contract.

---

## 10. Why CI Performance Is Hard

Shared CI machines vary because of:

- noisy neighbors;
- CPU model changes;
- frequency scaling;
- virtualization;
- background jobs.

A strict:

```text
fail if ns/op > baseline × 1.02
```

can create false alarms.

---

## 11. Large Regressions vs Small Regressions

CI is better at catching:

```text
+50%
```

than:

```text
+2%
```

For small changes, use:

- repeated controlled benchmarks;
- dedicated performance workers;
- trend analysis;
- manual benchstat review.

Do not pretend noisy infrastructure is precise.

---

## 12. Dedicated Benchmark Hardware

For high-value libraries/services, dedicated benchmark hosts improve repeatability.

Important controls include:

- fixed CPU model;
- stable OS;
- quiet workload;
- fixed Go version;
- recorded configuration.

This is justified when small regressions have meaningful business impact.

---

## 13. Benchmark Baselines

A baseline can be:

- previous commit;
- main branch;
- release tag;
- rolling median.

Each answers a different question.

### Main vs PR

> Did this PR regress?

### Release vs current

> Has performance drifted over months?

### Rolling trend

> Is gradual regression accumulating?

---

## 14. Repeated Samples

Use repeated benchmark runs.

Store raw output.

Use statistical comparison tools such as benchstat.

A performance CI system should avoid comparing one sample to one sample.

---

## 15. Statistical Alert, Not Automatic Failure

For noisy metrics, a useful model is:

```text
benchmark detects suspicious regression
↓
flag for review
↓
rerun on controlled machine
```

Not every statistically significant movement should block a merge automatically.

---

## 16. Engineering Threshold

Define what magnitude matters.

Example:

```text
<2%
ignore unless repeated

2–5%
investigate

>5%
block for this hotspot
```

Thresholds depend on benchmark variance and system value.

Do not copy arbitrary percentages across projects.

---

## 17. Allocation Thresholds

Allocation metrics are often less noisy than ns/op.

A regression:

```text
0 alloc/op
→ 1 alloc/op
```

can be a strong deterministic signal.

This makes allocation contracts attractive for compiler/API-sensitive hot paths.

---

## 18. Binary Size

Inlining/PGO/duplication optimizations can increase binary size.

If binary size matters:

- deployment;
- cold start;
- instruction cache;

track it as a guardrail.

---

## 19. Memory Regression

A change that improves CPU can increase memory.

Track:

- live heap;
- RSS;
- allocation rate;
- retained capacity.

Memory regressions often require longer-running benchmarks than microbenchmarks.

---

## 20. Retention Tests

Retention is not well represented by `allocs/op`.

A dedicated scenario can:

```text
process large outlier
↓
return to steady state
↓
measure live heap/RSS
```

This can catch pool/cache capacity poisoning.

---

## 21. Concurrency Scaling Regression

For synchronization optimizations, protect the scaling curve.

Example benchmark matrix:

```text
GOMAXPROCS 1
2
4
8
16
```

A refactor may leave single-thread performance unchanged while destroying scalability.

---

## 22. False-Sharing Regression

Padding/layout optimizations can disappear during innocent struct refactoring.

A scaling benchmark under independent writers can detect the regression.

Add an explanatory comment so the benchmark failure is understandable.

---

## 23. P99 Regression

Throughput alone can hide latency regressions.

For user-facing services, load tests should track:

- P50;
- P95;
- P99;
- errors;
- CPU.

Batching and queue changes especially need tail-latency guardrails.

---

## 24. Service Benchmark

A service regression suite may run:

```text
fixed request mix
fixed concurrency
fixed duration
```

and compare:

```text
RPS
CPU/request
P99
alloc rate
RSS
```

This is more expensive but validates system interaction.

---

## 25. Production Monitoring

Some regressions appear only with real traffic.

Maintain production dashboards for key efficiency metrics.

Examples:

```text
CPU / 1k requests
memory / active connection
GC CPU
P99 by endpoint
```

Normalized metrics are often more useful than raw host CPU.

---

## 26. Canary Deployment

A canary creates simultaneous control and candidate cohorts.

This helps reduce confounding from time-varying traffic.

Compare:

```text
same time
similar traffic
different binary
```

Canaries are especially valuable for changes whose local benchmark effect is small but deployment scale is large.

---

## 27. Traffic Normalization

Raw CPU can rise because traffic rises.

Use normalized metrics such as:

```text
CPU/request
bytes allocated/request
RSS/connection
```

where appropriate.

---

## 28. Workload Mix

Even normalized metrics can shift when endpoint/data mix changes.

Segment important measurements by:

- endpoint;
- workload class;
- request size;
- region.

Avoid high-cardinality observability that becomes expensive itself.

---

## 29. PGO Regression

PGO profiles can become stale.

Regression strategy should track:

- profile age;
- source commit/build context;
- representative workload;
- PGO vs non-PGO benchmark occasionally.

A stale profile may silently lose benefit.

---

## 30. Go Version Upgrades

Toolchain upgrades are natural revalidation points.

Rerun important:

- compiler proofs;
- benchmarks;
- PGO comparison;
- unsafe/runtime-specific tests.

An upgrade may make old workarounds unnecessary.

---

## 31. Version Baselines

Do not compare:

```text
Go 1.26 old implementation
```

against:

```text
Go 1.27 new implementation
```

and attribute all difference to source change.

Separate:

```text
toolchain effect
source effect
```

---

## 32. Hardware Upgrades

A new CPU can change:

- cache size;
- branch predictor;
- atomic scaling;
- memory bandwidth.

Hardware-sensitive proofs should be revalidated.

---

## 33. Dependency Upgrades

A dependency can change performance even when application source is untouched.

Examples:

- JSON implementation;
- database driver;
- compression library.

Release performance tests should capture dependency-driven drift.

---

## 34. Benchmark Drift

Benchmark code itself can become unrepresentative as production changes.

A regression suite should be reviewed periodically.

Ask:

- Does payload distribution still match?
- Is concurrency realistic?
- Are removed features still represented?
- Are new hot paths missing?

A stale benchmark can provide false confidence.

---

## 35. Golden Performance Numbers

Avoid treating one absolute value as eternal:

```text
BenchmarkX must be < 42 ns/op forever
```

Different CPUs and Go versions make this brittle.

Prefer relative comparisons within controlled environment where possible.

---

## 36. Relative Performance Contracts

Some specialized libraries may intentionally protect relative ordering:

```text
candidate optimized path
must remain faster than baseline safe path
```

This can be more portable than a fixed ns/op threshold.

It still needs noise-aware comparison.

---

## 37. Perf Budget

A project can define performance budgets.

Examples:

```text
parser allocations ≤ 1/op

steady RSS ≤ 2 GiB under workload W

P99 ≤ 50 ms at 5k RPS

CPU/request ≤ baseline × 1.05
```

Budgets turn performance into an explicit non-functional requirement.

---

## 38. Budget Ownership

Every budget should have:

- workload definition;
- measurement method;
- responsible component/team;
- escalation policy.

An unlabeled threshold quickly becomes meaningless.

---

## 39. Regression Triage

When a regression appears:

```text
1. reproduce
2. confirm environment
3. identify first bad change
4. profile
5. build cost-model hypothesis
6. fix or accept with documented reason
```

Do not immediately tweak benchmark thresholds upward.

---

## 40. Bisecting

`git bisect` combined with a stable benchmark can locate performance regressions.

Automated bisect is powerful when:

- benchmark is fast;
- signal is large;
- environment stable.

For noisy small regressions, manual validation may still be required.

---

## 41. Accepted Regression

Sometimes performance intentionally regresses for:

- correctness;
- security;
- better semantics;
- maintainability;
- new functionality.

If accepted, update the baseline and document why.

Performance budgets are engineering constraints, not dogma.

---

## 42. Regression Comments

If a source shape must not change for performance, comment it.

The regression guard should not be a mysterious CI failure.

Example:

```go
// Keep this padding; BenchmarkShardWrites detects false sharing if the
// independently written counters share a cache line.
```

This connects code and evidence.

---

## 43. Unsafe Regression Strategy

Unsafe optimizations need both:

```text
performance regression guard
+
correctness regression guard
```

Examples:

- benchmark;
- race test;
- checkptr;
- lifetime tests;
- architecture CI.

Speed alone is not enough.

---

## 44. Lock-Free Regression Strategy

Protect:

- functional invariants;
- stress behavior;
- race correctness;
- ABA scenarios;
- scaling;
- retry rate.

A throughput benchmark alone may miss correctness degradation.

---

## 45. mmap Regression Strategy

Test:

- unmap lifecycle;
- concurrent readers;
- file truncation/error handling where relevant;
- cold/warm performance;
- RSS/page behavior.

Platform-specific CI may be necessary.

---

## 46. cgo Regression Strategy

Record:

- Go version;
- C compiler;
- native library version.

Test:

- pointer lifetime;
- callbacks;
- pinning/handles;
- boundary benchmark;
- cross-platform ABI where supported.

---

## 47. Runtime-Private Dependencies

If the project deliberately uses runtime-private symbols, CI should include:

- every supported Go version;
- upcoming Go release/RC where possible;
- build checks;
- behavioral tests.

This is part of the cost of choosing the private dependency.

---

## 48. Alert Fatigue

A performance CI system that constantly produces false alarms will be ignored.

Tune regression detection for useful signal.

Better:

```text
few high-confidence alerts
```

than:

```text
daily noisy 1% warnings
```

---

## 49. Historical Data

Store benchmark trends for important hot paths.

Long-term data can reveal:

- gradual drift;
- sudden toolchain effects;
- dependency regressions;
- improvements that later disappear.

One PR comparison cannot show history.

---

## 50. Reproducibility Metadata

For benchmark records, store:

```text
commit
Go version
CPU
OS
GOMAXPROCS
command
PGO state
relevant env
```

Without provenance, old numbers lose value.

---

## 51. Stop Conditions for Guards

Do not keep obsolete guards forever.

Remove or revise a guard when:

- the hot path disappears;
- compiler makes workaround unnecessary;
- workload changes;
- performance objective changes.

Regression infrastructure also needs maintenance.

---

## 52. Suggested Strategy by Optimization Type

### Allocation optimization

- microbenchmark;
- allocs/op guard;
- component GC validation.

### Synchronization optimization

- scaling benchmark;
- mutex/block profile;
- service P99 validation.

### Compiler workaround

- compiler proof;
- microbenchmark;
- toolchain revalidation.

### Zero-copy/unsafe

- benchmark;
- retention check;
- race/checkptr/correctness tests.

### GC tuning

- representative load test;
- CPU/RSS/P99;
- production monitoring.

---

## 53. Regression Strategy Template

### Property

What must remain true?

### Workload

Under what input/concurrency?

### Metric

What is measured?

### Baseline

What is the comparison target?

### Noise Model

How variable is the metric?

### Trigger

What change should alert/fail?

### Revalidation

When should the guard be reconsidered?

---

## 54. Related Sources

- `testing`: https://pkg.go.dev/testing
- `benchstat`: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat
- Go diagnostics: https://go.dev/doc/diagnostics

---

## 55. Engineering Perspective

The final step of optimization is not:

> The benchmark is faster.

It is:

> The gain is protected by a guard appropriate to its mechanism and noise level, and future maintainers know why that guard exists.

That is how an optimization becomes durable engineering.
