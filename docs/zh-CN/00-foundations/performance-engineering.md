# 性能工程

[English](../../00-foundations/performance-engineering.md) | 简体中文

## 1. 性能工程不是技巧合集

性能优化经常被描述成若干局部技巧：

- 少一次 allocation；
- 去掉一个 branch；
- 加一个 pool；
- 改成 atomic；
- 避免一次 copy。

这些技术都可能有价值，但它们本身并不等于 Performance Engineering。

性能工程关注的是：

> 系统把资源花在哪里、为什么会产生这些成本，以及在现实约束下哪一种权衡最值得。

因为很多所谓“优化”只是把成本移动到另一个维度。

例如 zero-copy：

```text
少 memcpy
少 allocation
   ↓
更复杂的 ownership
更严格的 lifetime
可能产生 retention
```

lock-free 也是一样：

```text
减少 blocking
   ↓
增加 retry
增加 cache-line contention
增加 correctness complexity
```

只有当总系统权衡变得更好时，修改才有工程意义。

## 2. 性能是系统属性

Go 源码不会直接以原样运行在 CPU 上。

最终行为来自多个层次共同作用：

```text
Application behavior
        ↓
Architecture and data model
        ↓
Algorithms and data structures
        ↓
Go compiler
        ↓
Go runtime
        ↓
Operating system
        ↓
CPU and memory hardware
```

一个层面出现的热点，根因可能来自另一个层面。

例如 CPU profile 中：

```text
runtime.scanobject
runtime.gcBgMarkWorker
```

很热，不等于“Go GC 太慢”。

真实原因可能是：

- live heap 变大；
- allocation rate 太高；
- data structure pointer density 太高；
- GOMEMLIMIT 太紧导致 collector 更激进。

Runtime function 是成本出现的位置，不一定是问题产生的位置。

类似地，如果 `atomic.Add` 很热，根因可能不是 atomic implementation，而是：

```text
many cores
   ↓
one shared counter
   ↓
one cache line
   ↓
coherence traffic
```

真正的优化可能是 sharding 或 local accumulation。

## 3. 优化之前先定义“性能”

“让它更快”不是完整目标。

性能可能指：

- 更高 throughput；
- 更低 median latency；
- 更低 P99/P99.9；
- 更低 CPU/request；
- 更低 RSS；
- 更低 allocation rate；
- 更好的 multicore scalability；
- 更低基础设施成本。

这些目标之间可能冲突。

例如：

```text
larger cache
  ↓
higher hit rate
  ↓
lower CPU / latency
  ↓
higher RSS
```

或者：

```text
larger GOGC
  ↓
fewer GC cycles
  ↓
lower GC CPU
  ↓
higher heap footprint
```

所以每次优化都必须先知道最重要的 objective 和 guardrails。

## 4. Throughput、Latency 与 Efficiency

### Throughput

Throughput 表示单位时间完成多少工作，例如：

```text
requests / second
operations / second
MB / second
```

真正的上限通常由最先饱和的资源决定，例如 CPU、memory bandwidth、lock、database 或 network。

### Latency

Latency 表示一次操作需要多久。

尾延迟尤其重要：

```text
P50   = 5 ms
P99   = 40 ms
P99.9 = 200 ms
```

平均值无法解释极少数昂贵路径。

### Efficiency

Efficiency 关心完成一单位有效工作需要多少资源，例如：

```text
CPU seconds / request
bytes allocated / operation
memory / connection
```

如果 throughput 增加 10%，但 CPU 翻倍，这不一定是更好的系统。

## 5. Bottleneck 依赖 Workload

Bottleneck 不是某个函数与生俱来的属性。

它是代码与 workload 的关系。

同一个：

```go
mu.Lock()
critical()
mu.Unlock()
```

在一个 worker 下可能几乎没有 contention。

在 64 个 worker 下可能成为主瓶颈。

branch prediction、cache、allocation、GC、channel buffer、memory bandwidth 都有类似特性。

因此性能结论必须包含 workload 条件。

## 6. 优化是在重新分配资源

大多数优化都在不同成本维度之间做交换。

### CPU ↔ Memory

cache、pool、更高 GOGC 通常通过消耗更多 memory 来减少 CPU。

### Memory ↔ Complexity

compact layout 可能减少 memory，却增加 index/representation complexity。

### Throughput ↔ Tail Latency

batching 可能提高吞吐，却增加单个请求的等待时间。

### Safety ↔ Control

unsafe zero-copy 可以去掉 copy，但 lifetime correctness 从语言转移给程序员。

### Simplicity ↔ Specialized Performance

generic API 更好维护，specialized representation 可能在某个 hot path 更快。

正确选择取决于系统优先级。

## 7. 工程中的 Amdahl's Law

一个优化最多只能节省它影响的那部分成本。

如果某函数只占总 CPU 的 1%，即使让它无限快，系统 CPU 也最多下降约 1%。

这就是为什么 profile-guided 工作非常重要。

一个 microbenchmark 的 10× 改善，在服务层可能完全无意义。

做复杂优化之前，应该先估算：

```text
hotness
×
possible speedup
=
maximum system gain
```

## 8. Local Improvement 与 System Improvement

Local benchmark 回答：

> 在这个 test 中，实现 B 是否更快？

System benchmark 回答：

> 这个修改是否真的改善了应用？

例如 unsafe bytes→string 在 microbenchmark 中可能快很多，但如果转换只占 request CPU 的 0.2%，系统收益仍然可以忽略。

局部结果可以是正确的，而工程决策仍然是“不采用”。

## 9. Evidence-Driven Engineering

成熟的性能调查需要完整证据链：

```text
Symptom
  ↓
Measurement
  ↓
Cost model
  ↓
Hypothesis
  ↓
Candidate change
  ↓
A/B validation
  ↓
System validation
```

跳过不同环节会产生不同错误：

- 没有 measurement：可能优化非 bottleneck；
- 没有 cost model：不知道为什么变快；
- 没有 A/B：无法区分 signal 与 noise；
- 没有 system validation：可能局部更快、整体更差。

## 10. 性能工程与可维护性

优化后的代码经常看起来“不自然”。

例如：

```go
_ = b[7]
```

可能只是为了给 compiler 提供 dominating bounds proof。

或者：

```go
out := append([]byte(nil), input[:n]...)
```

故意 copy，是为了不让一个小结果保留巨大的 backing array。

padding 也可能只是为了拆开 cache line。

这些代码很容易被下一位维护者当作“冗余”清理掉。

因此一个优化只有在它的理由可以被恢复时才算完成。

重要优化应该通过：

- comment；
- benchmark；
- test；
- proof；
- design documentation；

保存意图。

## 11. 性能工程是反馈循环

系统会持续变化：

- traffic 变化；
- Go version 变化；
- CPU 变化；
- dataset 增长；
- architecture 变化。

健康的过程是循环的：

```text
observe
 ↓
measure
 ↓
understand
 ↓
change
 ↓
validate
 ↓
monitor
 ↓
repeat
```

目标不是写出永久“优化完成”的代码。

目标是持续维护一个成本可理解、可控制的系统。
