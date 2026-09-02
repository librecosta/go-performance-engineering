# Heap Ballast

| Property | Value |
|---|---|
| Category | Memory / GC |
| Proof type | Runtime mechanism |
| Go baseline | Go 1.27 |
| Evidence | L1 + L2 |
| Recommendation | **Historical** |

## Claim

Under ordinary GOGC-based pacing, with no binding runtime memory limit, a long-lived pointer-free heap ballast increases the live heap used by the GC pacer.

That increases the next heap goal and therefore gives the application more allocation headroom between GC cycles.

For the same short-lived allocation-churn workload, this should produce fewer GC cycles.

This proof demonstrates that mechanism.

It does **not** claim that heap ballast is a recommended modern Go tuning technique.

## Cost Model

At a high level, GOGC controls proportional heap growth.

Conceptually:

```text
larger live heap
        ↓
larger GC heap target
        ↓
more allocation headroom
        ↓
fewer GC cycles for the same churn
        ↓
less GC work frequency
```

A ballast historically exploited this by keeping a large pointer-free object live:

```go
ballast := make([]byte, size)
```

The ballast itself contains no Go pointers, so it adds heap size without creating a large pointer graph to scan.

The trade-off is explicit:

```text
GC frequency ↓
memory headroom / heap footprint ↑
```

## Baseline

The baseline runs the same allocation-churn workload with:

```text
ballast = 0
GOGC = 100
runtime memory limit = effectively disabled
```

The experiment forces a GC before measuring the starting heap state.

## Experiment

The ballast case is identical except that it keeps a 64 MiB `[]byte` alive through the churn phase.

The workload repeatedly creates short-lived 8 KiB byte slices in small batches.

This gives the experiment:

```text
small live working set
+
high allocation churn
```

which makes GC cycle frequency easy to observe.

The ballast is kept live explicitly:

```go
runtime.KeepAlive(ballast)
```

This avoids relying on lexical scope as a lifetime guarantee.

## Verification

The machine-checkable proof compares **separate child processes**.

This matters because running baseline and ballast sequentially in the same process would let:

- previous GC history;
- span reuse;
- heap state;
- scavenger state;

contaminate the second case.

The test asserts mechanism direction:

```text
with ballast:

HeapAlloc after setup GC  ↑
NextGC                    ↑
GC cycles during churn    ↓
```

It also checks that both child processes actually allocate at least the requested churn volume.

The test deliberately requires a large GC-cycle difference rather than a fragile few-percent threshold.

Run:

```bash
cd proofs
go test -run '^TestBallastMechanism$' -v ./memory/ballast
```

## Manual Reproduction

Run the baseline:

```bash
cd proofs/memory/ballast
go run . -ballast-mib=0
```

Then run the ballast case in a **new process**:

```bash
go run . -ballast-mib=64
```

Typical output fields:

```text
Heap after setup GC
Next GC goal
GC cycles in churn
GC pause in churn
Allocated in churn
Elapsed
```

The expected direction is:

| Metric | With ballast |
|---|---|
| Heap after setup GC | ↑ |
| Next GC goal | ↑ |
| GC cycles | ↓ |
| GC pause / elapsed | workload and machine dependent |
| Memory budget | ↑ |

Do not expect exact numbers to match across Go versions or machines.

For machine-readable output:

```bash
go run . -ballast-mib=64 -json
```

## Benchmark / Performance Observation

This proof intentionally does **not** make wall-clock timing a CI pass/fail condition.

`Elapsed` and GC pause data are printed because they are useful observations, but shared CI runners are too noisy to safely assert:

```text
ballast elapsed < baseline elapsed
```

A reliable proof should first establish the mechanism.

If you want to study performance impact, repeat both commands on a controlled machine and compare distributions.

The performance result may vary because of:

- Go version;
- CPU;
- available cores;
- scheduler noise;
- memory subsystem;
- GC implementation;
- workload size.

## Why the Proof Fixes GOGC and Disables GOMEMLIMIT

The point of this experiment is to isolate the historical **GOGC-derived ballast mechanism**.

The code therefore uses:

```go
debug.SetGCPercent(100)
debug.SetMemoryLimit(math.MaxInt64)
```

A binding `GOMEMLIMIT` changes GC pacing and answers a different question.

That behavior belongs in the separate:

```text
proofs/memory/gogc-memory-limit/
```

proof.

## Caveats

### This is a historical technique

The mechanism is real, but modern Go has supported memory controls.

Since Go 1.19, `GOMEMLIMIT` / `debug.SetMemoryLimit` provide a supported way to express a runtime memory budget.

Modern applications should normally tune the real policy:

```text
GOGC
+
GOMEMLIMIT
```

instead of manufacturing an artificial live heap.

### Ballast is not free memory

The proof intentionally does not claim that the nominal ballast has zero RSS cost.

Virtual-memory residency and page backing are operating-system/runtime details and vary by environment.

### GC implementation changes

The collector has evolved substantially, including Green Tea becoming the default in modern Go.

The exact CPU or elapsed-time effect of ballast is therefore version-sensitive even though the GOGC pacing mechanism remains demonstrable.

### This proof uses a synthetic workload

The allocation churn is deliberately simple so the mechanism is easy to understand.

It is not a model of a web service or every real workload.

## Recommendation

**Historical — do not introduce heap ballast as the default GC tuning strategy in modern Go applications.**

Use this proof to understand:

- how GOGC-based heap targets react to live heap size;
- why ballast historically reduced GC frequency;
- why "a mechanism works" is different from "the mechanism should be recommended today."

For modern tuning, prefer supported runtime controls and validate them with representative service workloads.

## Related Handbook Chapters

- [Garbage Collector](../../../docs/04-memory-and-gc/garbage-collector.md)
- [GC Pacing](../../../docs/04-memory-and-gc/gc-pacing.md)
- [Memory Limits, RSS, and Scavenging](../../../docs/04-memory-and-gc/memory-limits.md)

## Primary Sources

- Go GC guide: https://go.dev/doc/gc-guide
- `runtime/debug.SetGCPercent`: https://pkg.go.dev/runtime/debug#SetGCPercent
- `runtime/debug.SetMemoryLimit`: https://pkg.go.dev/runtime/debug#SetMemoryLimit
- Go 1.19 memory-limit release notes: https://go.dev/doc/go1.19
