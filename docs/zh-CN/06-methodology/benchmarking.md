# Benchmarking

[English](../../06-methodology/benchmarking.md) | 简体中文

## 1. Benchmark 能证明什么

Benchmark 在一个明确定义的 workload 下比较实现。

它回答：

> 在这些条件下，哪个 candidate 在这些 metric 上更好？

它不自动证明：

- universal superiority；
- production impact；
- bottleneck root cause。

Benchmark 的证据强度取决于 workload 是否真正代表工程问题。

## 2. Benchmark vs Profile

```text
profile
→ choose what to investigate

benchmark
→ compare proposed implementations
```

没有先证明 hotness，就围着 benchmark 优化，容易变成 benchmark-driven development。

## 3. 先定义 Metric

常见：

- time/op；
- B/op；
- allocs/op；
- throughput；
- custom metric。

Claim 必须匹配 metric。

例如：

```text
"reduce allocation churn"
→ B/op + allocs/op

"scale better under contention"
→ throughput across concurrency levels

"reduce parser CPU"
→ ns/op + component CPU validation
```

## 4. `testing.B`

Go 标准 benchmark harness 来自 `testing.B`。

新 benchmark 通常推荐：

```go
func BenchmarkEncode(b *testing.B) {
    input := makeInput()

    for b.Loop() {
        Encode(input)
    }
}
```

## 5. `B.Loop`

`B.Loop` 让 setup/cleanup 与 timed body 的边界更清晰，也避免传统 `b.N` calibration 反复重新执行整个 benchmark function。

新 benchmark 应优先采用。

## 6. B.Loop 与 Dead-Code Elimination

Modern testing 会对 `for b.Loop()` body 做一些特殊处理，降低 compiler 把被测 body 完全删除的风险。

但它不能修复 unrealistic workload。

Benchmark correctness 仍然由作者负责。

## 7. Loop Form

应使用标准：

```go
for b.Loop() {
    ...
}
```

不要为了抽象把 loop condition 包得很奇怪，然后假设 compiler/testing special behavior 完全一样。

## 8. Legacy `b.N`

旧写法：

```go
func BenchmarkEncode(b *testing.B) {
    input := makeInput()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        Encode(input)
    }
}
```

仍然可用。

已有正确 benchmark 不必机械重写。

## 9. Setup Cost

如果只想 measure lookup：

```go
func BenchmarkLookup(b *testing.B) {
    index := buildIndex()

    for b.Loop() {
        lookup(index, key)
    }
}
```

Setup 不应混入 timed region。

## 10. Setup 什么时候应该算进去

如果 production 每次 operation 都重建 state，那么 setup 就属于真实成本。

Benchmark lifecycle 必须匹配 real lifecycle，而不是为了数字好看随意排除工作。

## 11. Input Reuse

一直用相同 input：

```go
for b.Loop() {
    Parse(sameInput)
}
```

可能让：

- cache；
- branch predictor；
- allocation；
- mutation state；

变得不真实。

如果你就是要测 hot-cache case，应明确写出来。

## 12. Input Distribution

真实 benchmark 可能需要覆盖：

- small/medium/large；
- valid/invalid；
- sorted/random；
- hit/miss ratio；
- receiver mix；
- payload distribution。

单一 synthetic case 很容易隐藏 trade-off。

## 13. Branch Distribution

如果 benchmark 永远走同一 branch，而 production 接近随机，branch predictor 行为会不同。

Data distribution 是 benchmark correctness 的一部分。

## 14. Cache Warmth

小 dataset 反复执行可能整个 fit cache。

Production working set 可能大得多。

必要时分别建立：

```text
hot-cache benchmark
large/cold-working-set benchmark
```

## 15. Mutation

如果 operation 修改 input，后续 iteration 可能测到不同 state。

需要根据真实 lifecycle reset。

Reset cost 是算进去还是排除，也取决于 production semantics。

## 16. Allocation Metrics

使用：

```text
B/op
allocs/op
```

验证 allocation claim。

但 0 alloc/op 不是最终目标。

Time、retention、complexity 仍然重要。

## 17. `ReportAllocs`

必要时可以 `b.ReportAllocs()`。

更重要的是 repository 使用一致 command，保证 A/B 可比较。

## 18. Custom Metrics

有些 cost model 可以通过：

- MB/s；
- items/s；
- retries/op；

表达得更直接。

例如 CAS benchmark 加 `retries/op`，能解释 throughput collapse 的 mechanism。

## 19. `SetBytes`

Byte-processing benchmark 可以设置 bytes/op，让输出包含 MB/s 类 throughput。

适合：

- codec；
- hash；
- parser；
- copy；
- compression。

## 20. Microbenchmark

Microbenchmark 适合 isolate one mechanism：

- BCE；
- allocation shape；
- false sharing；
- atomic primitive。

优点是因果清楚，缺点是代表性有限。

## 21. Component Benchmark

Component benchmark 加入更多 realistic context，例如：

- parser + allocator；
- cache + synchronization；
- request processing without network。

它减少 isolation，但提高 representativeness。

## 22. End-to-End Benchmark

系统 benchmark 测用户真正看到的：

- RPS；
- P99；
- CPU/request；
- RSS。

它是大改动的最终验证层，但不适合发现 microscopic mechanism。

## 23. Benchmark Pyramid

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

每层回答不同问题。

## 24. Repeated Runs

小差异必须 repeated sampling。

例如：

```bash
go test -run='^$' -bench='BenchmarkX$' -count=10 ./pkg > old.txt
```

Candidate 同样执行。

## 25. `benchstat`

使用：

```bash
benchstat old.txt new.txt
```

比较 result distribution、effect size 与 statistical confidence。

比手工看一组 ns/op 更可靠。

## 26. Statistical Significance

Statistically significant 只说明 sample evidence 支持“distribution 不同”。

不代表工程收益大。

## 27. Engineering Significance

例如：

```text
-0.5% microbenchmark
+ unsafe aliasing
```

即使 statistically significant，也可能完全不值得。

System value 与 complexity/risk 必须一起考虑。

## 28. Confidence / Variance

Benchmark 不是精确常数。

Noise 来自：

- scheduling；
- frequency scaling；
- cache state；
- interrupt；
- background process。

要理解 distribution，而不是相信单次数字。

## 29. Environment Control

至少记录：

- CPU；
- OS/kernel；
- Go version；
- GOARCH；
- GOMAXPROCS；
- command；
- env；
- PGO state。

## 30. Go Version

Compiler/runtime change 会让同源码 benchmark 漂移。

Candidate A/B 应该在同 toolchain 比较。

## 31. CPU Frequency

Modern CPU 会根据 power/temperature/core usage 调频。

对小差异 benchmark，稳定 power setting 可以降低 noise。

但普通大幅收益不需要实验室级 tuning。

## 32. Thermal Effect

长 benchmark 会让 CPU 升温降频。

如果总是 old 先跑、new 后跑，会产生 ordering bias。

Repeated/interleaved run 更可靠。

## 33. Background Activity

Browser、IDE、antivirus、CI noisy neighbor 都会增加 variance。

Shared cloud runner 不适合判断 1% regression。

## 34. GOMAXPROCS

Concurrency benchmark 必须记录 `GOMAXPROCS`。

它会改变：

- scheduler；
- contention；
- cache coherence；
- GC worker behavior。

## 35. CPU Affinity

高级 hardware experiment 可以固定 CPU 降低 migration noise。

但也可能不代表 production。

只有 hypothesis 需要时才用。

## 36. GC State

不要为了“数字好看”关 GC，除非实验明确研究 allocator CPU independent of GC。

Production validation 应用 production-like GC config。

## 37. Race Detector

Race build 是 correctness test，不是 benchmark baseline。

## 38. PGO

如果 production 用 PGO，system benchmark 应包含相同 build mode。

Compiler experiment 也可以专门 A/B PGO on/off。

## 39. Compiler Flags

`-N` / `-l` 会大幅改变优化。

它们适合 diagnostic experiment，不是 normal benchmark baseline。

## 40. Concurrency Benchmark

Synchronization change 应测 scaling curve，而不是只测一个并发度：

```text
1
2
4
8
16
32
```

## 41. Contention Scenario

至少区分：

```text
uncontended
low contention
high contention
```

很多 primitive 在单线程和高 contention 下排名会反转。

## 42. Read/Write Ratio

Mutex vs RWMutex 应同时改变：

- read ratio；
- read duration；
- write duration；
- core count。

“90% reads”不定义完整 workload。

## 43. Sharding Benchmark

Sharding 同时改变 synchronization 与 memory layout。

要测：

- write throughput；
- read aggregation；
- memory overhead；
- shard count。

Shard 太多也会伤 cache/memory。

## 44. Lock-Free Benchmark

不要只看 throughput。

还看：

- retries/op；
- CPU；
- P99；
- scaling；
- starvation。

## 45. Memory Benchmark

Microbenchmark `allocs/op` 看不到 long-lived capacity retention。

Retention 需要更长 component/system scenario。

## 46. Zero-Copy Benchmark

需要覆盖 realistic payload/lifetime，并额外看：

- retention；
- ownership；
- downstream cache behavior。

Conversion benchmark 本身不够。

## 47. Compiler Optimization Benchmark

Compiler proof 最好组合：

```text
diagnostic
+
assembly
+
benchmark
```

Diagnostic 直接证明 mechanism，timing 证明 local cost。

## 48. Avoid Constant Folding Accident

Production input 是 runtime data 时，benchmark 不要全用 compile-time constant，避免 compiler 做不现实 simplification。

## 49. Avoid Setup Accident

Legacy benchmark 要正确 timer control。

B.Loop 改善了 setup boundary，但作者仍要把 lifecycle 写对。

## 50. Avoid Logging in Timed Body

Logging 会轻易 dominate 微小 operation。

除非就是 benchmark logging，否则不要放 timed body。

## 51. Avoid Shared Global Contamination

不同 benchmark 不应无意共享 mutable global state。

如果 global cache 是 system semantics，应明确 warm/cold policy。

## 52. Parallel Benchmark 不是 Production

`RunParallel` 很有用，但不会自动复制：

- arrival distribution；
- think time；
- key skew；
- backpressure。

Concurrency workload 需要人工建模。

## 53. Key Distribution

Cache/map/lock 对 key skew 非常敏感。

Uniform random 与 Zipf-like hot-key distribution 可能产生完全不同 contention。

应尽量匹配 production。

## 54. Latency Distribution

`testing.B` 主要提供 aggregate metrics。

Service P99 需要专门 load-testing harness，不能从 ns/op 推导。

## 55. Warmup

Go 不是典型 JIT runtime，但 system 仍有 warmup：

- cache；
- DNS；
- connection pool；
- page cache；
- lazy init。

End-to-end benchmark 应定义 warmup。

## 56. Baseline

Baseline 应该是：

- current main；
- current production；
- reasonable reference implementation。

不要构造刻意很差的 strawman。

## 57. One Variable at a Time

Mechanism proof 尽量只改一个关键因素，降低 confounder。

如果真实工程 change 很复杂，可以分别 microbench mechanism，再 component benchmark final design。

## 58. Multi-Change Optimization

复杂 transformation 可以：

```text
microbench individual mechanisms
+
component benchmark integrated design
```

同时得到机制与整体效果。

## 59. Benchmark Artifact

建议保存：

- source；
- command；
- input generator；
- environment metadata；
- raw result。

这样性能结论可复现。

## 60. Proof vs Product Benchmark

`proofs/` 可以用 synthetic workload isolate mechanism。

Product benchmark 则必须优先 realistic behavior。

二者都正确，只要 scope 明确。

## 61. Stop Conditions

停止 tuning 的条件：

- bottleneck 已解决；
- gains 低于 meaningful threshold；
- complexity budget 用完；
- system benefit 不再增长；
- 下一 bottleneck 已转移。

## 62. 官方资料

- `testing`: https://pkg.go.dev/testing
- `benchstat`: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat
- Diagnostics: https://go.dev/doc/diagnostics

## 63. 工程视角

好的 benchmark 不是 variance 最小、百分比最大。

而是：

> 它能够用可复现方式，尽可能直接回答当前工程问题。
