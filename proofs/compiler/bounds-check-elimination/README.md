# Bounds Check Elimination

Status: Go 1.26.9 mechanism diagnosis and development checks only. Formal
Windows/Linux measurements, a performance report, and a blog are not part of
this implementation.

## Claim

An explicit length guard or an earlier successful access to the highest needed
index can establish bounds for later reads. This proof compares the remaining
checks in three four-byte decoders without changing Go's bounds-safety rules.

The evidence layers are distinct: correctness tests establish the input
contract, compiler diagnostics show remaining SSA checks, and a microbenchmark
observes local cost. None alone establishes a production performance benefit.

## Cost Model

An index access must panic if the index is outside the slice's length. A
redundant check can disappear when prior control flow proves the access valid.
After a successful `b[3]` read, indices 0 through 2 are valid too; a successful
`b[0]` read alone does not prove that index 3 is valid.

Remaining checks may introduce comparisons and panic paths. Their diagnostic
count is not necessarily the number of final machine-code branches, and is not
a prediction of elapsed time.

## Baseline

All three functions decode the first four bytes as a little-endian `uint32` and
ignore trailing bytes on valid input. For example, `12 34 56 78` produces
`0x78563412`.

| Function | Source shape | Input shorter than four bytes |
| --- | --- | --- |
| `DecodeGuarded` | Explicit length guard, then reads 0 through 3 | Returns `(0, false)` |
| `DecodeForward` | Reads 0, 1, 2, 3 | Panics |
| `DecodeHighFirst` | Reads 3, 2, 1, 0 | Panics |

Forward and HighFirst agree on valid inputs, but may panic at different indices
with different messages on invalid inputs. Guarded has a different failure
contract and is a mechanism control, not a drop-in equivalent.

## Experiment

- **A — Guarded decode:** determine whether the length guard makes the four
  following index checks redundant. The explicit guard itself still matters.
- **B — Read ordering:** compare Forward and HighFirst with identical bytes,
  shifts, and OR operations, changing only the read order.
- **C — Local cost:** after observing a diagnostic difference in B, compare the
  two orderings with symmetric benchmarks on valid input.

All three functions use `//go:noinline` to isolate their own bounds proofs from
caller-specific length knowledge. This is an experimental control, not a
production optimization recommendation. No global optimization disabling,
unsafe access, or extra artificial memory reads are used.

## Verification

Use the shared [Proof module](../../go.mod). From the repository root, in
PowerShell:

```powershell
cd proofs
$env:GOTOOLCHAIN = 'go1.26.9'
go version
go build -gcflags='-d=ssa/check_bce/debug=1' ./compiler/bounds-check-elimination
```

Observed during implementation with `go1.26.9 windows/amd64`, `GOAMD64=v1`,
and empty `GOFLAGS` / `GOEXPERIMENT`:

| Function | Remaining `IsInBounds` diagnostics | Reported accesses |
| --- | ---: | --- |
| `DecodeGuarded` | 0 | None after the explicit guard |
| `DecodeForward` | 4 | `b[0]`, `b[1]`, `b[2]`, `b[3]` |
| `DecodeHighFirst` | 1 | `b[3]` |

The five diagnostics refer to [bce.go](bce.go) lines 26–29 and 40. No
`IsSliceInBounds` diagnostics were reported. These are observations for this
source and toolchain, not fixed assertions in tests or CI.

`Found IsInBounds` and `Found IsSliceInBounds` mean a check **remains** at the
SSA diagnostic phase; they do not mean a check was eliminated. Guarded's zero
reported index checks does not mean it has no conditional check: its length
guard still selects success or failure. HighFirst's first access supplies the
bound for its subsequent lower indices, so the observed difference supports
the read-order mechanism without a fallback case.

Only count diagnostics for these target functions. If a future compiler changes
the result, record and explain it instead of requiring these counts to stay
constant. If needed, inspect only this package with
`go build -gcflags='-S' ./compiler/bounds-check-elimination`; keep any captured
output in a system temporary directory, outside the repository.

Necessary correctness checks from `proofs/`:

```powershell
go vet ./...
go test -count=1 ./...
```

Table-driven tests use explicit expected values for representative legal inputs
and trailing bytes. They cover nil and lengths 0 through 3, including a short
slice with spare capacity. Short-input tests require Guarded failure and a
panic from each unguarded decoder without matching panic text. They do not
assert timing, allocation counts, or compiler-internal check counts.

## Benchmark

`BenchmarkDecodeForward` and `BenchmarkDecodeHighFirst` use the same four-byte
input, `b.Loop()`, `b.ReportAllocs()`, and a `uint64` sum published to the same
package-level sink after each loop. Setup is outside the timed loop. Calls stay
out of line, and the accumulated return values remain observable. Guarded is
excluded from the timing comparison because its failure contract differs.

One short development smoke run, from `proofs/`:

```powershell
go test -run '^$' -bench '^BenchmarkDecode(Forward|HighFirst)$' -benchmem -benchtime=20ms -count=1 ./compiler/bounds-check-elimination
```

For later manual measurement on Go 1.26.9 / Windows 10 only (not executed as
part of implementation):

```powershell
go test -run '^$' -bench '^BenchmarkDecode(Forward|HighFirst)$' -benchmem -benchtime=1s -count=7 ./compiler/bounds-check-elimination
```

Record `ns/op`, `B/op`, and `allocs/op` separately. No stable timing advantage is
claimed from the smoke run. Linux reproduction remains a later decision.
Keep raw output outside the Git working tree; no formal report is created here.

## Environment

For later measurements record the source commit, CPU model, OS/version,
`go version`, `GOOS`, `GOARCH`, `GOAMD64`, `GOMAXPROCS`, `GOFLAGS`,
`GOEXPERIMENT`, PGO/build options, exact commands, and sample count. The default
toolchain is Go 1.26.9; the existing CI also verifies correctness on Go 1.27.2.
CI does not enforce compiler diagnostic counts or run these benchmarks.

## Caveats

- Compiler versions and targets can change BCE, instruction scheduling, and
  final code generation. Re-run diagnostics before attributing later results.
- A stable timing difference still does not isolate BCE as its sole cause:
  branch prediction, code layout, instruction ordering, and function-call
  overhead can contribute. An unstable difference is not a speedup.
- The input is a repeated hot four-byte prefix. Results do not establish
  service throughput, P99 latency, or gains for general parsers.
- Disabling inlining excludes optimizations that callers may enable in real
  code. Invalid-input panic behavior is not fully equivalent across orderings.

## Recommendation

Prefer clear input contracts and readable guards. Investigate BCE only when
profiling identifies a relevant hot path, then validate the compiler evidence
and application impact. Do not routinely reverse indexed reads for a possible
microbenchmark gain.

## Primary Sources

- [Go 1.26.9 checkbce implementation](https://github.com/golang/go/blob/go1.26.9/src/cmd/compile/internal/ssa/checkbce.go)
- [Go 1.26.9 BCE regression tests](https://github.com/golang/go/blob/go1.26.9/test/checkbce.go)
- [Go 1.26.9 prove regression tests](https://github.com/golang/go/blob/go1.26.9/test/prove.go)
- [Go specification: index expressions](https://go.dev/ref/spec#Index_expressions)
- [Compiler Handbook](../../../docs/03-compiler/bounds-check-elimination.md)
- [Proof Guidelines](../../README.md)
