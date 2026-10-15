# 优化原则

[English](../../00-foundations/optimization-principles.md) | 简体中文

## 1. Measure Before Optimize

代码形态不是 bottleneck 证据。

allocation、mutex、interface、copy、branch、bounds check、atomic、pointer traversal 都只能作为候选调查点。

真正的优化从 measurement 开始。

## 2. 先删除工作，再让工作更快

最便宜的工作是根本没有发生的工作。

Compiler optimization 就体现了这个原则：

```text
heap allocation
→ stack allocation

bounds check
→ eliminated

function call
→ inline

constant branch
→ removed
```

Application design 也可以做到同样的事：

```text
shared write
→ local accumulation

many lock operations
→ one batch

many heap nodes
→ one flat backing array
```

在加速一项操作之前，先问它能不能被删除。

## 3. 优化 Bottleneck，而不是“代码味道”

源码里出现 Mutex 不等于 lock contention。

interface 不等于 dispatch bottleneck。

allocation 不等于 GC 问题。

把 performance smell 当作 investigation prompt，而不是 conclusion。

## 4. 优先 Representation Change，而不是 Micro-Trick

很多真正大的收益来自 representation 改变：

```text
pointer graph
→ index-based storage

large mixed struct
→ hot/cold split

single shared counter
→ sharded counters

temporary object tree
→ slab/arena-like storage
```

这些修改在消除根本成本，而不是围绕它省几条 instruction。

## 5. 让 Compiler 更容易证明

Compiler 只能删除它能证明为安全的工作。

源码结构可以暴露有用事实：

- length guard；
- clear control flow；
- concrete type；
- simple hot helper；
- predictable ownership。

不要因为 compiler 暂时没优化成功，就直接跳到 unsafe。

先看能否用 safe Go 表达同样的 invariant。

## 6. 把 Data Layout 当作算法决策

Data layout 会影响：

- cache；
- TLB；
- bandwidth；
- pointer chasing；
- GC scan work。

对于数量巨大且处于 hot path 的结构，layout 可能与 algorithm complexity 同样重要。

## 7. 优化 Synchronization 之前先减少 Sharing

高 contention 经常说明 shared-state design 本身需要改变。

例如：

```text
global counter
→ local counters + aggregation

global map
→ shards

shared mutable state
→ immutable snapshot

multi-writer state
→ single writer
```

Mutex 换成 atomic 并不会删除 true sharing。

## 8. 有意识地用 Memory 换 CPU

很多性能策略会花 memory：

- cache；
- pool；
- larger GC target；
- preallocation。

这可以完全正确。

但 memory 必须被视为 budget，而不是免费资源。

至少同时观察：

```text
CPU saved
memory retained
```

## 9. 不要崇拜 Zero Allocation

Zero allocation 在一些 hot path 中很有价值，但不是终极指标。

一个 1 alloc/op 的实现可能更快、更简单、更安全。

有时故意 copy 反而能释放巨大 backing array。

目标是 total system cost。

## 10. 不要崇拜 Lock-Free

Lock-free 描述 progress guarantee，不是 speed guarantee。

在 contention 下，它可能制造：

- retry storm；
- cache-line traffic；
- starvation；
- higher CPU。

一个设计良好的 Mutex 可能比 naive CAS loop 更快。

## 11. 不要崇拜 Zero-Copy

Zero-copy 减少数据移动，却引入：

- shared ownership；
- aliasing；
- lifetime coupling；
- retention risk。

如果 copy 很小或不在 hot path，复杂度可能不值得。

## 12. 保存 Optimization Intent

一些优化代码看起来会比 naive version 更奇怪：

- unused bounds proof；
- intentional copy；
- padding；
- unusual field order；
- index instead of pointer；
- pool capacity limit。

如果理由无法从局部语法恢复，就要通过 comment、test、benchmark 或 design note 保存。

这不是额外文档负担，而是防止优化被误删的必要条件。

## 13. 为目标环境优化

性能依赖：

- Go version；
- GOARCH；
- CPU；
- OS；
- workload；
- concurrency level。

不要把另一台机器的 benchmark 结果当成普遍事实。

Implementation-sensitive 结论在升级 Go toolchain 后应重新验证。

## 14. 优先 Stable Contract，而不是 Runtime Internal

Runtime source 对理解机制非常有价值。

但 private runtime symbol 不是 public API。

优先依赖：

```text
public language/runtime contract
```

而不是：

```text
linkname/private runtime behavior
```

除非这是明确承担 toolchain compatibility 成本的底层库。

## 15. 使用能够回答问题的最低层工具

从高层证据开始，只有必要时再升级：

```text
metrics / profile
→ benchmark
→ compiler diagnostics
→ SSA / assembly
→ perf / PMU
```

如果 scaling benchmark 已经能清楚说明问题，就没必要先从 assembly 开始。

## 16. 多层验证

Local benchmark 证明 local effect。

System benchmark 证明它是否真的影响应用。

成熟的验证路径是：

```text
microbenchmark
→ component
→ service
→ production/canary
```

修改越广，验证范围越应该扩大。

## 17. 保留 Guardrails

优化一个指标可能恶化另一个指标：

```text
throughput +10%
P99 +50%
```

或者：

```text
CPU -15%
RSS +2×
```

是否可接受取决于系统目标。

任何性能优化都应该明确 guardrails。

## 18. 收益太小时停止

以下情况应该拒绝或停止优化：

- hotspot 很小；
- theoretical gain 很低；
- complexity 明显不成比例；
- system-level benefit 没出现；
- correctness / maintainability risk 太高。

技术上“更快”的实现，并不自动是更好的工程实现。
