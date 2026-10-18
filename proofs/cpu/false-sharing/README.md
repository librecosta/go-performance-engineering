# False Sharing

This implements the Frozen V1.0 experiment contract (2026-10-18). It provides
layout/correctness checks and a benchmark ready for later controlled measurement.
No formal performance result, report, or Windows/Linux comparison is claimed.

## Claim

Two goroutines updating different atomic counters can still contend over a
shared cache line. With two fixed workers and the same total update count,
Packed may cost more than Separated at `GOMAXPROCS=2`, with a larger relative
layout effect than at `GOMAXPROCS=1`.

Address checks plus controlled timings can provide indirect evidence consistent
with false sharing. They cannot measure actual cache-line transfers, establish
physical-core residency, or demonstrate production-service benefits.

## Cost Model

Logical counter ownership and cache-line ownership are different. Independent
writes can require coherence work when their storage shares a line. Separating
the counters may reduce that interference but consumes more layout space in
real applications. This proof keeps allocation size identical between cases.

`GOMAXPROCS=1` prevents simultaneous execution of user Go code; it does not
disable coherence or OS thread migration. `GOMAXPROCS=2` permits parallelism
but does not pin the workers to separate physical cores.

## Baseline

The scope is **amd64 with an externally corroborated 64-byte cache-line model**.
The code validates addresses under that model; it does not detect hardware
cache-line width. Record how the machine's line width was confirmed. If reliable
information is unavailable, keep the assumption explicit and defer formal
interpretation. A conflicting line width makes this experiment inapplicable.

Each case allocates `make([]atomic.Uint64, 23)`. Before choosing pointers, it
checks the element size and every adjacent address stride are eight bytes,
selects the first 64-byte-aligned element, and verifies that a complete
16-element / 128-byte window fits. The allocator is not assumed to provide
64-byte alignment at the allocation's start.

| Property | Packed | Separated |
| --- | --- | --- |
| Backing storage | 23 atomic counters | Same type and length |
| Aligned window | 128 bytes | 128 bytes |
| A offset | 0 bytes | 0 bytes |
| B offset | 8 bytes | 64 bytes |
| `address / 64` relationship | Same line | Different lines |

Every benchmark invocation validates its own allocation. Only the two selected
counters are updated. Typed slices and pointers keep storage reachable through
completion and result checks, followed by `runtime.KeepAlive(storage)`.
`unsafe.Pointer` is converted to `uintptr` only for address inspection; integers
are never converted back to pointers or used as the sole ownership root.

## Experiment

Run both layouts at externally selected P=1 and P=2, for four controlled cells.
Both call the same `update` function, whose loop only performs `counter.Add(1)`.
There are exactly two worker goroutines; each writes only its own counter.

The shared benchmark helper follows this order:

1. Stop the timer; check architecture and actual `runtime.GOMAXPROCS(0)`.
2. Allocate and validate the layout; require distinct, initially zero counters.
3. Create two workers. Each announces readiness immediately before waiting on
   the shared start channel. Wait for both readiness notifications.
4. Reset elapsed time and allocation counters while stopped, then start timing
   and immediately close the start channel.
5. Wait for both workers to finish, then immediately stop timing.
6. Read and validate both final counts, keep storage alive, and log the layout,
   actual P, total updates, and split outside the timer.

The readiness barrier does not promise simultaneous physical execution. Timing
includes release, wakeup, scheduling, and completion-wait overhead. It excludes
storage setup, worker creation, readiness waiting, and final count validation.
There is no thread affinity, `Gosched`, `RunParallel`, special inlining directive,
forced GC, or change to GC configuration.

## Verification

From the shared `proofs/` module directory, select Go 1.26.9 and run only this
package. PowerShell examples below save and restore pre-existing process values:

```powershell
$oldToolchain = $env:GOTOOLCHAIN
try {
    $env:GOTOOLCHAIN = 'go1.26.9'
    go version
    go test -run '^TestFalseSharing' -count=1 ./cpu/false-sharing
} finally {
    $env:GOTOOLCHAIN = $oldToolchain
}
```

Tests cover size/stride checks, window bounds and first-aligned selection,
actual offsets and line relationships, zero initial counts, the start gate,
independent even/odd updates, invalid layout/storage/count/counter inputs, and
the benchmark's P validation. Ordinary unit tests do not require P=1 or P=2.
Unsupported architectures are explicitly skipped for layout/update tests;
supported-platform layout failures fail the test. Benchmarks reject unsupported
P values or bad layouts before starting workers or reporting a valid sample.
No timing or throughput ratio is a correctness assertion.

## Benchmark

**`b.N` is the total number of Atomic Add operations across both workers.**
Formal `-benchtime=Nx` must use a positive even N, giving N/2 updates per worker.
Odd harness probes use floor(N/2) and ceil(N/2), still summing to N. N=1 is
supported for harness probing; zero is rejected. Tiny runs are not formal samples.

The standard `ns/op` is total elapsed wall time divided by N: average wall-clock
cost per aggregate update. It is not a worker's operation latency or CPU cycles.
Derive aggregate updates/s as `1e9 / (ns/op)` without another factor of two,
and do not extrapolate throughput from near-zero-duration probes.

Short smoke checks only, from `proofs/`:

```powershell
$oldToolchain = $env:GOTOOLCHAIN
$oldProcs = $env:GOMAXPROCS
try {
    $env:GOTOOLCHAIN = 'go1.26.9'
    $env:GOMAXPROCS = '1'
    go test -run '^$' -bench '^BenchmarkFalseSharing(Packed|Separated)$' -benchtime=100000x -count=1 ./cpu/false-sharing
    $env:GOMAXPROCS = '2'
    go test -run '^$' -bench '^BenchmarkFalseSharing(Packed|Separated)$' -benchtime=100000x -count=1 ./cpu/false-sharing
} finally {
    $env:GOMAXPROCS = $oldProcs
    $env:GOTOOLCHAIN = $oldToolchain
}
```

These same-process layout pairs are smoke checks, not formal comparisons. For
later formal work, first verify the machine/64-byte preconditions and build one
test binary per platform into a temporary directory. Record build flags, source
commit/dirty state, and SHA-256. Select one positive even N per platform, starting
from 10,000,000 and adjusting only in a P=1 / Separated precheck to approximately
0.3–1.0 seconds. Keep that N for all four cells and repeats on that platform;
changing N requires restarting the whole formal series.

Each formal sample uses a fresh process, one exact benchmark name,
`-test.run=^$`, `-test.benchtime=<N>x`, and `-test.count=1`. Over four rounds,
alternate P-group order (1→2, 2→1, 1→2, 2→1) and within each P group alternate
ABBA / BAAB, where A=Packed and B=Separated. This is 32 processes per platform,
eight samples per cell. Exclude prechecks from formal statistics.

Analyze Windows and Linux separately. Preserve raw samples, order, process/run
identity, N, environment, and anomalies. Compare adjacent paired throughput
ratios `R_P = Separated / Packed`, and the interaction `R_2 / R_1`, with medians
and ranges. Do not pool OS results or gate success on an arbitrary percentage.
Classify results as consistent local evidence, insufficient evidence, no observed
expected effect, or invalid experiment; retain uncertain and negative results.

Linux reproduction is a future stage: cross-compile from Windows for
`linux/amd64`, `GOAMD64=v1`, `CGO_ENABLED=0`, supply build metadata and hashes,
and verify ELF architecture, hashes, permissions, CPU/OS and cgroup quotas in a
fresh temporary environment before running. Preserve any missing VCS metadata
or `vcs.modified=true` limitation. No formal report or binary is added here.

## Environment

Record actual Go version, GOOS/GOARCH/GOAMD64, build flags, CPU model, physical
and logical cores, OS, power/load conditions, affinity and virtualization/quota
limits, GOMAXPROCS, GOGC, GOMEMLIMIT, GODEBUG, and cache-line evidence. Keep GC
settings and build options identical across the four cells. Unsupported machine
conditions invalidate formal interpretation; an address-math pass is not a
hardware detector.

## Caveats

Scheduling, SMT placement, migration, other load, and synchronization overhead
can affect results. P=1 is not permanent single-core binding; P=2 is not a
guarantee of distinct physical cores. Neither timings nor address checks measure
HITM/cache-to-cache events. No stable speedup, production QPS/P99 gain, or
cross-machine ranking follows from smoke output.

## Recommendation

Separate frequently updated fields only when the real workload and evidence
justify the extra memory/cache footprint. Do not add padding unconditionally
or infer false sharing from independent variables alone.

## References

- [Linux Kernel: False Sharing](https://kernel.org/doc/html/latest/kernel-hacking/false-sharing.html)
- [Intel VTune Cookbook: False Sharing](https://www.intel.com/content/www/us/en/docs/vtune-profiler/cookbook/2024-0/false-sharing.html)
- [Go sync/atomic](https://pkg.go.dev/sync/atomic)
- [Go testing](https://pkg.go.dev/testing) and [go test flags](https://pkg.go.dev/cmd/go#hdr-Testing_flags)
- [Go runtime.GOMAXPROCS](https://pkg.go.dev/runtime#GOMAXPROCS) and [unsafe](https://pkg.go.dev/unsafe)
- [Proof Guidelines](../../README.md)
