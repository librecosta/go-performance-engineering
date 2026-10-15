# Go Performance Engineering

[English](./README.md) | 简体中文

一套以证据为基础的 Go 性能工程知识库，包含详细文档、Agent Skill 与可复现的性能机制证明。

## 概览

这个仓库关注的是如何通过证据理解并改善 Go 程序的性能。

它不是一份孤立的“性能技巧合集”。

项目覆盖：

- Compiler、SSA 与编译器优化；
- CPU Cache 与内存布局；
- 并发与同步；
- 内存分配与垃圾回收；
- `unsafe`、cgo 与 runtime boundary；
- Benchmark、Profiling 与性能工程方法论。

## 仓库结构

```text
.
├── docs/
│   完整的 Go Performance Engineering Handbook
│
├── skill/
│   面向 AI Agent 的性能工程 Skill
│
└── proofs/
    可复现的最小性能机制实验
```

## 文档

`docs/` 是完整知识体系，面向工程师阅读。

英文版本是 canonical source：

- [Go Performance Engineering Handbook](./docs/README.md)

简体中文版本按相同章节结构维护：

- [Go 性能工程手册](./docs/zh-CN/README.md)

如果中英文内容存在技术性差异，以英文版本以及对应的 Go 官方资料为准。

## Agent Skill

`skill/` 包含面向 AI Agent 的性能工程工作流。

Skill 内部统一使用英文，以便与 Go 官方文档、Compiler/Runtime 源码、诊断输出以及各类 Agent 工具保持一致。

它提供：

- diagnostic routing；
- evidence requirements；
- optimization risk rules；
- maintainability checks；
- validation workflow；
- regression strategy。

入口：

- [SKILL.md](./skill/SKILL.md)

Agent 的内部 Skill 使用英文，不限制最终回复语言。面向用户时应优先使用用户当前使用的语言，同时保留 Go API、编译器诊断、命令和官方术语的原始形式。

## Proofs

`proofs/` 包含针对具体性能行为的最小可复现实验。

一个 proof 用于证明：

> 某个机制在明确记录的条件下真实存在，并且可以被复现。

它不用于宣称：

> 某种优化永远更快。

参见：

- [Proof Guidelines](./proofs/README.md)

## 技术文章与实验记录

我们在 [Gopher Atlas](https://gopheratlas.com/) 发布与本项目对应的性能工程文章，将底层原理、工程取舍与可复现的实验证据关联起来。以下文章均提供简体中文和英文版本。

| 研究主题 | 技术文章 | 可复现实验证据 |
| --- | --- | --- |
| **Slice Retention** — 为什么 100 字节能留住 100 MiB？ | [简体中文](https://gopheratlas.com/notes/go-performance/go-slice-retention-100-bytes-100-mib/) · [English](https://gopheratlas.com/notes/go-performance/go-slice-retention-100-bytes-100-mib-en/) | [Proof](./proofs/memory/slice-retention/) · [Windows 报告](./proofs/memory/slice-retention/reports/2026-10-14_win_go1269_bc122b8.md) · [Linux 报告](./proofs/memory/slice-retention/reports/2026-10-14_linux_go1269_bc122b8.md) |
| **Heap Ballast（历史技术）** — Go GC 压舱石的原理与演变 | [简体中文](https://gopheratlas.com/notes/go-performance/go-gc-heap-ballast/) · [English](https://gopheratlas.com/notes/go-performance/go-gc-heap-ballast-en/) | [Proof](./proofs/memory/ballast/) · [Windows 报告](./proofs/memory/ballast/reports/2026-09-02_142711_80d692d.md) · [Linux 报告](./proofs/memory/ballast/reports/2026-10-14_linux_go1269_dab084e.md) |
| **Bounds Check Elimination** — 编译器如何证明边界检查是多余的？ | [简体中文](https://gopheratlas.com/notes/go-performance/go-bounds-check-elimination/) · [English](https://gopheratlas.com/notes/go-performance/go-bounds-check-elimination-en/) | [Proof](./proofs/compiler/bounds-check-elimination/) · [Windows 报告](./proofs/compiler/bounds-check-elimination/reports/2026-10-15_win_go1269_e835186.md) · [Linux 报告](./proofs/compiler/bounds-check-elimination/reports/2026-10-16_linux_go1269_e835186.md) |

这些实验仅在明确记录的条件下证明具体机制，不宣称普遍的性能收益。历史技术用于理解原理与技术演变，不代表现代 Go 应用的默认实践建议。

## 核心理念

```text
Measure
  ↓
Understand the cost model
  ↓
Make the smallest justified change
  ↓
Benchmark
  ↓
Validate system impact
  ↓
Preserve the reason for the optimization
```

性能工程的目标不是让代码看起来更聪明，而是让系统成本变得可测量、可解释、可验证、可维护。

## License

Apache License 2.0
