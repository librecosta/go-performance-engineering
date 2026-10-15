# Benchmarking

## 1. What a Benchmark Proves

A benchmark compares implementations under a defined workload.

It answers:

> Under these conditions, which candidate performs better on these metrics?

A benchmark does not prove:

- universal superiority;
- production impact;
- the root cause of a hotspot.

The strength of a benchmark depends on how closely its workload represents the engineering question.

---

## 2. Benchmark vs Profile

Use:

```text
profile
→ choose what to investigate

benchmark
→ compare proposed implementations
```

Optimizing a benchmark before proving the code is hot is benchmark-driven development, not performance engineering.

---

## 3. Define the Metric First

A benchmark can measure:

- time/op;
- bytes/op;
- allocations/op;
- throughput;
- custom metrics.

Choose metrics that match the claim.

Examples:

```text
"reduces allocation churn"
→ B/op + allocs/op

"scales better under contention"
→ throughput across concurrency levels

"reduces parser CPU"
→ ns/op plus component CPU validation
```

---

## 4. `testing.B`

Go's `testing` package provides the standard benchmark harness.

Modern benchmarks should generally prefer `B.Loop`.

Example:

```go
func BenchmarkEncode(b *testing.B) {
    input := makeInput()

    for b.Loop() {
        Encode(input)
    }
}
```

As of current Go, `B.Loop` is the preferred benchmark style.

---

## 5. `B.Loop`

`B.Loop` was introduced in Go 1.24.

It improves benchmark ergonomics in several ways.

Setup before the loop is not included in the timed region.

Cleanup after the loop is not included either.

The benchmark function is also run once per measurement rather than repeatedly re-entering the whole benchmark function as with traditional `b.N` calibration.

---

## 6. B.Loop and Dead-Code Elimination

Current testing documentation notes that values used syntactically inside the `for b.Loop()` body are kept alive in a way that helps prevent the compiler from eliminating the measured body completely.

This reduces one class of benchmark mistakes.

It does not make every benchmark realistic.

---

## 7. Exact Loop Form Matters

The special benchmark behavior applies to the syntactic form:

```go
for b.Loop() {
    ...
}
```

Do not wrap the loop condition in unusual abstractions and expect identical compiler treatment.

---

## 8. Legacy `b.N`

Older benchmarks use:

```go
func BenchmarkEncode(b *testing.B) {
    input := makeInput()

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        Encode(input)
    }
}
```

This remains supported.

Existing benchmark suites do not need immediate mechanical rewrites if they are already correct.

New code should normally prefer `B.Loop`.

---

## 9. Setup Cost

Benchmark setup should be separated from the operation under measurement unless setup is part of the claim.

Example:

```go
func BenchmarkLookup(b *testing.B) {
    index := buildIndex()

    for b.Loop() {
        lookup(index, key)
    }
}
```

This measures lookup, not index construction.

---

## 10. When Setup Belongs Inside

If real usage rebuilds state on every operation, setup belongs in the measured region.

Benchmark design must match semantics.

Bad methodology:

```text
remove expensive setup
because it makes benchmark nicer
```

Good methodology:

```text
measure exactly the lifecycle the system uses
```

---

## 11. Input Reuse

Reusing one input can distort:

- cache behavior;
- branch prediction;
- allocations;
- mutation patterns.

Example:

```go
for b.Loop() {
    Parse(sameInput)
}
```

may represent a hot-cache parser rather than production parsing.

Sometimes that is intended.

Document it.

---

## 12. Input Distribution

Performance depends on data shape.

A realistic benchmark may need:

- small/medium/large cases;
- valid/invalid data;
- sorted/random values;
- hit/miss ratios;
- multiple receiver types;
- realistic payload distributions.

One synthetic case can hide the true trade-off.

---

## 13. Branch Prediction

If the benchmark always takes one branch:

```text
predictable
```

but production input is random:

```text
unpredictable
```

the benchmark may understate branch cost.

Representative input distribution is part of benchmark correctness.

---

## 14. Cache Warmth

Repeatedly benchmarking one tiny data set can fit everything into cache.

Production may operate on a larger working set.

Design separate tests when useful:

```text
hot-cache benchmark
cold/large-working-set benchmark
```

Do not accidentally mix them.

---

## 15. Mutation

If the benchmark mutates shared input, later iterations may measure a different state from earlier ones.

Example:

```go
Compress(buf)
```

may alter buffer content/metadata.

Reset state explicitly when required.

Reset cost should be included or excluded based on real lifecycle.

---

## 16. Allocation Metrics

Use benchmark memory reporting to measure:

```text
B/op
allocs/op
```

A candidate claiming allocation reduction should include those metrics.

But:

```text
0 alloc/op
```

is not a goal by itself.

Time, memory retention, and complexity still matter.

---

## 17. `ReportAllocs`

When allocation reporting is not otherwise enabled by command-line options, `b.ReportAllocs()` can request allocation metrics.

Repository benchmark commands should be standardized so comparisons remain consistent.

---

## 18. Custom Metrics

Benchmarks can report domain metrics such as:

- MB/s;
- items/s;
- bytes processed;
- retries/op.

Custom metrics are useful when they directly express the cost model.

Example CAS benchmark:

```text
retries/op
```

can explain why throughput collapses.

---

## 19. `SetBytes`

For throughput-oriented byte processing, setting bytes per operation can produce MB/s-style output.

This is useful for:

- codecs;
- hashing;
- parsers;
- memory copy;
- compression.

---

## 20. Microbenchmarks

A microbenchmark isolates a small mechanism.

Good uses:

- compare two encoding loops;
- verify BCE impact;
- compare allocation shapes;
- test false sharing.

Its purpose is controlled causality.

Its weakness is limited representativeness.

---

## 21. Component Benchmarks

Component benchmarks include more realistic context:

- parser plus allocations;
- cache plus synchronization;
- request processing without network;
- storage engine operation.

They reduce isolation but improve representativeness.

---

## 22. End-to-End Benchmarks

System-level benchmarks measure the behavior users actually experience.

Examples:

- HTTP requests/sec;
- P99 latency;
- CPU/request;
- RSS under load.

They are essential for validating large changes.

They are less useful for identifying microscopic mechanisms.

---

## 23. Benchmark Pyramid

A useful model:

```text
micro
→ mechanism

component
→ subsystem

service
→ system impact

production/canary
→ real-world validation
```

Different layers answer different questions.

---

## 24. Repeated Runs

One benchmark run is rarely enough for small differences.

Run repeated samples.

Example:

```bash
go test -run='^$' -bench='BenchmarkX$' -count=10 ./pkg > old.txt
```

and the same for the candidate.

Current benchstat documentation recommends at least 10 runs for statistically meaningful comparison.

---

## 25. `benchstat`

`golang.org/x/perf/cmd/benchstat` compares Go benchmark result sets.

Typical flow:

```bash
benchstat old.txt new.txt
```

It provides statistical summaries and significance testing.

This is stronger than manually comparing one pair of ns/op numbers.

---

## 26. Statistical Significance

A statistically significant difference means the samples provide evidence that the distributions differ.

It does not mean the engineering effect is large.

Example:

```text
-0.5%
statistically significant
```

may still be irrelevant.

---

## 27. Engineering Significance

Ask:

> Is the improvement large enough to justify complexity?

Examples:

```text
-0.5% local benchmark
+ unsafe aliasing
→ likely reject

-15% CPU in top production hotspot
+ small maintainable change
→ likely valuable
```

Statistics and engineering value are separate gates.

---

## 28. Confidence Intervals

Performance results should be understood as distributions, not exact constants.

Noise sources include:

- scheduling;
- frequency scaling;
- cache state;
- interrupts;
- background processes.

Confidence ranges help communicate uncertainty.

---

## 29. Environment Control

Record:

- CPU model;
- OS/kernel;
- Go version;
- GOARCH;
- GOMAXPROCS;
- benchmark command;
- relevant environment variables;
- PGO state.

Without environment metadata, future reproduction becomes harder.

---

## 30. Go Version

Compiler/runtime changes can shift benchmarks even when source is unchanged.

Examples:

- allocator improvements;
- GC changes;
- inlining changes;
- standard-library optimization.

Always compare candidates under the same toolchain.

---

## 31. CPU Frequency

Modern CPUs change frequency dynamically based on:

- power;
- temperature;
- workload;
- core count.

This can create run-to-run variance.

For high-precision local benchmarking, stable power/performance settings can improve repeatability.

Do not require specialized lab tuning for ordinary large-effect comparisons.

---

## 32. Thermal Effects

A long benchmark can heat the CPU and reduce frequency.

If one candidate always runs first and the other always runs second, ordering can bias results.

Repeated/interleaved runs reduce this risk.

---

## 33. Background Activity

Browsers, IDE indexing, antivirus, container workloads, and CI neighbors can add noise.

Dedicated or quiet systems provide more stable measurements.

Cloud shared runners are often poor environments for detecting 1% changes.

---

## 34. GOMAXPROCS

Concurrency benchmarks must record `GOMAXPROCS`.

Changing available parallelism changes:

- scheduling;
- contention;
- cache coherence;
- GC worker behavior.

A throughput result without concurrency configuration is incomplete.

---

## 35. CPU Affinity

For advanced hardware-sensitive experiments, CPU affinity can reduce migration/noise.

It can also create unrealistic constraints.

Use it only when the hypothesis requires it.

---

## 36. GC State

A benchmark can be influenced by garbage collection.

Do not disable GC merely to obtain prettier numbers unless the experiment explicitly studies allocation CPU independent of GC.

Production validation should use production-like GC configuration.

---

## 37. Race Detector

Never compare normal benchmark results against race-enabled results.

The race detector fundamentally changes execution.

Use it in a separate correctness test stage.

---

## 38. PGO

If production uses PGO, relevant system benchmarks should include the production build configuration.

But compiler-level experiments may intentionally compare with and without PGO.

Always state which configuration is being measured.

---

## 39. Compiler Flags

Flags such as:

```text
-N
-l
```

disable optimizations/inlining and dramatically change performance.

They are useful diagnostic experiments.

They are not normal benchmark baselines.

---

## 40. Benchmarking Concurrency

A single:

```text
GOMAXPROCS=8
```

number is not enough for synchronization changes.

Measure a scaling curve:

```text
1
2
4
8
16
32
```

where hardware supports it.

The shape often reveals the mechanism.

---

## 41. Contention Scenarios

Concurrency benchmarks should separate:

```text
uncontended
low contention
high contention
```

A primitive that wins under one worker may collapse under many.

---

## 42. Read/Write Ratios

For `Mutex` vs `RWMutex`, vary:

- read percentage;
- critical-section duration;
- writer frequency.

"90% reads" alone does not define a workload.

---

## 43. Sharding Benchmarks

Sharding changes both:

- synchronization;
- memory layout.

Measure:

- write throughput;
- read aggregation cost;
- memory overhead;
- shard counts.

Too many shards can hurt cache/memory use.

---

## 44. Lock-Free Benchmarks

Measure more than throughput.

Useful metrics:

- retries/op;
- CPU;
- P99;
- scaling;
- starvation indicators.

A lock-free algorithm can look excellent at low contention and waste enormous CPU at high contention.

---

## 45. Memory Benchmarks

Allocation benchmarks should distinguish:

```text
allocation rate
vs
retention
```

A microbenchmark that ends immediately may not reveal long-lived retained capacity.

Use longer component/system tests for retention.

---

## 46. Zero-Copy Benchmarks

Benchmark realistic payload sizes and lifetimes.

A conversion benchmark alone does not capture:

- retention;
- aliasing;
- downstream cache behavior.

Add memory/system validation where ownership changes.

---

## 47. Benchmarking Compiler Effects

Compiler optimization proofs often combine:

```text
compiler diagnostic
+
assembly
+
benchmark
```

Example BCE:

```text
diagnostic confirms check removed
benchmark measures local effect
```

The diagnostic proves mechanism more directly than timing alone.

---

## 48. Avoid Benchmarking Constants Accidentally

If input is compile-time constant, the compiler may simplify work unrealistically.

Use representative runtime values when the production operation receives runtime data.

---

## 49. Avoid Measuring Setup Accidentally

With legacy benchmarks, forgetting timer control can include expensive setup.

With `B.Loop`, setup before the loop is handled more naturally.

Still read the benchmark as a lifecycle specification.

---

## 50. Avoid Measuring Logging

Logging inside a hot benchmark can dominate the operation.

Diagnostics should not remain in the timed body unless logging itself is being measured.

---

## 51. Avoid Cross-Benchmark Shared State

Benchmarks sharing global mutable state can contaminate each other.

Prefer isolated setup.

If global caches are intentionally part of the system, document warm/cold state.

---

## 52. Avoid Unrealistic Parallel Benchmarks

`RunParallel` is useful for some concurrent operations.

But it does not automatically reproduce production:

- arrival distribution;
- request think time;
- key skew;
- backpressure.

Construct the concurrency workload deliberately.

---

## 53. Key Distribution

Map/cache/lock benchmarks depend heavily on key skew.

Uniform random:

```text
low hot-key concentration
```

Zipf-like workload:

```text
few hot keys
```

can produce dramatically different contention.

Use production-like distributions.

---

## 54. Latency Benchmarks

`testing.B` primarily focuses on aggregate benchmark metrics.

For service latency distributions, use a load-testing harness capable of recording percentiles.

Do not infer P99 from ns/op.

---

## 55. Warmup

JIT warmup is not a Go concern in the traditional sense, but system warmup still exists:

- caches;
- DNS;
- connection pools;
- page cache;
- lazy initialization.

End-to-end benchmarks should define warmup behavior.

---

## 56. Baseline Selection

The baseline should be:

- current production;
- current main branch;
- known reference implementation.

Avoid comparing against an artificially bad strawman.

---

## 57. One Variable at a Time

For mechanism proof, change one key factor.

Example:

```text
baseline global counter
candidate sharded counter
```

Do not simultaneously change:

- data format;
- concurrency;
- algorithm;
- compiler flags.

Otherwise attribution becomes unclear.

---

## 58. Multi-Change Optimization

Real engineering changes can involve multiple coordinated transformations.

In that case:

```text
microbench individual mechanisms
+
component benchmark final design
```

lets you understand both local causes and overall effect.

---

## 59. Benchmark Artifacts

Store:

- benchmark source;
- commands;
- representative input generation;
- environment metadata;
- before/after result files when valuable.

This makes performance claims reproducible.

---

## 60. Proof vs Product Benchmark

A `proofs/` benchmark may intentionally use synthetic input to isolate one mechanism.

A product benchmark should prioritize representative behavior.

These are both valid if the scope is explicit.

---

## 61. Stop Conditions

Do not endlessly tune a benchmark.

Stop when:

- target bottleneck is resolved;
- gains fall below meaningful threshold;
- complexity budget is exhausted;
- system benefit no longer appears;
- next bottleneck lies elsewhere.

---

## 62. Related Official Sources

- `testing`: https://pkg.go.dev/testing
- benchmark format: https://go.dev/design/14313-benchmark-format
- `benchstat`: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat
- Go performance diagnostics: https://go.dev/doc/diagnostics

---

## 63. Engineering Perspective

A good benchmark is not the one with the smallest noise or largest percentage.

It is the one that most clearly answers the engineering question while remaining reproducible.
