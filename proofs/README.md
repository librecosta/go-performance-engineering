# Performance Proofs

Minimal reproducible experiments for Go performance engineering claims.

## Running Proofs

All current proofs share [go.mod](go.mod), with module path
`github.com/librecosta/go-performance-engineering/proofs` and minimum Go language
version `1.26.0`. Add new proof packages under this module without nested modules
or a `go.work` file.

The repository's [.go-version](../.go-version) selects Go **1.26.9** as the default
experiment baseline. The Go command does not read that file automatically: use
a version manager or explicitly select `GOTOOLCHAIN=go1.26.9` (PowerShell:
`$env:GOTOOLCHAIN = 'go1.26.9'`) before running checks. Record the actual `go version`
for every experiment. Historical reports retain the versions used at the time.

From the repository root:

```bash
cd proofs
go version
go vet ./...
go test -count=1 ./...
```

The [Proofs workflow](../.github/workflows/proofs.yml) runs these checks on Ubuntu
with Go 1.26.9 and 1.27.2 when proofs, the version file, or the workflow change;
it can also be dispatched manually. CI validates correctness and mechanism
contracts, without formal benchmarks or timing/RSS performance gates. Individual
proof READMEs document manual experiments and package-specific commands.

## Purpose

Proofs demonstrate specific compiler, runtime, CPU, memory, and concurrency behaviors.

Each proof should explain:

- the claim;
- the cost model;
- the baseline;
- the experiment;
- the verification method;
- the benchmark or evidence;
- limitations.

A proof is not a benchmark competition.

The goal is understanding, not claiming universal speed improvements.

## Structure

```text
proof-name/
├── README.md
├── source files
└── benchmark/test files
```

Recommended sections:

```markdown
# Title

## Claim

## Cost Model

## Baseline

## Experiment

## Verification

## Benchmark

## Environment

## Caveats

## Recommendation
```

## Categories

```text
proofs/
├── compiler/
│   Compiler and optimizer behavior
│
├── cpu/
│   Cache, layout, and microarchitecture behavior
│
├── concurrency/
│   Synchronization and contention behavior
│
├── memory/
│   Allocation, GC, and retention behavior
│
└── unsafe/
    Unsafe, FFI, and runtime-boundary behavior
```

## Evidence Principle

Prefer:

- compiler diagnostics;
- benchmarks;
- runtime metrics;
- profiles;
- traces;
- hardware counters when required.

Avoid:

> This is faster.

Prefer:

> Under this workload and environment, this implementation reduces this measured cost.

## Historical Proofs

Some proofs may demonstrate techniques that are no longer recommended.

Examples:

- GC ballast;
- old unsafe patterns;
- previous compiler/runtime behavior.

Their purpose is understanding why a technique worked and why recommendations changed.
