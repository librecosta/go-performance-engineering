# Profiling

[English](../../06-methodology/profiling.md) | 简体中文

## 1. Profiling 用来回答什么

Profiling 回答：

> 程序正在把某种资源花在哪里？

不同 profile 对应不同资源：

- CPU time；
- live heap；
- cumulative allocation；
- blocked time；
- mutex contention；
- goroutine；
- OS thread。

Profiling 首先是 **localization tool**。

它把一个大系统缩小到少数值得调查的 function、call path、object 或 synchronization point。

它不会自动回答：

- 为什么成本存在；
- 成本能否删除；
- 哪个 replacement 更快；
- 修改是否改善整个系统。

这些问题还需要 cost model 与 controlled comparison。

## 2. Profiling vs Benchmarking

基础区分：

```text
Profile
→ Where is the cost?

Benchmark
→ Which implementation is better?
```

例如 CPU profile 显示：

```text
runtime.scanobject = 18%
```

并不能直接推出：

```text
tune GC
```

真实原因可能是 pointer-heavy representation 或 high allocation rate。

反过来，microbenchmark 显示 parser B 快 20%，也不证明 parser 是系统 bottleneck。

成熟方法是两者配合。

## 3. 从 Symptom 开始

不要因为工具存在就把所有 profile 都收集一遍。

先从 observed problem 选择最低成本证据。

例如：

```text
CPU saturation
→ CPU profile

RSS growth
→ heap + process memory

high allocation rate
→ allocs profile

lock contention
→ mutex profile

waiting/blocking
→ block profile

latency mystery
→ execution trace

goroutine growth
→ goroutine / goroutineleak
```

## 4. CPU Profile

CPU profile 通过 sampling 近似记录程序把 active CPU time 花在哪里。

常见来源：

- `runtime/pprof`；
- `net/http/pprof`；
- `go test` profile。

例如：

```bash
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

采样时长应该覆盖稳定且具有代表性的 workload。

## 5. Sampling 是 Approximation

CPU profile 不记录每条 instruction。

它擅长定位 dominant CPU consumer。

它不适合单独证明两个几乎相同实现有 1% 差异。

那种问题应该交给 repeated benchmark。

## 6. Flat vs Cumulative

pprof 常见两个视角：

### Flat

Sample 直接落在该 function body 的时间。

### Cumulative

Function 加上所有 callee 的总成本。

如果某 wrapper flat 很小、cum 很大，真正优化点通常在 descendants。

## 7. Interpreting Flat Time

High flat time 可能来自：

- tight compute；
- hash；
- encoding；
- atomic retry；
- GC scanning；
- copying。

但 function name 不是 root cause。

例如 `memmove` 很热可能来自：

- slice growth；
- string conversion；
- serialization；
- buffer assembly。

继续看 callers。

## 8. Interpreting Cumulative Time

Cumulative time 更适合 architecture-level localization。

例如 Handler 本身没做多少 work，却是 45% CPU call tree 的入口。

不要先重写 wrapper，要看它调用了什么。

## 9. Call Graph

应该继续问：

```text
who calls this?
↓
under what workload?
↓
why so often?
```

减少 call count 有时比加速 callee 更有价值。

## 10. Runtime Function 很热时怎么理解

Runtime function 出现在 profile 很正常。

例如：

- GC；
- allocation；
- map；
- synchronization；
- scheduler。

不要停在：

> runtime 是问题。

而要翻译回 application behavior：

```text
GC CPU
→ allocation / pointer graph / live heap

mutex runtime cost
→ contention topology

map runtime cost
→ access frequency / representation
```

## 11. Heap Profile

Heap profile 主要用于 live/in-use memory。

它回答：

> 哪些 allocation site 负责当前仍然 live 的 heap object？

这本质是 retention question。

## 12. Allocs Profile

`allocs` profile 强调 cumulative allocation activity。

它回答：

> 哪些 call path 在长期运行中制造了最多 allocation traffic？

Temporary object 可能在 allocs 很热，在 live heap 中几乎不存在。

## 13. Heap vs Allocs

```text
Heap
→ What remains?

Allocs
→ What churns?
```

这两个问题必须分开。

## 14. Bytes vs Objects

Memory profile 可以看：

- bytes；
- object count。

Few huge buffers 与 millions tiny nodes 代表不同成本模型。

前者偏向 RSS/bandwidth/retention，后者偏向 allocator/GC graph overhead。

## 15. Memory Profile 也是 Sampled

Heap profiling 有 sampling。

它适合识别趋势与 dominant source，不是精确账本。

Per-operation exact allocation 更适合 benchmark `B/op` / `allocs/op`。

## 16. Mutex Profile

Mutex profile 用于观察 synchronization contention。

它更接近回答：

> 哪些 critical section 让其他 goroutine 等待？

而不是单纯“哪里调用了 Lock”。

## 17. Contention Time 可以超过 Wall Time

一个 goroutine 持 lock 1 秒，5 个 goroutine 同时等待，就可以产生约 5 goroutine-seconds waiting。

所以 aggregate contention time 大于 wall-clock 完全正常。

## 18. Mutex Sampling

Mutex profiling 本身有 sampling/overhead。

不要在 production 永久用最高 detail，除非理解成本。

## 19. Block Profile

Block profile 记录 goroutine 在 synchronization 上被阻塞的时间，例如：

- Mutex/RWMutex；
- channel；
- select；
- WaitGroup；
- condition variable。

它回答：

> Goroutine 在哪里无法 progress，因为它正在等？

## 20. Block vs Mutex

```text
mutex profile
→ who causes lock wait

block profile
→ where goroutines wait
```

一个 contention 问题可能需要两边一起看。

## 21. Goroutine Profile

Goroutine profile 是当前 goroutine stack snapshot。

适合：

- goroutine count growth；
- large blocked population；
- duplicate background worker；
- stuck pipeline。

怀疑 leak 时最好做时间对比，而不是只看一次 snapshot。

## 22. `goroutineleak` Profile

现代 Go 提供 `goroutineleak` pprof profile，用 reachability reasoning 检测一类可以证明永久 blocked 的 goroutine。

概念上：

```text
G blocked on synchronization object P
↓
nothing runnable can ever reach/unblock P
↓
G is provably leaked
```

这比“blocked 很久”更强。

## 23. Goroutine Leak Detection 的边界

空 `goroutineleak` profile 不等于没有 leak。

如果 synchronization object 仍通过 global state theoretically reachable，runtime 可能无法证明它永远不能 unblock。

它检测的是可证明的一类 leak。

## 24. Thread Creation Profile

Thread-create profile 可以辅助调查：

- cgo/native blocking；
- OS thread explosion；
- unusual scheduler interaction。

使用频率比 CPU/heap/block 低，但某些 boundary 问题很有价值。

## 25. Execution Trace

Trace 保留 runtime event 的时间顺序。

它适合回答：

```text
latency spike 前发生了什么？
为什么这个 goroutine 没运行？
谁唤醒了它？
GC / syscall / scheduling 发生在什么时候？
```

Profile 聚合，trace 保留 chronology。

## 26. Trace 能看到什么

通常包括：

- goroutine scheduling；
- runnable/running；
- blocking；
- syscall；
- synchronization；
- GC；
- user task/region。

因此特别适合 concurrency/latency investigation。

## 27. Trace 不是“更好的 CPU Profile”

Trace data 更丰富，也更昂贵、更难读。

如果问题只是：

> 谁烧 CPU？

CPU pprof 通常更简单。

只有 temporal causality 重要时再用 trace。

## 28. Flight Recorder

`runtime/trace.FlightRecorder` 可以保留最近一段滚动 trace history。

它解决 rare incident 的典型难题：

```text
symptom detected now
but cause happened seconds earlier
```

可以在 anomaly 发生后 snapshot 之前的 window。

## 29. 为什么 Flight Recorder 很重要

Traditional trace 要在 incident 前决定开始。

Rare P99/P99.9 很难预测。

Flight Recorder 变成：

```text
always keep bounded recent history
↓
detect anomaly
↓
dump preceding trace
```

很适合 long-running service。

## 30. Flight Recorder 的 Buffer Budget

它会占用 memory/CPU。

配置要平衡：

- history duration；
- max bytes；
- overhead。

诊断 buffer 也不能无限增长。

## 31. User Region / Task

Trace annotation 可以把 runtime timeline 映射到业务阶段：

```text
request
parse
database
encode
```

只标记有诊断价值的 domain operation，不要给每个 function 都加 region。

## 32. `runtime/metrics`

`runtime/metrics` 适合 ongoing observability/hypothesis generation。

可以看到 GC、scheduler、memory、runtime CPU 等指标。

## 33. Metrics 不是 Profile

Metric 告诉你：

> Something changed.

Profile 告诉你：

> Cost appears here.

例如 GC CPU 上升后，再用 profile 找 allocation/scan source。

## 34. Metric Set 会演进

Access API 相对稳定，但具体 implementation-defined metric set 会变化。

应该以目标 Go version 的 metric description 为准。

## 35. `GODEBUG=gctrace=1`

GC trace 是很便宜的快速诊断手段，可以看到：

- cadence；
- heap；
- timing；
- CPU contribution。

长期 production observability 更适合 structured metrics。

## 36. Compiler Diagnostics 也是诊断链的一部分

Profile 定位 hot function 后，如果怀疑：

- escape；
- bounds check；
- missed inline；

就可以进一步用 compiler diagnostic 回答：

> 为什么 generated code 仍然包含这项 work？

## 37. pprof Label

Label 可以把 sample 关联到逻辑 workload：

- tenant；
- endpoint；
- operation class。

这有助于回答“哪个 workload 制造 hotspot”。

注意 sensitive/high-cardinality label 风险。

## 38. Production Profiling

Production profile 很有价值，因为它包含真实：

- traffic mix；
- data distribution；
- cache state；
- concurrency；
- dependency。

但必须考虑：

- access control；
- duration；
- endpoint exposure；
- sensitive data；
- overhead。

不要公开暴露 pprof endpoint。

## 39. Representativeness

Profile 只对被采样 workload 有证据价值。

30 秒 backup job profile 不一定代表 normal request traffic。

应记录：

- timestamp；
- version；
- workload condition；
- traffic level；
- Go version。

## 40. Incident vs Baseline Profile

### Incident

回答：

> 这次异常发生了什么？

### Baseline

回答：

> 平时主要资源花在哪里？

两者作用不同。

## 41. Tool Intrusion

Observation 会改变 system。

尤其：

- high-rate block profile；
- detailed trace；
- race；
- checkptr；
- PMU/tracing。

必须知道工具是否扭曲目标行为。

## 42. Race Build 不是 Performance Baseline

Race detector 用于 correctness。

其 instrumentation 会显著改变执行，不应该与 production performance 直接比较。

## 43. `checkptr` 不是 Performance Baseline

同理，pointer-checking instrumentation 是 correctness validation，不是正常性能测量环境。

## 44. Hardware Profiling

如果 pprof 已找到 hot loop，却仍解释不了“为什么慢”，PMU 可以看：

- cycles；
- instructions；
- cache misses；
- branch misses；
- stalls；
- IPC。

这属于 advanced evidence layer。

## 45. IPC

IPC 可以帮助判断 CPU 是在高效执行还是经常 stall。

但 low IPC 只是 symptom，不是 root cause。

可能来自：

- memory latency；
- dependency；
- branch；
- synchronization。

## 46. Cache-Miss Evidence

如果 layout optimization 声称改善 locality，可以用 PMU 加强 mechanism evidence：

```text
same workload
↓
fewer cache misses
↓
fewer cycles/op
↓
higher throughput
```

## 47. NUMA

NUMA 对 multi-socket system 可能重要。

但它是 advanced diagnostic variable。

优先先排除：

- algorithm；
- contention；
- layout；
- ordinary cache。

## 48. Profile Diffing

Before/after profile 可以看到 cost 是否只是移动了。

例如：

```text
JSON CPU ↓
GC CPU ↑
```

不要只盯 intended hotspot。

## 49. Profile-Driven Loop

推荐流程：

```text
1. Define symptom
2. Collect appropriate profile
3. Identify dominant cost
4. Build mechanism hypothesis
5. Design candidate
6. Benchmark A/B
7. Re-profile system
8. Verify cost moved as expected
```

## 50. Profile Quality Checklist

信任 profile 前检查：

- workload representative？
- duration 足够？
- system warm？
- traffic stable？
- overhead acceptable？
- profile type 正确？
- sampled binary/source version 对得上？

## 51. 官方资料

- `runtime/pprof`: https://pkg.go.dev/runtime/pprof
- `net/http/pprof`: https://pkg.go.dev/net/http/pprof
- `runtime/trace`: https://pkg.go.dev/runtime/trace
- Flight Recorder: https://go.dev/blog/flight-recorder
- `runtime/metrics`: https://pkg.go.dev/runtime/metrics
- Diagnostics: https://go.dev/doc/diagnostics

## 52. 工程视角

Profiling 的成功标准是把：

> 我觉得这段代码看起来很贵。

变成：

> 在这个 workload 下，这条 call path 占据了这部分被测量成本。

从这一刻开始，真正的优化工作才有依据。
