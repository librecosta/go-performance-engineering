# Go Performance Engineering Skill

This directory contains the Agent Skill for `go-performance-engineering`.

The Skill is the operational layer of the repository. It helps an AI agent investigate, review, and implement Go performance work without treating source-level patterns as proof of bottlenecks.

## Role in the Repository

The repository has three distinct layers:

```text
docs/
    Detailed human-readable handbook
    Mechanisms, cost models, trade-offs, engineering reasoning

skill/
    Agent operational layer
    Routing, evidence thresholds, decisions, safety, review rules

proofs/
    Reproducible evidence
    Minimal experiments for specific compiler/runtime/hardware claims
```

The Skill should not duplicate the handbook.

It should convert detailed knowledge into concise decision rules that an agent can progressively load as needed.

## Layout

```text
skill/
├── SKILL.md
├── README.md
├── agents/
│   └── openai.yaml
├── assets/
│   └── icon.svg
└── references/
    ├── methodology.md
    ├── benchmarking-profiling.md
    ├── cpu-memory.md
    ├── concurrency.md
    ├── compiler.md
    ├── memory-gc.md
    ├── unsafe-runtime.md
    ├── optimization-review.md
    ├── technique-catalog.md
    ├── tool-reference.md
    ├── version-notes.md
    └── official-sources.md
```

## `SKILL.md`

`SKILL.md` is the entry point.

It contains the mandatory performance-engineering workflow and the rules that should be available before any deeper reference is loaded.

Core concerns include:

- diagnostic routing;
- evidence levels;
- cost-model reasoning;
- benchmark requirements;
- optimization risk escalation;
- maintainability requirements;
- unsafe/runtime-boundary rules;
- stop conditions;
- definition of done.

A central rule is:

> Never infer a bottleneck from source shape alone.

Allocation, mutexes, interfaces, bounds checks, copies, atomics, pointer-heavy structures, and GC activity are candidates for investigation, not proof of a performance problem.

## Progressive References

`references/` is designed for progressive loading.

An agent should load only the material relevant to the current investigation.

Conceptually:

```text
SKILL.md
   ↓
identify problem class
   ↓
load one or more focused references
   ↓
measure / reason / validate
```

The references are not intended to be smaller copies of `docs/`.

They should focus on:

- when a mechanism matters;
- the cost model;
- what evidence is required;
- which actions are preferred;
- which actions require escalation;
- which conclusions must remain version-sensitive.

## Language Policy

The Skill package is maintained in **English only**.

This includes:

- `SKILL.md`;
- `references/*`;
- `agents/openai.yaml`;
- code/command examples;
- performance terminology used as operational identifiers.

The reason is consistency with:

- Go documentation;
- compiler/runtime source;
- compiler diagnostics;
- pprof and trace tooling;
- benchmark output;
- cross-agent portability.

This does **not** require user-facing responses to be English.

An agent should respond in the user's language unless explicitly requested otherwise, while preserving Go identifiers, commands, diagnostics, API names, and quoted official terminology in their original form.

Human-oriented bilingual material belongs in `docs/`, with English as the canonical handbook and Simplified Chinese under `docs/zh-CN/`.

## Evidence Model

The Skill uses a graduated evidence model:

```text
L0  Source inspection
L1  Profile / compiler / runtime diagnostics
L2  Reproducible microbenchmark
L3  Component / service benchmark
L4  Production / canary evidence
L5  Hardware / PMU evidence
```

Evidence requirements increase with optimization risk.

A simple idiomatic preallocation does not require the same proof burden as:

- unsafe zero-copy;
- a custom lock-free algorithm;
- assembly;
- compiler contracts;
- private runtime dependencies.

## Maintainability Policy

Maintainability is part of performance correctness.

Non-obvious performance code must preserve the reason it exists.

Typical examples include:

- bounds-proof patterns;
- cache-line padding;
- unusual field order;
- intentional copy to avoid retention;
- pool capacity cutoffs;
- pointer-to-index representations;
- AoS/SoA transformations;
- hot/cold splits;
- sharding and batching invariants;
- CAS/backoff logic;
- unsafe zero-copy;
- mmap lifetime;
- cgo pointer contracts;
- compiler directives;
- runtime-private dependencies.

When relevant, comments should preserve:

```text
WHY
MECHANISM
PRESERVATION
SAFETY CONTRACT
```

Ordinary idiomatic optimizations should not be buried under redundant comments.

## Runtime and Unsafe Boundary

The Skill should escalate deliberately:

```text
safe Go
→ compiler-friendly restructuring
→ safe representation / ownership change
→ public unsafe API
→ specialized FFI / assembly contract
→ private runtime mechanism only in exceptional low-level work
```

Runtime source is an important source of cost models.

It is not automatically a public performance API.

## Version Sensitivity

Compiler and runtime implementation details must be revalidated on the target Go version.

Examples include:

- inlining;
- escape analysis;
- BCE;
- devirtualization;
- generic lowering;
- allocator paths and thresholds;
- GC implementation;
- cgo overhead;
- architecture-specific generated code.

Historical benchmark results should be treated as hypotheses unless they closely match the target toolchain and environment.

## Relationship to Proofs

The Skill may use `proofs/` as reproducible evidence for specific mechanisms.

A proof should demonstrate:

> This mechanism exists under these documented conditions.

It should not be converted into:

> Always apply this optimization.

The agent must still evaluate workload, system impact, complexity, and guardrails.

## Design Goal

The Skill should help an agent produce performance work that is:

- measured;
- explainable;
- reproducible;
- risk-aware;
- maintainable;
- regression-guarded.

The desired outcome is not clever code.

It is justified engineering.
