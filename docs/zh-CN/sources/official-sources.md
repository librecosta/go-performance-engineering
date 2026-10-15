# 官方资料索引

[English](../../sources/official-sources.md) | 简体中文

本文是 Go Performance Engineering Handbook 的一手资料索引。

它比 Agent Skill 中的 source router 更详细，目的不是让每篇文档充满 citation，而是保存支撑 Handbook 技术结论的 authoritative documentation 与 implementation source。

## 资料优先级

对于 Go 性能相关、尤其是 version-sensitive 的事实，建议按以下优先级取证：

```text
1. 当前 Go release notes
2. 当前 language / package documentation
3. 当前 compiler / runtime source
4. Go 官方 blog、design、proposal
5. 外部 benchmark / technical article，仅作为 secondary evidence
```

不要用一篇多年以前的 blog 替代当前 compiler/runtime 的真实行为。

如果结论依赖 implementation detail，应记录：

```text
Go version
GOOS
GOARCH
CPU / hardware when relevant
```

当前 Handbook 以 **Go 1.27** 作为主要 toolchain baseline。

## Release 与 Versioning

- [Go 1.27 Release Notes](https://go.dev/doc/go1.27)
- [Go 1.27 Release Blog](https://go.dev/blog/go1.27)
- [Go Release History](https://go.dev/doc/devel/release)
- [Go 1.26 Release Notes](https://go.dev/doc/go1.26)
- [Go 1.26 Release Blog](https://go.dev/blog/go1.26)

当需要判断 compiler、runtime、GC、cgo 或标准库行为是否跨版本变化时，release notes 应该是第一站。

---

## 00 — 基础

### Language 与 Memory Semantics

- [The Go Language Specification](https://go.dev/ref/spec)
- [The Go Memory Model](https://go.dev/ref/mem)

Go Memory Model 是 goroutine 间 synchronization/visibility 的语言级 contract。

Runtime implementation detail 不能用来弱化 language-level correctness。

### Diagnostics

- [Diagnostics](https://go.dev/doc/diagnostics)

这是选择 profiling、tracing、debugging 与 runtime observation 工具的重要官方入口。

---

## 01 — CPU 与内存

Go language 不定义 CPU cache、TLB、branch predictor、NUMA 等硬件行为。

因此 hardware-sensitive claim 必须依赖目标机器上的 evidence。

Go 侧可参考：

- [`golang.org/x/sys/cpu`](https://pkg.go.dev/golang.org/x/sys/cpu)
- [`runtime` package](https://pkg.go.dev/runtime)

Linux hardware-counter experiment：

- [Linux `perf-stat(1)` manual](https://man7.org/linux/man-pages/man1/perf-stat.1.html)

PMU 结果只能说明 target machine/workload 下的行为，不能直接升级为 universal Go rule。

---

## 02 — 并发

### Public Contract

- [The Go Memory Model](https://go.dev/ref/mem)
- [`sync`](https://pkg.go.dev/sync)
- [`sync/atomic`](https://pkg.go.dev/sync/atomic)

### Runtime / Standard-Library Implementation

- [Internal Mutex implementation](https://go.dev/src/internal/sync/mutex.go)
- [Runtime channel implementation](https://go.dev/src/runtime/chan.go)
- [Runtime semaphore implementation](https://go.dev/src/runtime/sema.go)
- [Runtime lock-free stack](https://go.dev/src/runtime/lfstack.go)

这些 source 很适合研究：

- spinning；
- parking；
- queueing；
- starvation control；
- tagged lock-free state。

但 private runtime behavior 不属于 public compatibility contract。

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

Inlining、escape、BCE、devirtualization、generic lowering、specialized allocation 都属于 implementation-sensitive behavior。

必须以目标 toolchain 的 diagnostics/generated code 为准。

---

## 04 — Memory 与 GC

### Garbage Collector

- [A Guide to the Go Garbage Collector](https://go.dev/doc/gc-guide)
- [Runtime GC implementation](https://go.dev/src/runtime/mgc.go)
- [GC pacer](https://go.dev/src/runtime/mgcpacer.go)
- [GC scavenger](https://go.dev/src/runtime/mgcscavenge.go)

### Allocator

- [Runtime allocator](https://go.dev/src/runtime/malloc.go)
- [Runtime heap implementation](https://go.dev/src/runtime/mheap.go)

### Runtime Metrics 与 Tuning

- [`runtime/metrics`](https://pkg.go.dev/runtime/metrics)
- [`runtime/debug`](https://pkg.go.dev/runtime/debug)

`runtime/debug` 是 GOGC percentage、runtime memory limit 等 public tuning API 的主要参考。

### Green Tea GC

- [The Green Tea Garbage Collector](https://go.dev/blog/greenteagc)
- [Go 1.26 Release Notes](https://go.dev/doc/go1.26)

Green Tea 在 Go 1.26 成为默认 GC。

这意味着历史 GC microbenchmark 与某些 implementation-specific trick 必须在现代 Go 上重新验证。

### Modern Allocation Behavior

- [Allocating on the Stack](https://go.dev/blog/allocation-optimizations)
- [Go 1.27 Release Notes](https://go.dev/doc/go1.27)

分析 small allocation、slice backing-store escape 等历史结论时，应优先查当前 compiler/runtime 行为。

---

## 05 — Runtime Boundary

### `unsafe`

- [`unsafe`](https://pkg.go.dev/unsafe)

Modern low-level pointer/slice/string work 应优先使用这里的 documented public API。

不要把历史 `reflect.StringHeader` / `reflect.SliceHeader` trick 当成现代默认写法。

### Runtime Lifetime APIs

- [`runtime`](https://pkg.go.dev/runtime)
- [`runtime.KeepAlive`](https://pkg.go.dev/runtime#KeepAlive)
- [`runtime.Pinner`](https://pkg.go.dev/runtime#Pinner)

这些是 lifetime-sensitive runtime interaction 的主要 public contract。

### cgo / FFI

- [`cmd/cgo`](https://pkg.go.dev/cmd/cgo)
- [`runtime/cgo`](https://pkg.go.dev/runtime/cgo)
- [`runtime/cgo.Handle`](https://pkg.go.dev/runtime/cgo#Handle)

Go pointer passing rule、`noescape`、`nocallback` 等应以 `cmd/cgo` current documentation 为准。

### Compiler Directives

- [Compiler directive documentation](https://go.dev/src/cmd/compile/doc.go)

这里是以下 directive 的重要一手资料：

```text
//go:noescape
//go:nosplit
//go:linkname
```

它们不是可以任意互换的“性能 hint”，其中一些属于 privileged correctness contract。

### Runtime Internals

- [Go runtime source](https://go.dev/src/runtime/)
- [Runtime lock-free stack](https://go.dev/src/runtime/lfstack.go)

Runtime internal 适合做 implementation archaeology 和 cost-model research，而不是 third-party stable API。

### OS / mmap

- [`golang.org/x/sys/unix`](https://pkg.go.dev/golang.org/x/sys/unix)

mmap 的完整行为还需要参考目标 OS 的 primary system-call documentation。

---

## 06 — 方法论

### Profiling

- [`runtime/pprof`](https://pkg.go.dev/runtime/pprof)
- [`net/http/pprof`](https://pkg.go.dev/net/http/pprof)
- [Diagnostics](https://go.dev/doc/diagnostics)

Go 1.27 中 `goroutineleak` profile 已进入正式可用阶段，具体行为以当前 `runtime/pprof` 文档为准。

### Execution Trace

- [`runtime/trace`](https://pkg.go.dev/runtime/trace)
- [Flight Recorder in Go 1.25](https://go.dev/blog/flight-recorder)

当 temporal causality、scheduler、blocking、goroutine interaction 比 aggregate hotspot 更重要时，execution trace 更适合。

### Benchmarking

- [`testing`](https://pkg.go.dev/testing)
- [More Predictable Benchmarking with `testing.B.Loop`](https://go.dev/blog/testing-b-loop)

新 Go benchmark 应优先使用 `testing.B.Loop`。

### Statistical Comparison

- [`golang.org/x/perf`](https://pkg.go.dev/golang.org/x/perf)
- [`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)

小幅 benchmark difference 应使用 repeated A/B sample + statistical comparison。

同时必须区分：

```text
statistical significance
≠
engineering significance
```

---

## Proof 与 Reproducibility

构建仓库 `proofs/` 时，可以优先使用以下证据：

| Mechanism | Primary Evidence |
|---|---|
| Escape analysis | compiler `-m` diagnostics + compiler source |
| Inlining | compiler `-m` + generated code |
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

Benchmark 负责证明 local effect。

Primary source 负责说明为什么这个 mechanism 在当前 Go/runtime/hardware model 中合理存在。

---

## Source Quality Rules

### 优先 Current Source

对于 compiler/runtime claim：

```text
current target toolchain
>
historical implementation
```

Go 1.22 需要的 workaround，在 Go 1.27 可能已经 redundant，甚至反而更差。

### 区分 Contract 与 Implementation

较稳定/public 的 contract，例如：

```text
Go memory model
sync / sync/atomic semantics
unsafe package API
runtime.KeepAlive
runtime.Pinner
cmd/cgo rules
```

Implementation-sensitive 的行为，例如：

```text
inline decision
escape result
BCE result
allocator size-class details
GC implementation
private runtime symbols
specific machine instructions
```

第二类必须明确标记 version-sensitive。

### 外部资料只做 Secondary Evidence

External article、conference talk、benchmark、blog 可以提供很好的案例和思路。

但它们不应覆盖：

- current release notes；
- current package docs；
- current compiler/runtime source；
- target environment 的 reproduced measurement。

### Preserve Provenance

重要 performance claim 应记录足够复现实验的信息：

```text
Go version
GOOS / GOARCH
CPU
GOMAXPROCS
benchmark/profile command
relevant flags / GOEXPERIMENT
source commit
```

这个 source index 的目标不是让每篇 Handbook 都变成 citation list。

它的作用是让技术结论背后的 authority 可以被未来维护者重新找到。
