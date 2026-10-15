# Profile-Guided Optimization

## 1. Why Static Analysis Has Limits

The compiler sees program structure.

It does not automatically know production behavior.

Static analysis may know:

```text
this interface has five possible implementations
```

but not:

```text
99% of real calls use implementation A
```

It may know:

```text
these two functions are both reachable
```

but not:

```text
one is extremely hot and the other almost never runs
```

Profile-guided optimization (PGO) adds runtime evidence to compiler decisions.

---

## 2. What a CPU Profile Contributes

A CPU profile approximates where real workloads spend CPU.

It can provide information about:

- hot functions;
- hot call edges;
- common dynamic receivers.

The compiler can use this information to spend optimization effort where it matters most.

---

## 3. PGO Is Not Runtime JIT

Go PGO is still ahead-of-time compilation.

Conceptually:

```text
production-like workload
      ↓
CPU profile
      ↓
go build with PGO
      ↓
optimized binary
```

The compiler uses the profile when producing the binary.

---

## 4. Hot Inlining

Inlining has code-size trade-offs.

Without profile data, the compiler uses heuristics.

PGO can indicate that a call site is important enough to justify more aggressive optimization.

This can unlock downstream:

- escape improvements;
- constant propagation;
- BCE.

---

## 5. Profile-Guided Devirtualization

Suppose an interface call has multiple valid receiver types.

A profile may show one dominant concrete receiver.

The compiler can specialize that common path.

Conceptually:

```text
hot receiver
→ direct path

other receivers
→ generic fallback
```

This preserves correctness while optimizing common production behavior.

---

## 6. Representative Profiles

The quality of PGO depends on the quality of the profile.

A profile should represent the workload the binary is expected to serve.

Bad profile choices include:

- one rare incident;
- a synthetic benchmark unrelated to production;
- only one endpoint when traffic is diverse;
- debugging workload.

---

## 7. Workload Drift

Production workload changes.

A PGO profile from months ago may no longer represent:

- endpoint distribution;
- data shape;
- receiver types;
- hot call paths.

Therefore PGO artifacts need lifecycle management.

They are performance inputs, not permanent truth.

---

## 8. One Workload vs Many Workloads

Services may have multiple important modes.

Examples:

- read-heavy daytime traffic;
- write-heavy batch jobs;
- regional traffic differences.

A single profile may overrepresent one mode.

Possible strategies include:

- collecting a representative mixed profile;
- using profiles from the dominant business workload;
- merging profiles where appropriate;
- maintaining workload-specific binaries only when justified.

---

## 9. PGO and Architecture Design

PGO is particularly valuable because it can preserve clean abstractions.

Instead of manually special-casing every hot interface or helper, the compiler can specialize paths using observed behavior.

This reduces pressure to distort source architecture prematurely.

---

## 10. Comparing PGO Builds

A proper evaluation compares:

```text
same source
same toolchain
same environment
PGO off
vs
PGO on
```

under a representative workload.

Useful measurements include:

- throughput;
- latency;
- CPU;
- binary size;
- allocation changes.

---

## 11. PGO Can Change Secondary Metrics

More aggressive inlining may:

- increase binary size;
- change instruction-cache behavior;
- change escape results;
- alter allocation patterns.

Therefore PGO should not be evaluated only by one benchmark number.

---

## 12. PGO and Microbenchmarks

A microbenchmark profile may overfit the compiler to behavior that is not representative of the full application.

PGO inputs should preferably come from:

- production;
- staging with production-like traffic;
- representative integration workloads.

---

## 13. Diagnostic Profile vs PGO Profile

A profile collected specifically during an incident answers:

> What went wrong during this anomaly?

A PGO profile answers:

> What is normally hot enough to optimize?

Those are different questions.

Do not automatically use an incident profile as a PGO artifact.

---

## 14. Profile Freshness

PGO should be re-evaluated when:

- major features ship;
- traffic mix changes;
- dependencies change significantly;
- Go version changes;
- architecture changes.

The optimization is tied to workload reality.

---

## 15. Build Reproducibility

PGO introduces an additional build input.

For reproducible engineering, record:

- source commit;
- Go version;
- profile origin;
- profile date/workload;
- build flags.

A binary built with an unknown stale profile is harder to reason about.

---

## 16. PGO Does Not Replace Profiling

PGO is not a substitute for performance investigation.

If the application allocates excessively or serializes through one global lock, PGO will not magically fix the architecture.

It optimizes compiler decisions within the existing program.

---

## 17. Expected Benefit

PGO benefits are workload-dependent.

Some programs improve significantly.

Others change little because:

- hot code is already optimized well;
- bottlenecks are external;
- contention dominates;
- memory bandwidth dominates;
- profile information adds little new knowledge.

Do not treat one published percentage as a universal expectation.

---

## 18. Version Sensitivity

PGO implementation evolves with Go releases.

The compiler may add:

- new profile uses;
- better devirtualization;
- improved inlining heuristics.

Therefore PGO should be benchmarked again after toolchain upgrades.

---

## 19. Common Misconceptions

### "PGO is only for huge applications"

False.

### "PGO replaces hand optimization"

False.

### "Any CPU profile is a good PGO profile"

False.

### "PGO always improves performance"

False.

### "PGO changes Go semantics"

No; it changes optimization decisions while preserving program semantics.

---

## 20. Engineering Perspective

PGO gives the compiler something static analysis cannot fully recover:

> Real information about what the application actually does most often.

Its greatest value is not one specific optimization.

It is aligning compiler effort with production reality.
