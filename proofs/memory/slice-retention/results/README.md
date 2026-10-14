# Results collection

Formal measurements are pending. This file is a recording template, not a
completed experiment or a table of expected numbers. Development smoke output
must not be presented as a formal 100 MiB result.

The proof-local `.gitignore` excludes generated files and directories under
`results/`, except this README. Keep binaries, heap profiles, raw JSON, and
temporary benchmark output local. Later, manually reviewed summaries can be
deliberately archived in a separate change.

## Manual run procedure

1. Record the environment below and use the same commit in both environments.
2. From the proof directory, build the CLI and run the three formal commands in
   the [proof README](../README.md), each in its own process, with a 100 MiB
   source and 100 retained bytes. Save the JSON and retained profiles.
3. Inspect each profile using `go tool pprof -top -inuse_space`. Preserve the
   text output alongside the raw profiles. Record large allocation attribution
   and compare baseline, retained, and released heap snapshots.
4. Run the documented 100 MiB benchmark with five samples per variant. Record
   timing and allocation data separately from retained heap data.
5. Repeat on the second environment. Use distinct filenames/directories per
   machine and run; profile writes overwrite an existing path.
6. Review all results, including anomalies and noise, before writing a summary.
   Do not infer a general performance improvement from this mechanism proof.

## Environment record

| Field | Observed value |
| --- | --- |
| Date and run identifier | Pending |
| Commit and working-tree status | Pending |
| Machine and CPU model | Pending |
| OS/kernel and power/background-load conditions | Pending |
| Go version, GOOS, GOARCH | Pending |
| GOMAXPROCS and available CPUs | Pending |
| GOGC, GOMEMLIMIT, GODEBUG | Pending |
| GOEXPERIMENT, GOFLAGS, PGO/build flags | Pending |
| MemProfileRate | Pending |
| Source bytes / retained bytes | Pending |
| Exact build, CLI, benchmark, and pprof commands | Pending |

Useful recording commands (not changes to configuration):

```powershell
git rev-parse HEAD
git status --short
go version
go env GOOS GOARCH GOEXPERIMENT GOFLAGS
Get-CimInstance Win32_Processor | Select-Object Name
Get-CimInstance Win32_OperatingSystem | Select-Object Caption, Version, BuildNumber
```

Use the corresponding system information commands on other operating systems.
The CLI JSON records runtime GOMAXPROCS and process GC environment values;
`go env` captures build configuration that may not be in the process environment.

## Evidence record

For every mode and repetition, preserve the raw JSON with integer byte counts:

- `source_bytes`, `keep_bytes`, `len`, `cap`, and `checksum`;
- `baseline`, `retained`, and `released`: `heap_alloc`, `heap_inuse`,
  and `heap_objects`;
- retained-minus-baseline and released-minus-baseline deltas (signed);
- profile path, sample type (`inuse_space`), and allocation-site observations.

Verify matching prefix checksums between modes. Do not use `cap == 100` or a
missing tiny profile sample as evidence of collection on its own.

For each benchmark sample preserve variant, iterations, `ns/op`, `B/op`, and
`allocs/op`. Label it as local operation cost with a live source, not retained
heap. Do not substitute `alloc_space` for `inuse_space` in retention analysis.

## Reviewed interpretation

- Supported or contradicted mechanism: Pending.
- Environment-specific evidence and repeatability: Pending.
- Copy cost versus retained-memory trade-off: Pending.
- Confounders, sampling limitations, and unresolved observations: Pending.
- Scope of any practical recommendation: Pending.
