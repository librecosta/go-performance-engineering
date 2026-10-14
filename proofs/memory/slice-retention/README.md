# Slice Retention — Proof 001

Status: implementation and development checks only. Formal 100 MiB measurements,
repeated benchmarks, and comparison across environments are pending. No formal
results are claimed here.

## Claim

A small slice can keep a much larger backing array reachable. A full slice
expression such as `large[:100:100]` limits capacity but still refers to that
array. Copying the prefix into independent storage permits the source array to
be collected **only when no other references retain it**.

This controlled proof combines source reasoning (L0), post-GC heap metrics and
retained heap profiles (L1), and a local operation benchmark (L2). It does not
establish a production speedup or recommend copying every slice.

## Cost Model

| Operation | Storage relationship | Local work | Potential long-lived storage |
| --- | --- | --- | --- |
| `ViewPrefix(src, 100)` | Shares source | Slice descriptor | Entire source array |
| `LimitedViewPrefix(src, 100)` | Shares source, capacity 100 | Slice descriptor | Entire source array |
| `CopyPrefix(src, 100)` | Independent prefix | Allocation and 100-byte copy | Small result, if source has no other owner |

Length and capacity describe operations available through a slice; they do not
measure the allocation kept reachable by its data pointer. Limiting capacity
forces a later append beyond that capacity to use new storage, but does not
detach the original view.

`HeapAlloc` measures allocated heap object bytes; after explicit GC it is useful
for this reachability comparison. `HeapInuse` includes space in occupied spans
and can exceed live object bytes. Benchmark `B/op` measures allocation traffic
per operation, not retained heap. None of these is process RSS.

## Baseline

Use one 100 MiB source (`100 * 1024 * 1024` bytes) and a 100-byte prefix. Every
source byte is initialized with `byte(i*31 + 7)`. All modes return the same data.
The baseline implementation is `src[:100]`; compare capacity-limited slicing and
copying with the same input and normal GC configuration.

The CLI's `baseline` snapshot is a separate concept: it is the process heap
before constructing the source. Compare each process's `retained - baseline`
delta, then inspect the `released` snapshot. Do not compare raw process totals
as if unrelated runtime allocations were identical.

## Experiment

Run commands from this directory. The independent module uses only the standard
library and requires Go 1.24 or newer for `testing.B.Loop`; it does not force a
toolchain upgrade.

Experiments A and B run **one mode per process**. The CLI defaults to 100 MiB and
100 retained bytes; `-size-mib` accepts 1..128, and `-keep-bytes` accepts
1..source size. `-mode` accepts `view`, `cap`, or `copy`. Invalid arguments or
profile write failures exit nonzero. Profile parent directories must exist;
existing profile files at the chosen path are overwritten.

The lifecycle is deliberately explicit:

1. Collect a baseline snapshot before allocating the source.
2. Allocate, fully initialize, and derive the result inside `makeResult`.
   Its non-inlined frame returns only the result slice, leaving no separate
   source reference in the measurement function.
3. Store the result in `retainedRoot`, then collect the retained snapshot.
4. Write the retained heap profile while that root is still reachable. Consume
   the result through a checksum and call `runtime.KeepAlive` afterward.
5. Clear the root and collect the released snapshot. Encode JSON only afterward.

Each snapshot runs two explicit GCs before `runtime.ReadMemStats`, allowing for
the [memory profile's documented GC-cycle lag](https://pkg.go.dev/runtime#MemProfile).
The program leaves GC settings and `runtime.MemProfileRate` unchanged and reports
the sampling rate. These are quiescent observations, not production GC timing.

For later **manual formal runs**, build once and launch each mode separately.
These PowerShell commands use the default formal sizes explicitly:

```powershell
go build -o ./results/retention.exe ./cmd/retention
./results/retention.exe -mode view -size-mib 100 -keep-bytes 100 -profile ./results/view.heap.pprof > ./results/view.json
./results/retention.exe -mode cap -size-mib 100 -keep-bytes 100 -profile ./results/cap.heap.pprof > ./results/cap.json
./results/retention.exe -mode copy -size-mib 100 -keep-bytes 100 -profile ./results/copy.heap.pprof > ./results/copy.json
```

On other systems, use `./results/retention` as the executable name and path.
Use distinct result directories or filenames for each environment and repetition.

## Verification

Development checks use a small source and are not formal measurement results:

```powershell
go test ./...
go build -o ./results/retention.exe ./cmd/retention
./results/retention.exe -mode view -size-mib 8 -profile ./results/smoke-view.heap.pprof
./results/retention.exe -mode cap -size-mib 8 -profile ./results/smoke-cap.heap.pprof
./results/retention.exe -mode copy -size-mib 8 -profile ./results/smoke-copy.heap.pprof
go tool pprof -top -inuse_space ./results/smoke-view.heap.pprof
go tool pprof -top -inuse_space ./results/smoke-cap.heap.pprof
go tool pprof -top -inuse_space ./results/smoke-copy.heap.pprof
```

Unit tests verify content, mutation sharing in both directions, copy
independence, full-slice capacity, append detachment, and slicing boundaries.
The mutation tests show that a limited view still aliases the source; they do
not assert heap thresholds or infer reclamation from capacity alone. Execution
is sequential; neither the CLI root nor benchmark sinks are used concurrently.

For the formal profiles, inspect:

```powershell
go tool pprof -top -inuse_space ./results/view.heap.pprof
go tool pprof -top -inuse_space ./results/cap.heap.pprof
go tool pprof -top -inuse_space ./results/copy.heap.pprof
```

The hypothesis is a source-sized retained heap delta in view and cap modes,
with the large allocation attributed to `main.makeResult`; copy should no
longer retain that large allocation. Released snapshots should no longer show
source-sized retention. Check these trends against the actual data, not exact
byte thresholds. The sampled 100-byte copy may not appear in a profile at all.
Profile stacks identify allocation sites, not retaining reference paths; use
them together with the controlled root and heap measurements.

If evidence disagrees, inspect extra references, compiler liveness, sampling,
and the recorded environment before interpreting it. An absent profile sample
alone does not prove that an object was collected.

## Benchmark

Experiment C uses one shared, initialized source outside all timed loops.
`BenchmarkPrefix/View`, `/LimitedView`, and `/Copy` each use `for b.Loop()` and
`b.ReportAllocs()`. All publish to the same slice sink and consume the same two
bytes. The source stays alive through `runtime.KeepAlive`; each result sink is
cleared after its loop. Setup and cleanup are excluded by
[`B.Loop` timer semantics](https://pkg.go.dev/testing#B.Loop).

For a brief development check only:

```powershell
go test -run '^$' -bench '^BenchmarkPrefix$' -benchmem -benchtime=20ms -count=1 . -args -bench-size-mib=8
```

For later manual formal measurements:

```powershell
go test -run '^$' -bench '^BenchmarkPrefix$' -benchmem -benchtime=1s -count=5 . -args -bench-size-mib=100 > ./results/benchmark.txt
```

Report `ns/op`, `B/op`, and `allocs/op` separately from retained-heap evidence.
This is a repeated hot-prefix workload, including common sink and byte-read
overhead, which can dominate tiny view operations. The benchmark deliberately
keeps the source alive and therefore cannot measure its reclamation. Repeated
copy allocations contribute to allocation traffic even though old copies die.

## Environment

Record the commit, date, machine/CPU model, OS/kernel, Go version, GOOS, GOARCH,
GOMAXPROCS, available CPUs, build flags/PGO state, input sizes, commands, and
GC-related configuration. The CLI includes runtime fields, its arguments,
sampling rate, and process environment values in its single-line JSON. Empty
environment values mean unset/empty, not a numeric GC setting. CPU model and
build configuration need separate records; see the [results template](results/README.md).

Keep normal GC settings and comparable builds between modes. Do not disable
GC, change memory limits to manufacture a difference, or compare instrumented
timings with normal binaries.

## Caveats

- Copy still constructs the entire source first; peak lifetime memory is not
  reduced to 100 bytes. During copying both source and destination exist.
- If another owner retains the source, copying cannot make it collectible.
- Profiling allocates memory after the retained snapshot. Its caches and
  runtime bookkeeping can affect the released snapshot; exact equality with
  baseline is not required. JSON formatting happens after all snapshots.
- Heap profiles are sampled estimates. Size classes, runtime background work,
  compiler versions, and platforms affect exact counts and attribution.
- `[]byte` is pointer-free backing storage; retained bytes do not directly
  quantify GC scanning work or GC CPU savings.
- Heap reclamation does not guarantee an immediate RSS reduction. RSS is
  optional context and is not an acceptance criterion.
- Synthetic microbenchmarks do not establish application latency, throughput,
  or the best ownership policy for a real workload.

## Recommendation

Conditional: consider an intentional small copy when a long-lived prefix is
otherwise the sole owner keeping a much larger temporary source alive. Prefer
a view when sharing and lifetimes are appropriate. Use capacity limiting for
append isolation, not as a memory-release mechanism. Validate the ownership
trade-off in the actual application before adopting it.

Related handbook material: [Retention](../../../docs/zh-CN/04-memory-and-gc/retention.md),
[Heap Model](../../../docs/zh-CN/04-memory-and-gc/heap-model.md),
[Benchmarking](../../../docs/zh-CN/06-methodology/benchmarking.md), and
[Evidence Model](../../../docs/zh-CN/06-methodology/evidence-model.md).
