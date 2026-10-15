# Go Performance Engineering Handbook

[English](./README.md) | [简体中文](./zh-CN/README.md)

This handbook explains Go performance engineering from first principles.

It is the human-readable knowledge layer of this repository:

- `docs/` explains mechanisms, cost models, trade-offs, and engineering reasoning;
- `skill/` converts that knowledge into an operational workflow for AI agents;
- `proofs/` contains minimal reproducible experiments for specific performance claims.

The goal is not to collect optimization tricks. The goal is to understand **why** a Go program behaves the way it does, how to verify that behavior, and when an optimization is justified.

## Languages

English is the canonical version of the handbook.

The Simplified Chinese translation is maintained under:

```text
docs/zh-CN/
```

The Chinese tree mirrors the English chapter and file layout so that links and concepts remain easy to compare across languages.

Translation policy:

- technical meaning should remain equivalent to the English source;
- Go identifiers, commands, diagnostics, API names, and compiler/runtime terms should remain in their original form where translation would reduce precision;
- common terms may be introduced bilingually on first use, for example `逃逸分析（escape analysis）`;
- when the two language versions disagree on a technical claim, the English canonical document and the cited primary source take precedence.

## Scope

The handbook studies performance across several interacting layers:

```text
Application design
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

A symptom visible in one layer may originate in another.

Examples:

- excessive allocation may appear as GC CPU;
- a hot atomic operation may originate from cache-line coherence;
- interface abstraction may matter because it prevents devirtualization or inlining;
- a tiny slice may retain a very large backing array;
- a lock-free algorithm may consume more CPU than a mutex under contention;
- low pause time does not rule out GC assist or concurrent GC CPU as a latency contributor.

The handbook provides the cost models needed to reason across these boundaries.

## Reading Order

### 00 — Foundations

Start here before studying individual mechanisms.

- [Performance Engineering](./00-foundations/performance-engineering.md)
- [Cost Models](./00-foundations/cost-models.md)
- [Optimization Principles](./00-foundations/optimization-principles.md)

These chapters define the vocabulary and evidence-oriented mindset used throughout the handbook.

### 01 — CPU and Memory

- [Cache Hierarchy](./01-cpu-and-memory/cache-hierarchy.md)
- [Memory Locality](./01-cpu-and-memory/memory-locality.md)
- [Branch Prediction](./01-cpu-and-memory/branch-prediction.md)
- [TLB and Pages](./01-cpu-and-memory/tlb-and-pages.md)
- [Data Layout](./01-cpu-and-memory/data-layout.md)

These chapters explain how access patterns and Go data structures interact with modern CPU and memory systems.

### 02 — Concurrency

- [Synchronization Model](./02-concurrency/synchronization-model.md)
- [Mutex](./02-concurrency/mutex.md)
- [Atomic Operations](./02-concurrency/atomic.md)
- [RWMutex](./02-concurrency/rwmutex.md)
- [Channels](./02-concurrency/channels.md)
- [Lock-Free Algorithms](./02-concurrency/lock-free.md)
- [The ABA Problem](./02-concurrency/aba.md)
- [Ownership-Oriented Design](./02-concurrency/ownership-design.md)

These chapters explain concurrency performance through shared state, cache coherence, waiting, retry, scheduling, and ownership.

### 03 — Compiler

- [Compiler Pipeline](./03-compiler/compiler-pipeline.md)
- [Static Single Assignment](./03-compiler/ssa.md)
- [Escape Analysis](./03-compiler/escape-analysis.md)
- [Inlining](./03-compiler/inlining.md)
- [Bounds Check Elimination](./03-compiler/bounds-check-elimination.md)
- [Devirtualization](./03-compiler/devirtualization.md)
- [Profile-Guided Optimization](./03-compiler/pgo.md)

These chapters explain how source-level facts become compiler proofs and how those proofs can eliminate runtime work.

### 04 — Memory and GC

- [Allocator](./04-memory-and-gc/allocator.md)
- [Heap Model](./04-memory-and-gc/heap-model.md)
- [Allocation Patterns](./04-memory-and-gc/allocation-patterns.md)
- [Garbage Collector](./04-memory-and-gc/garbage-collector.md)
- [GC Pacing](./04-memory-and-gc/gc-pacing.md)
- [Memory Retention](./04-memory-and-gc/retention.md)
- [sync.Pool](./04-memory-and-gc/sync-pool.md)
- [Memory Limits, RSS, and Scavenging](./04-memory-and-gc/memory-limits.md)

These chapters separate allocation cost, allocation churn, live heap, scannable heap, retention, GC work, runtime memory, and process RSS.

### 05 — Runtime Boundary

- [Unsafe](./05-runtime-boundary/unsafe.md)
- [Zero-Copy](./05-runtime-boundary/zero-copy.md)
- [cgo](./05-runtime-boundary/cgo.md)
- [mmap](./05-runtime-boundary/mmap.md)
- [Runtime and Compiler Boundaries](./05-runtime-boundary/runtime-internals.md)

These chapters cover the highest-risk performance techniques, where ownership, lifetime, GC visibility, ABI behavior, or private implementation contracts become part of correctness.

### 06 — Methodology

- [Profiling](./06-methodology/profiling.md)
- [Benchmarking](./06-methodology/benchmarking.md)
- [Evidence Model](./06-methodology/evidence-model.md)
- [Optimization Review](./06-methodology/optimization-review.md)
- [Regression Strategy](./06-methodology/regression-strategy.md)

These chapters turn the previous mechanisms into a complete engineering process: locate cost, formulate a mechanism hypothesis, compare candidates, validate system impact, and preserve the result.

### Sources

- [Official Sources](./sources/official-sources.md)

The source index records the primary Go documentation, package references, release notes, runtime/compiler source, and diagnostic tooling used by the handbook.

## How to Read Performance Claims

A useful performance claim should connect:

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

Statements such as:

> Atomics are faster than mutexes.

or:

> Zero-copy is always better.

are not useful without workload conditions.

A performance result should answer:

- What cost changed?
- Why should that mechanism affect the cost?
- Under what workload was it measured?
- What new cost or complexity was introduced?
- How was the effect verified?
- Does the result still hold on the target Go version, architecture, and hardware?

## Relationship to Proofs

The handbook explains mechanisms.

The `proofs/` tree demonstrates selected mechanisms with minimal reproducible experiments.

For example:

```text
docs/03-compiler/bounds-check-elimination.md
        ↓ explains

proofs/compiler/bounds-check-elimination/
        ↓ demonstrates
```

A proof is not expected to show a universal percentage improvement.

Its job is to establish that a specific mechanism exists and can be reproduced under documented conditions.

## Relationship to the Agent Skill

The handbook and Skill intentionally serve different purposes.

The handbook answers:

> Why does this behavior exist, and how should an engineer reason about it?

The Skill answers:

> When should an agent investigate this behavior, what evidence is required, and what actions are safe to recommend?

The Skill should therefore remain smaller, more operational, and progressively load only the references needed for the current problem.

## Version Sensitivity

Compiler and runtime behavior evolve.

Implementation-sensitive claims involving:

- inlining;
- escape analysis;
- bounds-check elimination;
- devirtualization;
- generic lowering;
- allocator paths and thresholds;
- garbage-collector internals;
- runtime synchronization;
- cgo boundary cost;
- architecture-specific code generation;

must be revalidated on the target Go toolchain.

Hardware-sensitive claims must also be interpreted in the context of the target CPU, operating system, and workload.

Historical benchmark results are useful evidence, but they are not permanent implementation contracts.

## Evidence Principle

The central principle of this handbook is:

> Optimize measured system behavior, not code cleverness.

A valid performance investigation may conclude that no optimization should be made.
