# GC Pacing

## 1. Why a Pacer Is Needed

A tracing collector cannot wait until memory is completely exhausted and then begin work.

It must decide:

- when to start;
- how much CPU to spend;
- how to keep up with allocation;
- how much heap growth to allow.

This is the job of GC pacing.

---

## 2. The Heap-Growth Trade-Off

The collector balances:

```text
more heap headroom
→ fewer GC cycles
→ more memory

less heap headroom
→ more GC cycles
→ less memory
```

Go exposes this trade-off primarily through:

- GOGC;
- GOMEMLIMIT / `debug.SetMemoryLimit`.

---

## 3. GOGC

GOGC controls heap-growth proportionality.

A useful modern approximation is:

```text
Target heap =
Live heap +
(Live heap + GC roots) × GOGC / 100
```

The exact runtime implementation includes additional pacing details, but this model explains the trade-off well.

---

## 4. Example

Suppose:

```text
live heap = 2 GiB
roots ≈ 0.2 GiB
GOGC = 100
```

Approximate extra headroom:

```text
(2 + 0.2) GiB
```

Approximate target:

```text
4.2 GiB
```

The application can allocate roughly that headroom before the next cycle target is exhausted.

---

## 5. Higher GOGC

Increasing:

```text
GOGC 100 → 200
```

allows more heap growth between collections.

Potential benefits:

- lower GC frequency;
- lower GC CPU.

Potential costs:

- higher heap footprint;
- higher RSS;
- greater memory pressure.

This is a deliberate CPU-for-memory trade.

---

## 6. Lower GOGC

Lower GOGC reduces allowed growth.

Potential benefits:

- lower heap footprint.

Potential costs:

- more frequent collection;
- more GC CPU;
- more assist pressure.

A low GOGC can be harmful when the application already has high allocation rate.

---

## 7. Allocation Rate and Cycle Frequency

Suppose the application has:

```text
1 GiB heap headroom
```

and allocates:

```text
500 MiB/s
```

Very roughly, headroom is consumed in around two seconds.

If allocation rate rises to:

```text
2 GiB/s
```

the same headroom disappears much faster.

This is why GC frequency can spike while live heap remains unchanged.

---

## 8. Concurrent Marking

GC work is spread across time.

The pacer schedules background mark work while the application continues executing.

The goal is to complete marking before heap growth exceeds the target.

---

## 9. GC Assist Ratio

If the application allocates faster than background marking can keep up, allocation is charged against required GC work.

Allocating goroutines may perform mark assist.

This creates a feedback mechanism:

```text
higher allocation
→ more assist obligation
→ more mutator time spent marking
```

---

## 10. Assist and Latency

A request that allocates heavily may be forced to perform collector work.

This can create latency spikes that are not visible as long stop-the-world pauses.

Therefore high P99 with low pause time can still be GC-related.

---

## 11. GOMEMLIMIT

GOMEMLIMIT introduces an absolute soft memory budget for Go-managed runtime memory.

It changes pacing when normal GOGC growth would exceed the memory target.

Conceptually:

```text
normal GOGC pacing
       ↓
memory limit becomes binding
       ↓
collector becomes more aggressive
```

---

## 12. GOGC and GOMEMLIMIT Together

These controls solve different problems.

### GOGC

Controls normal proportional growth.

### GOMEMLIMIT

Provides a soft memory ceiling.

A healthy configuration often uses both:

```text
GOGC
→ normal CPU/memory trade

GOMEMLIMIT
→ upper operational budget
```

---

## 13. `GOGC=off` / `SetGCPercent(-1)` With a Limit

It is possible to disable ordinary GOGC-driven collection while keeping a memory limit.

Conceptually:

```text
normal proportional trigger disabled
        ↓
memory-limit pressure becomes dominant
```

This can minimize unnecessary collections when a fixed memory budget is available.

But it changes the workload behavior substantially and should be benchmarked carefully.

---

## 14. Memory-Limit Thrashing

If the true live set is very close to the memory limit:

```text
GC runs
↓
little memory reclaimed
↓
allocation resumes
↓
limit pressure returns
↓
GC runs again
```

This is GC thrashing.

The collector cannot reclaim genuinely live data.

Solutions include:

- reduce live set;
- increase memory limit;
- change representation.

More aggressive GC is not the answer.

---

## 15. Soft Limit

Go's memory limit is intentionally soft rather than an absolute process kill boundary.

The runtime prioritizes making progress over obeying an impossible memory target.

This avoids a pathological state where the program spends all CPU collecting memory that cannot be reclaimed.

---

## 16. Root Costs

Modern GOGC modeling includes root scanning costs.

Large goroutine stacks or large global root sets can affect pacing.

This is one reason live heap bytes alone do not fully predict GC work.

---

## 17. Pacing and Bursty Allocation

A bursty workload can surprise the pacer.

Example:

```text
steady allocation
↓
sudden batch allocates several GiB
```

The runtime may need aggressive assist/collection behavior during the burst.

Average allocation metrics can hide this.

---

## 18. Pacing and CPU Availability

GC workers compete with application goroutines for CPU.

On a saturated CPU-bound service, additional GC work directly reduces application throughput.

On a mostly idle service, background collection may have less visible effect.

The same heap behavior can therefore have different user-visible cost depending on CPU saturation.

---

## 19. What to Measure

When tuning pacing, observe together:

```text
GC cycle frequency
GC CPU
mark assist CPU
heap live
heap goal
RSS
allocation bytes/sec
P99 latency
```

Changing GOGC based only on GC count is insufficient.

---

## 20. Related Official Sources

- GC guide: https://go.dev/doc/gc-guide
- Runtime GC pacer: https://go.dev/src/runtime/mgcpacer.go
- `runtime/debug`: https://pkg.go.dev/runtime/debug

---

## 21. Engineering Perspective

GC pacing is a feedback-control problem.

The application supplies:

```text
allocation rate
live set
roots
CPU availability
```

The runtime responds with:

```text
heap target
background marking
assist work
```

Tuning is therefore meaningful only when those inputs are understood.
