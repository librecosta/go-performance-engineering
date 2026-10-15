# Garbage Collector

[English](../../04-memory-and-gc/garbage-collector.md) | 简体中文

## 1. 为什么 Go 使用 GC

Go 允许程序创建 heap object，而不需要手工为每个 object free。

Garbage Collector 根据 reachability 判断何时 object 不再可达，并让 memory 可复用。

这大幅简化 ownership，但 runtime 必须持续付出 tracing 成本。

理解 GC performance 就是在理解：

```text
what must be traced
how often tracing occurs
how much work allocation creates
```

## 2. Tracing GC

Go 使用 tracing garbage collection。

概念上：

```text
roots
 ↓
follow pointers
 ↓
mark reachable objects
 ↓
unmarked objects are dead
```

Collector 不问“谁被显式 free”，而问“谁还 reachable”。

## 3. Roots

Tracing 从 root 开始，例如：

- global variable；
- goroutine stack；
- runtime-maintained root。

然后沿 Go pointer 遍历 object graph。

这使 pointer density 与 graph shape 成为重要性能变量。

## 4. Marking

例如：

```text
root
 ↓
A → B → C
    ↓
    D
```

A reachable 后，collector 继续追踪 B、D，再继续向下。

Mark work 不只取决于 allocated bytes，也取决于 reachable pointer-bearing structure。

## 5. Pointer-Free Object

Byte array 没有 Go pointer。

一旦 collector 知道 object reachable，就不需要把每个 byte 当成 reference。

所以：

```go
[]byte
```

与：

```go
[]*Node
```

即使 bytes 接近，也会有不同 scan cost。

## 6. Object Graph Shape

Flat dense representation 相比 millions scattered linked nodes，通常具有：

- fewer objects；
- fewer pointers；
- better locality；
- less metadata traversal。

这也是 pointer→index 同时改善 cache 与 GC 的原因。

## 7. Concurrent Collection

现代 Go GC 大部分工作与 application goroutine concurrent 进行，从而保持小 STW。

但 concurrent GC 仍然消耗 CPU。

所以：

```text
small STW
```

不等于：

```text
zero GC latency cost
```

Application 与 collector 仍然竞争 CPU。

## 8. Write Barrier

Concurrent marking 期间 application 仍会修改 pointer graph。

例如：

```go
a.Next = b
```

Collector 需要 write barrier 维护必要 invariant。

所以 pointer mutation 在 mark phase 中会有额外成本。

## 9. Allocation During GC

Application 在 GC 运行时仍继续 allocation。

Collector 必须在 heap 超过 target 前完成足够 work。

如果 allocation 太快，allocating goroutine 可能被要求 assist。

## 10. GC Assist

概念上：

```text
goroutine allocates quickly
      ↓
accumulates GC work debt
      ↓
must perform marking work
```

这是 pacer 的 feedback mechanism。

它能帮助 GC 跟上 heap growth，但会增加 request latency。

## 11. Sweeping

Marking 确认 unreachable object 后，sweep 让 dead slot 重新可用于 allocator。

Go 大量 sweep work 也以 concurrent 方式进行。

很多 workload 中 marking/scanning 比 sweep 更重要。

## 12. Reclamation vs OS Release

Object death：

```text
GC reclaims object slot
```

意味着 Go 可以 reuse memory。

不等于：

```text
OS RSS immediately drops
```

Page release 是独立 scavenger 过程。

## 13. Green Tea GC

Green Tea 在现代 Go 中成为默认 collector，并重构了 marking/scanning 的局部性与 scalability，特别针对 small-object-heavy heap。

稳定结论不是：

> pointer-heavy structure 现在免费了。

而是：

> GC implementation 已变化，旧 microbenchmark 必须在当前 Go version 重新验证。

Allocation rate、live heap、pointer density、graph shape 仍然重要。

## 14. Small-Object Scanning

历史上，扫描大量 scattered small object 会形成差 locality。

Green Tea 改善了 scanning organization，使 collector 更好地批量处理 memory。

具体收益 workload-dependent。

## 15. GC CPU vs Pause Time

两个指标要分开：

### Pause

Application globally stopped 多久？

### CPU

Collector 总共消耗多少 processor time？

一个 service 可以 STW 极小，同时花大量 CPU 做 concurrent GC。

## 16. GC 与 Tail Latency

潜在 GC latency 来源包括：

- STW；
- mutator assist；
- CPU competition；
- write barrier；
- root scanning interaction。

因此不能只看 pause duration。

## 17. Allocation Rate

GC cycle frequency 强烈受 heap growth 速度影响。

一个程序可以：

```text
small stable live heap
```

同时：

```text
huge allocation churn
```

并因此频繁 GC。

## 18. Live Heap

如果大量 heap object GC 后仍 reachable，就不能被 reclaim。

更频繁 GC 解决不了大 live set。

需要：

- retain less；
- compact representation；
- more memory budget。

## 19. Pointer Density

10 GiB pointer-free cache 与 10 GiB pointer graph 的 GC 成本不一样。

因此需要同时理解：

```text
live bytes
+
scannable bytes
+
object count
+
pointer topology
```

## 20. Historical Heap Ballast

Heap ballast 曾用一个巨大 long-lived pointer-free allocation 人为增大 live heap。

在旧 GOGC pacing 下：

```text
larger apparent live heap
→ larger heap target
→ less frequent GC
```

它利用 virtual-memory behavior 达成 memory→CPU trade。

现代 Go 已有 GOMEMLIMIT 等正式机制。

Ballast 应作为历史机制理解，而不是默认 recommendation。

## 21. Finalizer / Cleanup

GC-driven finalization 是 nondeterministic。

File、socket、transaction 等 external resource 应尽量 explicit lifecycle。

GC 负责 memory reachability，不负责 deterministic resource release。

## 22. GC 解决不了什么

GC 不能解决：

- excessive live data；
- bad cache locality；
- memory bandwidth；
- mutex contention；
- external mmap/cgo memory；
- bad ownership design。

GC tuning 不是 architecture 的替代品。

## 23. 官方资料

- GC guide: https://go.dev/doc/gc-guide
- Green Tea GC: https://go.dev/blog/greenteagc
- Runtime GC source: https://go.dev/src/runtime/mgc.go

## 24. 工程视角

可以把 GC 看成 application 持续付费的一项 runtime service。

真正决定“账单”的是：

```text
allocation rate
live heap
pointer density
object graph
```

最强的 GC optimization 经常是减少这些输入，而不是微调 collector 本身。
