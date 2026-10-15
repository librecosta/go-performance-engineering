# Optimization Review

[English](../../06-methodology/optimization-review.md) | 简体中文

## 1. 为什么 Performance Change 需要额外 Review 视角

普通 code review 关注：

- correctness；
- readability；
- maintainability；
- API。

Performance change 还需要额外问：

- problem 是否 measured？
- mechanism 是否针对 problem？
- benefit 是否足够大？
- cost 是否只是移动到别处？
- future maintainer 能否恢复优化理由？

一个 change 可以更快，但仍然是错误工程决策。

## 2. 先 Review Problem，再 Review Solution

弱：

> Replace Mutex with atomic for speed.

强：

> 32-way load 下，这个 global metrics lock 占 19% blocked time，throughput 在 8 workers 后停止 scaling。

第二种 statement 才能被验证。

## 3. Define Objective

先定义 primary objective：

```text
CPU/request
throughput
P99
RSS
allocation rate
GC CPU
```

再定义 guardrails。

例如：

```text
Goal:
CPU/request -5%

Guardrails:
P99 regression <2%
RSS regression <5%
no runtime-private dependency
```

## 4. Baseline

Performance review 必须有 baseline：

- current main；
- current production；
- previous implementation；
- idiomatic safe implementation。

没有 baseline，“faster”没有定义。

## 5. Bottleneck Evidence

Change 应针对 measured bottleneck：

- CPU profile；
- heap/alloc；
- mutex；
- trace；
- metrics；
- service benchmark。

Source smell 不能支撑 high-complexity optimization。

## 6. Cost Model

Author 应解释预期 mechanism。

例如：

```text
one shared atomic counter
↓
cache-line ownership transfer
↓
poor scaling

candidate:
sharded counters
↓
independent writes
↓
aggregate on read
```

这样 reviewer 可以挑战 mechanism，而不是争论 style。

## 7. Candidate Simplicity

复杂技术前先问有没有简单方案。

例如：

```text
custom lock-free queue
vs
batch + channel

unsafe zero-copy
vs
append-style API

runtime procPin
vs
explicit sharding

sync.Pool
vs
stack/local scratch
```

优先最低风险、足以达成 objective 的机制。

## 8. Remove Work First

在“加速操作”之前先问是否能删除它。

例如：

```text
make global atomic increment faster
```

之前先问：

```text
是否真的需要每个 event 都更新 global counter？
```

Architecture 经常比 micro-tuning 更强。

## 9. A/B Evidence

Performance PR 通常应该给 representative before/after。

Local change：

```text
old
new
benchstat
```

大系统 change：

```text
load test / canary
```

微小 difference 不应该只给一次 run。

## 10. Statistical vs Practical Gain

Statistically significant 的 0.3% 可能完全不值得引入 unsafe/assembly/extra abstraction。

Engineering significance 必须单独判断。

## 11. Amdahl Bound

结合 profile hotness 估计 total ceiling。

例如 target 只占 4% CPU，local speedup 25%，系统理论上限约 1%。

Complexity 很高时，应尽早停止。

## 12. Correctness First

Performance optimization 不能偷偷改变 semantics。

高风险领域：

- lock-free；
- unsafe aliasing；
- cgo lifetime；
- mmap lifetime；
- runtime-private。

Benchmark 不能证明 correctness。

## 13. Maintainability 是 Hard Gate

一个无法安全维护的 optimization 不算完成。

本项目把 maintainability 当硬条件。

## 14. 哪些 Optimization 需要 Comment

通常包括：

- `_ = b[7]`；
- cache-line padding；
- unusual field order；
- intentional retention-breaking copy；
- pool capacity cutoff；
- pointer→index；
- AoS→SoA；
- hot/cold split；
- unusual sharding/batching；
- CAS/backoff；
- lock-free；
- unsafe zero-copy；
- mmap lifetime；
- `runtime.KeepAlive`；
- cgo directives；
- assembly；
- runtime-private dependency。

## 15. 哪些通常不需要 Special Comment

例如：

```go
items := make([]Item, 0, len(src))
```

这种 idiomatic preallocation 的意图本身已清晰。

不要制造：

```go
// Preallocate for performance.
```

这种没有新信息的噪声。

## 16. Performance Comment Structure

Non-obvious comment 应覆盖相关部分：

### WHY

为什么这种 unusual form 是 intentional？

### MECHANISM

减少什么 measured cost？

### PRESERVATION

哪种看似更简单的改法会重新引入 regression？

### SAFETY CONTRACT

Unsafe/FFI/lifetime code 的 ownership/lifetime 为什么正确？

## 17. BCE Example

```go
// Keep this bounds check: it proves len(b) >= 8 to the compiler,
// allowing the fixed indexed reads below to eliminate redundant checks.
_ = b[7]
```

这类 comment 保存真正理由。

## 18. Retention Example

```go
// Copy the small result instead of retaining the full input buffer.
// Returning input[:n] here may keep a multi-megabyte backing array alive.
out := append([]byte(nil), input[:n]...)
```

它解释为什么“多一次 copy”反而是优化。

## 19. False-Sharing Example

```go
type shard struct {
    counter atomic.Uint64

    // Keep independently written shard counters on separate cache lines.
    // Removing this padding can reintroduce false sharing under contention.
    _ cpu.CacheLinePad
}
```

它保存 source logic 看不见的 hardware invariant。

## 20. Unsafe Example

```go
// bytesToString returns a zero-copy view of b.
// The backing bytes must remain immutable and must not be reused while
// the returned string is reachable.
```

Unsafe comment 必须解释 correctness/lifetime，不只是“更快”。

## 21. Benchmark 也是 Documentation

命名清晰的 benchmark：

```text
BenchmarkFalseSharing
BenchmarkShardedCounter
BenchmarkRetentionCopy
```

可以长期保存 optimization mechanism。

## 22. Proof Link

高度 non-obvious change 可以链接 `proofs/`，把详细 evidence 从 local comment 分离出去。

## 23. Unsafe Escalation Model

风险逐层升级：

```text
safe Go
↓
public unsafe API
↓
OS / FFI
↓
compiler directive / assembly
↓
runtime-private
```

不要跳级。

## 24. Safe Go First

Unsafe zero-copy 前先问：

- caller-provided destination？
- batching？
- representation redesign？
- copy really hot？

Safe redesign 经常已经能拿到大部分收益。

## 25. Public Unsafe API

Public `unsafe` 有 documented contract。

仍然危险，但比 private runtime hack 更可控。

## 26. OS / FFI Boundary

mmap/cgo review 必须回答：

- who allocates？
- who releases？
- pointer escapes？
- view outlives storage？

## 27. Compiler Contract

noescape 等 directive 是 correctness assertion。

需要：

- implementation proof；
- tests；
- comment；
- toolchain compatibility plan。

## 28. Runtime-Private Red Zone

`//go:linkname` private runtime symbol 应非常 exceptional。

Review 必须问：

- public alternative？
- measured gain？
- supported Go versions？
- upgrade detection？
- fallback？

## 29. Lock-Free Review

Custom lock-free 至少要审：

- invariant；
- linearization point；
- progress guarantee；
- ABA；
- lifetime；
- retry；
- contention scaling；
- race/stress test。

“用了 atomic”不是 correctness proof。

## 30. Atomic Review

问：

- state 是否真的 single-word？
- compound invariant 是否被拆坏？
- write frequency？
- true sharing？
- ownership/sharding 是否更简单？

## 31. Mutex Review

不要因为出现 Mutex 就拒绝。

问：

- contended？
- critical section duration？
- work 能否移出去？
- state 能否 partition？

Short uncontended Mutex 经常是非常好的工程方案。

## 32. RWMutex Review

Hot code 中使用 RWMutex 应有 workload evidence：

- read duration；
- writer frequency；
- core count；
- Mutex baseline。

“Mostly reads”不够。

## 33. Pool Review

问：

- churn measured？
- reset complete？
- giant capacity poisoning？
- old pointers cleared？
- temporary object？
- RSS 真的改善了吗？

## 34. Data Layout Review

问：

- hot fields？
- access pattern？
- object count？
- cache/TLB？
- pointer density？
- readability cost？

不要为 rarely allocated config struct 做理论 packing。

## 35. PGO Review

如果依赖 PGO：

- profile origin；
- representativeness；
- PGO on/off comparison；
- build reproducibility。

如果 compiler 已能自动 specialization，不要无证据手工破坏 abstraction。

## 36. Compiler-Sensitive Review

Escape/BCE/inline change：

- record Go version；
- diagnostics；
- benchmark；
- avoid hard-coded heuristic。

Workaround 应该在 compiler 进步后可删除。

## 37. Version Sensitivity

明确标注依赖：

- allocator threshold；
- GC implementation；
- inline heuristic；
- private runtime layout；
- architecture-specific code。

并设置 revalidation trigger。

## 38. Portability

amd64 +10% 不保证 arm64 也有。

Multi-arch project 应考虑：

- generated code；
- benchmark；
- fallback；
- correctness。

## 39. Memory Guardrail

CPU optimization 可能花 memory。

同时看：

```text
CPU ↓
RSS ?
live heap ?
retention ?
```

## 40. Latency Guardrail

Throughput optimization 可能伤 P99：

- batching；
- queue；
- spin；
- larger GC target。

User-facing system 必须看 tail latency。

## 41. CPU Guardrail

Lock-free 可能 throughput 更高，却烧更多 CPU。

需要看 CPU/request，而不是只看 ops/s。

## 42. Complexity Budget

每个 optimization 都会消耗 complexity budget。

规则：

```text
complexity must be proportional to measured value
```

不要为了微小收益把普通 service 变成 runtime research project。

## 43. Reversibility

Evidence 仍不确定时，优先可逆设计：

- feature flag；
- isolated implementation；
- fallback；
- build option。

## 44. Failure Mode

Review assumption 失效时会发生什么：

```text
pool sees 100 MiB buffer
shard count insufficient
input distribution changes
PGO profile stale
mmap closes early
```

Optimization robustness 也要设计。

## 45. Observability

System-level change merge 后，应仍能持续观察相关 metric。

例如改 GC tuning，就要继续监控 GC CPU/live heap/RSS。

## 46. Regression Guard

Merge 前决定如何长期保护收益：

- benchmark；
- allocation assertion；
- load test；
- CI comparison；
- canary；
- proof。

## 47. Stop Condition

Review 可以得出：

> Do not optimize.

合理原因：

- bottleneck 太小；
- gain below noise；
- complexity too high；
- system neutral；
- guardrail regression；
- evidence weak。

## 48. 推荐 Review Narrative

```text
Problem
↓
Baseline evidence
↓
Cost model
↓
Candidate
↓
A/B result
↓
Trade-offs
↓
Maintainability
↓
Regression guard
```

## 49. Human Review Template

### Goal
什么 system metric 要改善？

### Baseline
当前 evidence 是什么？

### Cost Model
为什么当前 implementation 花这个成本？

### Candidate
改变什么 mechanism？

### Evidence
A/B 支持什么结论？

### Guardrails
什么不能 regression？

### Maintainability
什么 non-obvious invariant 必须保存？

### Version Sensitivity
依赖 compiler/runtime/hardware 吗？

### Regression Strategy
未来怎么发现收益丢失？

## 50. Decision Categories

可以明确分类：

```text
ACCEPT
ACCEPT WITH GUARD
EXPERIMENTAL
REJECT
NO OPTIMIZATION NEEDED
```

比“快/慢”二元判断更有工程信息。

## 51. 工程视角

Optimization review 的目的不是让性能代码更难 merge。

而是保证 speed improvement 同时：

- correct；
- measurable；
- understandable；
- maintainable；
- reproducible。

只有下一位工程师能够因为正确理由保留或删除它，这个优化才真正完成。
