# Official Sources

This document is the primary-source index for the Go Performance Engineering Handbook.

It is intentionally more detailed than the Agent Skill source router. Its purpose is to preserve the authoritative documentation and implementation sources behind the handbook's technical claims.

## Source Policy

For version-sensitive Go performance claims, prefer evidence in this order:

```text
1. Current Go release notes
2. Current language / package documentation
3. Current compiler and runtime source
4. Official Go blog, design, and proposal material
5. External benchmark or technical article as secondary evidence only
```

Do not use an old blog post as a substitute for current compiler/runtime behavior.

When a claim depends on implementation details, record the target:

```text
Go version
GOOS
GOARCH
CPU / hardware where relevant
```

This handbook currently uses **Go 1.27** as its primary toolchain baseline.

## Release and Versioning

- [Go 1.27 Release Notes](https://go.dev/doc/go1.27)
- [Go 1.27 Release Blog](https://go.dev/blog/go1.27)
- [Go Release History](https://go.dev/doc/devel/release)
- [Go 1.26 Release Notes](https://go.dev/doc/go1.26)
- [Go 1.26 Release Blog](https://go.dev/blog/go1.26)

Use release notes first when validating whether compiler, runtime, GC, cgo, or standard-library behavior changed between Go releases.

---

## 00 — Foundations

### Language and Memory Semantics

- [The Go Language Specification](https://go.dev/ref/spec)
- [The Go Memory Model](https://go.dev/ref/mem)

The memory model is the primary contract for synchronization and visibility between goroutines. Runtime implementation details must not be used to weaken language-level correctness requirements.

### Diagnostics Overview

- [Diagnostics](https://go.dev/doc/diagnostics)

This is the primary Go overview for selecting profiling, tracing, debugging, and runtime-observation tools.

---

## 01 — CPU and Memory

Go does not define CPU cache, TLB, branch-predictor, or NUMA behavior as language contracts. Hardware-sensitive claims therefore require target-machine evidence.

Useful Go sources:

- [`golang.org/x/sys/cpu`](https://pkg.go.dev/golang.org/x/sys/cpu)
- [`runtime` package](https://pkg.go.dev/runtime)

For hardware-counter experiments on Linux:

- [Linux `perf-stat(1)` manual](https://man7.org/linux/man-pages/man1/perf-stat.1.html)

Hardware-counter results should be treated as target-machine evidence, not universal Go behavior.

---

## 02 — Concurrency

### Public Contracts

- [The Go Memory Model](https://go.dev/ref/mem)
- [`sync`](https://pkg.go.dev/sync)
- [`sync/atomic`](https://pkg.go.dev/sync/atomic)

### Runtime / Standard-Library Implementation

- [Internal Mutex implementation](https://go.dev/src/internal/sync/mutex.go)
- [Runtime channel implementation](https://go.dev/src/runtime/chan.go)
- [Runtime semaphore implementation](https://go.dev/src/runtime/sema.go)
- [Runtime lock-free stack](https://go.dev/src/runtime/lfstack.go)

Use runtime source to understand cost models such as spinning, parking, queueing, and tagged lock-free state.

Do not treat private runtime implementation as a public API contract.

---

## 03 — Compiler

### Compiler Overview

- [Go Compiler README](https://go.dev/src/cmd/compile/README)
- [`cmd/compile` documentation](https://pkg.go.dev/cmd/compile)

### Compiler Source

- [Escape analysis](https://go.dev/src/cmd/compile/internal/escape/)
- [SSA implementation](https://go.dev/src/cmd/compile/internal/ssa/)
- [Devirtualization](https://go.dev/src/cmd/compile/internal/devirtualize/)
- [Inlining](https://go.dev/src/cmd/compile/internal/inline/)
- [Bounds-check diagnostics (`checkbce.go`)](https://go.dev/src/cmd/compile/internal/ssa/checkbce.go)

### Profile-Guided Optimization

- [Profile-Guided Optimization Guide](https://go.dev/doc/pgo)
- [Profile-Guided Optimization in Go](https://go.dev/blog/pgo)

### Recent Compiler / Allocation Context

- [Allocating on the Stack](https://go.dev/blog/allocation-optimizations)
- [Go 1.27 Release Notes](https://go.dev/doc/go1.27)

Compiler decisions such as inlining, escape placement, BCE, devirtualization, generic lowering, and specialized allocation are implementation-sensitive and must be checked on the target toolchain.

---

## 04 — Memory and GC

### Garbage Collector

- [A Guide to the Go Garbage Collector](https://go.dev/doc/gc-guide)
- [Runtime GC implementation](https://go.dev/src/runtime/mgc.go)
- [GC pacer](https://go.dev/src/runtime/mgcpacer.go)
- [GC scavenger](https://go.dev/src/runtime/mgcscavenge.go)

### Allocator

- [Runtime allocator](https://go.dev/src/runtime/malloc.go)
- [Runtime heap implementation](https://go.dev/src/runtime/mheap.go)

### Runtime Metrics and Tuning

- [`runtime/metrics`](https://pkg.go.dev/runtime/metrics)
- [`runtime/debug`](https://pkg.go.dev/runtime/debug)

Use `runtime/debug` as the public reference for controls such as GC percentage and the runtime memory limit.

### Green Tea GC

- [The Green Tea Garbage Collector](https://go.dev/blog/greenteagc)
- [Go 1.26 Release Notes](https://go.dev/doc/go1.26)

Green Tea became the default GC in Go 1.26. Historical GC optimization results should be revalidated on current Go versions rather than assumed to carry forward unchanged.

### Modern Allocation Behavior

- [Allocating on the Stack](https://go.dev/blog/allocation-optimizations)
- [Go 1.27 Release Notes](https://go.dev/doc/go1.27)

These sources are especially important when evaluating historical rules about small allocations or slice backing-store escape behavior.

---

## 05 — Runtime Boundary

### `unsafe`

- [`unsafe`](https://pkg.go.dev/unsafe)

Use the public functions documented here for modern low-level pointer, slice, and string operations. Prefer these APIs over historical `reflect.StringHeader` / `reflect.SliceHeader` reinterpretation patterns.

### Runtime Lifetime APIs

- [`runtime`](https://pkg.go.dev/runtime)
- [`runtime.KeepAlive`](https://pkg.go.dev/runtime#KeepAlive)
- [`runtime.Pinner`](https://pkg.go.dev/runtime#Pinner)

These are the primary public contracts for lifetime-sensitive runtime interaction.

### cgo / FFI

- [`cmd/cgo`](https://pkg.go.dev/cmd/cgo)
- [`runtime/cgo`](https://pkg.go.dev/runtime/cgo)
- [`runtime/cgo.Handle`](https://pkg.go.dev/runtime/cgo#Handle)

Use `cmd/cgo` documentation for pointer-passing rules and cgo directives such as `noescape` and `nocallback`.

### Compiler Directives

- [Compiler directive documentation](https://go.dev/src/cmd/compile/doc.go)

This is the primary source for directives such as:

```text
//go:noescape
//go:nosplit
//go:linkname
```

These directives are not interchangeable performance hints. Some are privileged correctness contracts.

### Runtime Internals

- [Go runtime source](https://go.dev/src/runtime/)
- [Runtime lock-free stack](https://go.dev/src/runtime/lfstack.go)

Runtime internals are useful for implementation archaeology and cost-model understanding, not as stable third-party APIs.

### OS Interfaces / mmap

- [`golang.org/x/sys/unix`](https://pkg.go.dev/golang.org/x/sys/unix)

For mmap behavior, also consult the target operating system's primary system-call documentation.

---

## 06 — Methodology

### Profiling

- [`runtime/pprof`](https://pkg.go.dev/runtime/pprof)
- [`net/http/pprof`](https://pkg.go.dev/net/http/pprof)
- [Diagnostics](https://go.dev/doc/diagnostics)

As of Go 1.27, `runtime/pprof` includes the generally available `goroutineleak` profile.

### Execution Trace

- [`runtime/trace`](https://pkg.go.dev/runtime/trace)
- [Flight Recorder in Go 1.25](https://go.dev/blog/flight-recorder)

Use execution trace when temporal causality, scheduling, blocking, or goroutine interaction matters more than aggregate hotspot attribution.

### Benchmarking

- [`testing`](https://pkg.go.dev/testing)
- [More Predictable Benchmarking with `testing.B.Loop`](https://go.dev/blog/testing-b-loop)

`testing.B.Loop` is the preferred form for new Go benchmarks.

### Statistical Comparison

- [`golang.org/x/perf`](https://pkg.go.dev/golang.org/x/perf)
- [`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)

Use repeated A/B samples and statistical comparison for small benchmark differences. Statistical significance must still be separated from engineering significance.

---

## Proof and Reproducibility Sources

The following sources are especially useful when building repository proofs:

| Mechanism | Primary Evidence |
|---|---|
| Escape analysis | compiler `-m` diagnostics + compiler source |
| Inlining | compiler `-m` diagnostics + generated code |
| BCE | `ssa/check_bce` diagnostics + assembly |
| Devirtualization | compiler diagnostics / generated code / PGO |
| Allocation | benchmark alloc metrics + allocs profile |
| Retention | heap profile + runtime memory metrics |
| Mutex contention | mutex profile + scaling benchmark |
| Blocking | block profile + execution trace |
| Lock-free / ABA | algorithm invariant + stress/correctness tests |
| Cache / false sharing | scaling benchmark + PMU when needed |
| Zero-copy | allocation/copy benchmark + lifetime/retention validation |
| GC tuning | GC metrics + representative service workload |
| PGO | representative CPU profile + PGO/non-PGO A/B |

The benchmark result should demonstrate the local effect. The primary source should explain why the mechanism is expected to exist.

---

## Source Quality Rules

### Prefer Current Sources

For compiler/runtime claims:

```text
current target toolchain
>
historical implementation
```

A technique that was necessary on Go 1.22 may be redundant or harmful on Go 1.27.

### Separate Contract from Implementation

Examples of stable or public contracts:

```text
Go memory model
sync / sync/atomic semantics
unsafe package API
runtime.KeepAlive
runtime.Pinner
cmd/cgo rules
```

Examples of implementation-sensitive behavior:

```text
inline decision
escape result
BCE result
allocator size-class details
GC implementation
private runtime symbols
specific machine instructions
```

Always label the second category as version-sensitive.

### External Sources Are Secondary

External articles, conference talks, benchmarks, and blog posts may provide valuable examples.

They should not override:

- current release notes;
- current package documentation;
- current compiler/runtime source;
- reproduced measurements on the target environment.

### Preserve Provenance

Important performance claims should record enough context to be reproduced:

```text
Go version
GOOS / GOARCH
CPU
GOMAXPROCS
benchmark/profile command
relevant flags or experiments
source commit
```

The goal of this source index is not to make every chapter citation-heavy.

Its purpose is to make the technical authority behind the handbook recoverable.
