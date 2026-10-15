# Regression Strategy

[English](../../06-methodology/regression-strategy.md) | 简体中文

## 1. 为什么 Performance Regression 更难保护

Functional test 往往是：

```text
pass
fail
```

Performance metric 有 noise。

代码没变，benchmark 也可能波动。

因此 performance guard 不能总是简单写：

```text
if ns/op > exact threshold:
    fail
```

需要根据 mechanism、effect size、环境噪声选择 guard。

## 2. Regression Guard 保护什么

可以保护：

### Mechanism

> 这个 parser hot path 应保持 allocation-free。

### Outcome

> 这个 endpoint 的 CPU/request 应保持在 budget 内。

Local optimization 常适合 mechanism guard，系统目标需要 outcome guard。

## 3. Regression Pyramid

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

不是每个 optimization 都需要全部层。

## 4. Functional Correctness First

Performance guard 只是在 unit/race/fuzz/integration 之上增加 protection。

错误但很快的实现不是 valid baseline。

## 5. Allocation Contract

一些 hot API 可以明确用 allocation assertion：

```go
allocs := testing.AllocsPerRun(100, func() {
    parse(input)
})
```

当 specific allocation count 真的是 contract 时，它非常有价值。

## 6. Allocation Count 也可能 Version-Sensitive

Go upgrade 可能改变 compiler allocation behavior。

Strict assertion fail 不一定是坏事，它强制 revalidation。

但不要给所有 function 都加 allocation contract。

## 7. Compiler Diagnostic Guard

Specialized library 可以定期检查：

- escape；
- BCE；
- assembly。

普通 application 通常会过于 brittle。

## 8. Proof 作为 Regression Reference

例如：

```text
proofs/compiler/bounds-check-elimination
```

Toolchain upgrade 后 rerun proof。

如果 workaround 已不必要，就应该简化 source。

## 9. Microbenchmark Guard

长期保留 benchmark 本身就是 executable contract。

即使 CI 不 hard fail，它仍能：

- compare release；
- bisect；
- rerun after upgrade。

## 10. 为什么 CI Performance 很难

Shared CI 存在：

- noisy neighbor；
- CPU model change；
- frequency scaling；
- virtualization；
- background task。

严格 2% threshold 容易制造 false positive。

## 11. Large Regression vs Small Regression

CI 更容易可靠发现：

```text
+50%
```

而不是：

```text
+2%
```

小 regression 更适合 dedicated machine + repeated/benchstat。

## 12. Dedicated Benchmark Hardware

高价值 library/service 可以使用 dedicated host：

- fixed CPU；
- fixed OS；
- quiet system；
- fixed Go config。

只有 small regression 真有 business value 时才值得这项成本。

## 13. Baseline 类型

可以比较：

- PR vs main；
- current vs release；
- rolling median。

它们分别回答：

- 这个 PR 是否 regression？
- 长期是否 drift？
- gradual degradation 是否累积？

## 14. Repeated Samples

保存 raw repeated benchmark output，并用 statistical comparison。

不要 one sample vs one sample。

## 15. Alert vs Automatic Failure

Noisy metric 可以：

```text
detect suspicious regression
↓
alert/review
↓
rerun controlled
```

而不是每次小波动都 block merge。

## 16. Engineering Threshold

Threshold 应结合 variance 和 system value。

例如某 project 可以定义：

```text
<2% ignore
2–5% investigate
>5% block
```

但不能把这些数字复制成通用规则。

## 17. Allocation Threshold

Allocation metric 往往比 ns/op 稳定。

例如：

```text
0 alloc/op
→ 1 alloc/op
```

可能是很强 deterministic signal。

## 18. Binary Size

Inlining/PGO 可能增加 binary size。

如果 deployment/cold start/instruction cache 重要，应把 binary size 设为 guardrail。

## 19. Memory Regression

CPU 优化可能增加：

- live heap；
- RSS；
- allocation；
- retained capacity。

Memory regression 往往需要 longer-running benchmark。

## 20. Retention Test

可以设计：

```text
process large outlier
↓
return to normal workload
↓
measure heap/RSS
```

捕获 pool/cache poisoning。

## 21. Concurrency Scaling Regression

Synchronization optimization 要保护 scaling curve：

```text
GOMAXPROCS 1
2
4
8
16
```

Single-thread unchanged 不代表 multicore 没 regression。

## 22. False-Sharing Regression

Padding/layout 很容易在 struct refactor 中被无意破坏。

Independent-writer scaling benchmark 能检测。

Source comment 需要解释为什么 benchmark 会 fail。

## 23. P99 Regression

Throughput 不足以保护 user-facing performance。

Batching/queue 等 change 必须看：

- P50；
- P95；
- P99；
- errors；
- CPU。

## 24. Service Benchmark

Service regression workload 可以固定：

```text
request mix
concurrency
duration
```

比较：

```text
RPS
CPU/request
P99
alloc rate
RSS
```

## 25. Production Monitoring

有些 regression 只会在真实 traffic 出现。

长期 dashboard 应尽量用 normalized efficiency metric：

```text
CPU / 1k requests
memory / connection
GC CPU
P99 by endpoint
```

## 26. Canary Deployment

Canary 同时运行 control/candidate，更容易排除时间变化：

```text
same time
similar traffic
different binary
```

对大规模 deployment 很有价值。

## 27. Traffic Normalization

Raw CPU 随 traffic 增加而增加很正常。

更应该看：

```text
CPU/request
bytes allocated/request
RSS/connection
```

## 28. Workload Mix

即使 normalized metric，也会受到 endpoint/data mix 改变。

必要时按：

- endpoint；
- workload class；
- request size；
- region；

分组。

## 29. PGO Regression

PGO profile 会 stale。

应跟踪：

- profile age；
- source/build context；
- workload representativeness；
- occasional PGO on/off comparison。

## 30. Go Version Upgrade

升级 toolchain 是天然 revalidation point。

重新运行：

- compiler proof；
- benchmark；
- PGO；
- unsafe/runtime test；
- system load。

旧 workaround 可能已经不必要。

## 31. 分离 Toolchain Effect 与 Source Effect

不要拿：

```text
Go 1.26 old source
```

直接和：

```text
Go 1.27 new source
```

比较并把差异全部归因于 source。

应该单独测 toolchain effect。

## 32. Hardware Upgrade

新 CPU 会改变：

- cache；
- branch predictor；
- atomic scaling；
- bandwidth。

Hardware-sensitive proof 应重新跑。

## 33. Dependency Upgrade

Dependency 也会带来 regression/improvement：

- JSON；
- DB driver；
- compression；
- crypto。

Release performance suite 应覆盖这种 drift。

## 34. Benchmark Drift

Benchmark 自己也会过时。

定期问：

- payload 还真实吗？
- concurrency 还真实吗？
- old feature 已删除吗？
- new hot path 被覆盖了吗？

Stale benchmark 会制造 false confidence。

## 35. Golden Absolute Number

避免：

```text
BenchmarkX must always <42ns
```

不同 CPU/Go version 会让 absolute number 失效。

优先 relative comparison in controlled environment。

## 36. Relative Contract

Specialized library 可以保护：

```text
optimized path should remain better than baseline
```

这种 relative contract 往往比固定 ns/op 更 portable。

仍需 noise-aware comparison。

## 37. Performance Budget

可以明确：

```text
parser allocations ≤ 1/op
steady RSS ≤ 2 GiB under W
P99 ≤ 50 ms at load X
CPU/request ≤ baseline × 1.05
```

这把 performance 变成 explicit non-functional requirement。

## 38. Budget Ownership

每个 budget 应绑定：

- workload definition；
- measurement method；
- owner/component；
- escalation policy。

否则 threshold 很快失去意义。

## 39. Regression Triage

发现 regression：

```text
1. reproduce
2. verify environment
3. identify first bad change
4. profile
5. build cost model
6. fix or intentionally accept
```

不要第一反应就是调高 threshold。

## 40. Bisecting

稳定 benchmark 可以配合 `git bisect` 找 first bad commit。

Signal 大、benchmark 快、环境稳定时效果最好。

## 41. Accepted Regression

有时为了：

- correctness；
- security；
- semantics；
- maintainability；
- new feature；

接受性能下降是正确决策。

应该更新 baseline 并记录原因。

## 42. Regression Comment

如果 source shape 必须保留，应在代码里说明与 benchmark 的关系。

例如：

```go
// Keep this padding; BenchmarkShardWrites detects false sharing if the
// independently written counters share a cache line.
```

## 43. Unsafe Regression Strategy

Unsafe change 必须同时保护：

```text
performance
+
correctness
```

例如：

- benchmark；
- race；
- checkptr；
- lifetime test；
- architecture CI。

## 44. Lock-Free Regression Strategy

除了 throughput，还要保护：

- invariant；
- stress；
- race；
- ABA scenario；
- scaling；
- retry。

## 45. mmap Regression Strategy

测试：

- unmap lifecycle；
- concurrent reader；
- file error/truncation；
- cold/warm performance；
- RSS/page behavior。

## 46. cgo Regression Strategy

记录：

- Go version；
- C compiler；
- native library version。

测试 pointer lifetime、callback、pinning/handle、boundary benchmark、ABI。

## 47. Runtime-Private Dependency

如果确实依赖 private runtime：

- supported Go version CI；
- release/RC testing；
- build checks；
- behavior tests。

这是选择 private dependency 的维护成本。

## 48. Alert Fatigue

一个每天误报 1% regression 的系统最后会被所有人忽略。

优先：

```text
few high-confidence alerts
```

而不是大量低质量 warning。

## 49. Historical Data

长期 benchmark trend 能发现：

- gradual drift；
- toolchain effect；
- dependency regression；
- lost optimization。

PR comparison 看不到这些。

## 50. Reproducibility Metadata

保存：

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

## 51. Guard 也需要退休

Hot path 删除、compiler improved、workload changed、objective changed 时，应更新或删除 obsolete guard。

Regression infrastructure 也需要维护。

## 52. 按 Optimization Type 选择 Guard

### Allocation
- microbenchmark；
- alloc contract；
- GC validation。

### Synchronization
- scaling；
- mutex/block profile；
- P99。

### Compiler workaround
- compiler proof；
- microbenchmark；
- toolchain revalidation。

### Unsafe zero-copy
- benchmark；
- retention；
- race/checkptr/lifetime tests。

### GC tuning
- load test；
- CPU/RSS/P99；
- production monitoring。

## 53. Regression Strategy Template

### Property
什么必须保持？

### Workload
什么 input/concurrency？

### Metric
测什么？

### Baseline
与谁比较？

### Noise Model
波动多大？

### Trigger
什么程度 alert/fail？

### Revalidation
什么时候重新考虑 guard？

## 54. 官方资料

- `testing`: https://pkg.go.dev/testing
- `benchstat`: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat
- Diagnostics: https://go.dev/doc/diagnostics

## 55. 工程视角

优化的最终步骤不是：

> Benchmark 变快了。

而是：

> 这个收益被与其 mechanism/noise 相匹配的 regression guard 保护，并且 future maintainer 知道为什么这个 guard 存在。

这才是 durable performance engineering。
