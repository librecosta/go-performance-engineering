# Go Performance Engineering

English | [简体中文](./README.zh-CN.md)

Evidence-driven Go performance engineering with documentation, Agent Skill, and reproducible proofs.

## Overview

This repository focuses on understanding and improving Go performance through evidence.

It is not a collection of isolated optimization tricks.

The project covers:

- compiler and SSA optimization;
- CPU cache and memory layout;
- concurrency and synchronization;
- allocation and garbage collection;
- unsafe and runtime boundaries;
- benchmarking and profiling methodology.

## Repository Structure

```text
.
├── docs/
│   Complete performance engineering documentation
│
├── skill/
│   Agent Skill package
│
└── proofs/
    Minimal reproducible experiments
```

## Documentation

The `docs/` directory contains the complete knowledge base.

Start here:

- [Go Performance Engineering Handbook](./docs/README.md)

## Agent Skill

The `skill/` directory contains the AI-oriented performance workflow.

It provides:

- diagnostic routing;
- evidence requirements;
- optimization risk rules;
- maintainability checks;
- validation workflow.

Start here:

- [SKILL.md](./skill/SKILL.md)

## Proofs

The `proofs/` directory contains minimal experiments for specific performance behaviors.

A proof demonstrates:

> This mechanism exists and can be reproduced under documented conditions.

It does not claim:

> This optimization is always faster.

See:

- [Proof Guidelines](./proofs/README.md)

## Articles & Case Studies

Articles published on [Gopher Atlas](https://gopheratlas.com/) connect Go performance concepts with reproducible experiments and documented evidence. Both English and Simplified Chinese editions are available.

| Study | Articles | Reproducible evidence |
| --- | --- | --- |
| **Slice Retention** — How a 100-byte slice can keep a 100 MiB backing array alive | [English](https://gopheratlas.com/notes/go-performance/go-slice-retention-100-bytes-100-mib-en/) · [简体中文](https://gopheratlas.com/notes/go-performance/go-slice-retention-100-bytes-100-mib/) | [Proof](./proofs/memory/slice-retention/) · [Windows report](./proofs/memory/slice-retention/reports/2026-10-14_win_go1269_bc122b8.md) · [Linux report](./proofs/memory/slice-retention/reports/2026-10-14_linux_go1269_bc122b8.md) |
| **Heap Ballast (Historical)** — The mechanism and evolution of a classic Go GC tuning technique | [English](https://gopheratlas.com/notes/go-performance/go-gc-heap-ballast-en/) · [简体中文](https://gopheratlas.com/notes/go-performance/go-gc-heap-ballast/) | [Proof](./proofs/memory/ballast/) · [Windows report](./proofs/memory/ballast/reports/2026-09-02_142711_80d692d.md) · [Linux report](./proofs/memory/ballast/reports/2026-10-14_linux_go1269_dab084e.md) |

The experiments establish specific mechanisms under recorded conditions, not universal performance gains. Historical techniques are preserved for study and reproducibility, not as default recommendations for modern Go applications.

## Philosophy

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

## License

Apache License 2.0
