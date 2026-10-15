# Go 性能工程手册

[English](../README.md) | 简体中文

本手册从基础原理出发，系统解释 Go Performance Engineering（Go 性能工程）。

它是本仓库中面向工程师阅读的知识层：

- `docs/` 解释机制、成本模型、工程权衡与推理过程；
- `skill/` 将这些知识压缩为面向 AI Agent 的操作工作流；
- `proofs/` 为特定性能机制提供最小可复现实验。

这里的目标不是收集“优化技巧”，而是理解 **Go 程序为什么会表现成这样、如何验证这种行为，以及什么时候值得优化**。

## 语言

英文版本是 Handbook 的 canonical source。

简体中文版本位于：

```text
docs/zh-CN/
```

中文目录与英文目录保持一一对应，方便对照维护。

翻译规则：

- 技术含义与英文版本保持一致；
- Go identifier、命令、编译器诊断、API 名称、runtime/compiler 术语在翻译会损失精度时保留英文；
- 常见专业术语首次出现时可以使用“中文（English term）”，例如 `逃逸分析（escape analysis）`；
- 如果中英文对某个技术事实出现差异，以英文 canonical 文档与对应的一手官方资料为准。

## 范围

本手册从多个相互作用的层次研究性能：

```text
应用设计
   ↓
算法与数据结构
   ↓
Go Compiler
   ↓
Go Runtime
   ↓
Operating System
   ↓
CPU 与 Memory Hardware
```

一个层面看到的症状，根因可能来自另一个层面。

例如：

- 大量 allocation 可能最终表现为 GC CPU；
- 一个很热的 atomic 操作，根因可能是 cache-line coherence；
- interface 的损失可能不只是一次 indirect call，而是阻断 devirtualization 与 inlining；
- 一个很小的 slice 可能保留巨大的 backing array；
- lock-free 算法在高 contention 下可能比 Mutex 消耗更多 CPU；
- STW 很小并不意味着 GC assist 或 concurrent GC CPU 不会影响尾延迟。

本手册的目标就是提供跨越这些层次进行推理所需要的成本模型。

## 阅读顺序

### 00 — 基础

建议在研究具体机制之前先阅读：

- [性能工程](./00-foundations/performance-engineering.md)
- [成本模型](./00-foundations/cost-models.md)
- [优化原则](./00-foundations/optimization-principles.md)

这些章节建立全书共同使用的术语、证据观与工程边界。

### 01 — CPU 与内存

- [缓存层级](./01-cpu-and-memory/cache-hierarchy.md)
- [内存局部性](./01-cpu-and-memory/memory-locality.md)
- [分支预测](./01-cpu-and-memory/branch-prediction.md)
- [TLB 与内存页](./01-cpu-and-memory/tlb-and-pages.md)
- [数据布局](./01-cpu-and-memory/data-layout.md)

这些章节解释 Go 数据结构和访问模式如何映射到真实 CPU / memory system。

### 02 — 并发

- [同步模型](./02-concurrency/synchronization-model.md)
- [Mutex](./02-concurrency/mutex.md)
- [Atomic 操作](./02-concurrency/atomic.md)
- [RWMutex](./02-concurrency/rwmutex.md)
- [Channel](./02-concurrency/channels.md)
- [Lock-Free 算法](./02-concurrency/lock-free.md)
- [ABA 问题](./02-concurrency/aba.md)
- [面向所有权的设计](./02-concurrency/ownership-design.md)

这些章节从 shared state、cache coherence、waiting、retry、scheduler 与 ownership 解释并发性能。

### 03 — Compiler

- [Compiler Pipeline](./03-compiler/compiler-pipeline.md)
- [Static Single Assignment](./03-compiler/ssa.md)
- [逃逸分析](./03-compiler/escape-analysis.md)
- [Inlining](./03-compiler/inlining.md)
- [Bounds Check Elimination](./03-compiler/bounds-check-elimination.md)
- [Devirtualization](./03-compiler/devirtualization.md)
- [Profile-Guided Optimization](./03-compiler/pgo.md)

这些章节解释源码中的事实如何变成 compiler proof，以及 compiler 如何据此删除运行时工作。

### 04 — 内存与 GC

- [Allocator](./04-memory-and-gc/allocator.md)
- [Heap Model](./04-memory-and-gc/heap-model.md)
- [Allocation Patterns](./04-memory-and-gc/allocation-patterns.md)
- [Garbage Collector](./04-memory-and-gc/garbage-collector.md)
- [GC Pacing](./04-memory-and-gc/gc-pacing.md)
- [Memory Retention](./04-memory-and-gc/retention.md)
- [sync.Pool](./04-memory-and-gc/sync-pool.md)
- [Memory Limit、RSS 与 Scavenging](./04-memory-and-gc/memory-limits.md)

这些章节把 allocation cost、allocation churn、live heap、scannable heap、retention、GC work、runtime memory 与 process RSS 区分开来。

### 05 — Runtime Boundary

- [Unsafe](./05-runtime-boundary/unsafe.md)
- [Zero-Copy](./05-runtime-boundary/zero-copy.md)
- [cgo](./05-runtime-boundary/cgo.md)
- [mmap](./05-runtime-boundary/mmap.md)
- [Runtime 与 Compiler Boundary](./05-runtime-boundary/runtime-internals.md)

这些章节讨论风险最高的一组技术：ownership、lifetime、GC visibility、ABI 或 private implementation contract 会直接参与 correctness。

### 06 — 方法论

- [Profiling](./06-methodology/profiling.md)
- [Benchmarking](./06-methodology/benchmarking.md)
- [Evidence Model](./06-methodology/evidence-model.md)
- [Optimization Review](./06-methodology/optimization-review.md)
- [Regression Strategy](./06-methodology/regression-strategy.md)

这一部分把前面的机制知识收束成完整工程过程：定位成本、建立机制假设、比较候选方案、验证系统收益，并长期保护优化结果。

### Sources

- [官方资料索引](./sources/official-sources.md)

这里整理本手册依赖的 Go 官方文档、package reference、release notes、runtime/compiler source 与诊断工具。

## 如何阅读性能结论

一个有价值的性能结论应该建立这样的证据链：

```text
Observation
    ↓
Mechanism
    ↓
Cost Model
    ↓
Evidence
    ↓
Conditions
    ↓
Trade-off
```

类似：

> Atomic 比 Mutex 快。

或者：

> Zero-copy 总是更好。

如果没有 workload 条件，这些说法几乎没有工程价值。

一个性能结论至少应该回答：

- 改变了什么成本？
- 为什么这个机制会影响该成本？
- 在什么 workload 下测量？
- 引入了什么新的成本或复杂度？
- 机制是否被直接验证？
- 在目标 Go version、architecture 与 hardware 上是否仍然成立？

## 与 Proofs 的关系

Handbook 解释机制。

`proofs/` 使用最小可复现实验验证其中一部分机制。

例如：

```text
docs/03-compiler/bounds-check-elimination.md
        ↓ explains

proofs/compiler/bounds-check-elimination/
        ↓ demonstrates
```

一个 proof 不应该宣称某种技术存在“通用百分比提升”。

它的职责是证明：

> 在明确记录的条件下，这个机制确实存在并且可以复现。

## 与 Agent Skill 的关系

Handbook 与 Skill 的职责不同。

Handbook 回答：

> 这种行为为什么存在？工程师应该如何理解它？

Skill 回答：

> Agent 什么时候应该调查它？需要什么证据？哪些操作可以安全推荐？

因此 Skill 应保持更小、更偏决策与操作，并只按需加载相关 references。

## 版本敏感性

Compiler 和 Runtime 会持续演进。

以下 implementation-sensitive 结论必须在目标 Go toolchain 上重新验证：

- inlining；
- escape analysis；
- bounds-check elimination；
- devirtualization；
- generic lowering；
- allocator path 与 threshold；
- garbage collector internals；
- runtime synchronization；
- cgo boundary cost；
- architecture-specific code generation。

Hardware-sensitive 结论也必须结合目标 CPU、OS 与 workload。

历史 benchmark 是有价值的证据，但不是永久 implementation contract。

## 核心原则

本手册最重要的一条原则是：

> 优化经过测量的系统行为，而不是优化代码的“聪明程度”。

一个完全有效的性能调查结果，也可以是：

> 不应该优化这条路径。
