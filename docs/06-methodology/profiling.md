# Profiling

## 1. What Profiling Is For

Profiling answers:

> Where is the program spending a resource?

Depending on the profile, that resource may be:

- CPU time;
- live heap;
- cumulative allocation;
- blocked time;
- mutex contention;
- goroutines;
- operating-system threads.

Profiling is primarily a **localization tool**.

It narrows a large system into a small set of functions, call paths, objects, or synchronization points that deserve investigation.

A profile does not automatically tell you:

- why the cost exists;
- whether it is avoidable;
- which replacement is faster;
- whether a proposed optimization improves the system.

Those questions require a cost model and controlled comparison.

---

## 2. Profiling vs Benchmarking

The distinction is foundational:

```text
Profile
→ Where is the cost?

Benchmark
→ Which implementation is better?
```

A profile can show:

```text
runtime.scanobject = 18% CPU
```

but it cannot by itself prove that:

```text
GC should be tuned
```

The root cause may be:

- pointer-heavy data structures;
- high allocation rate;
- unusually large live heap.

Likewise, a benchmark can show that one parser is 20% faster in isolation without proving that parsing is an important system bottleneck.

Strong performance engineering uses the two together.

---

## 3. Start from a Symptom

Do not collect every diagnostic artifact merely because tools are available.

Start from the observed problem.

Examples:

```text
CPU saturation
→ CPU profile

RSS growth
→ heap + process memory investigation

high allocation rate
→ allocs profile

lock contention
→ mutex profile

waiting/blocking
→ block profile

latency mystery
→ execution trace

goroutine growth
→ goroutine / goroutineleak profile
```

Use the lowest-cost tool that can distinguish plausible causes.

---

## 4. CPU Profile

A CPU profile samples program execution while the process is running.

The result approximates where CPU time is being spent.

Typical sources include:

- `runtime/pprof`;
- `net/http/pprof`;
- `go test` CPU profiling.

For an HTTP-exposed profile:

```bash
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

The exact collection duration should reflect workload stability.

---

## 5. Sampling Means Approximation

CPU profiles are sampled.

They do not record every instruction.

Therefore small differences can be statistical noise.

A profile is excellent for identifying dominant CPU consumers.

It is weaker for proving that two nearly identical implementations differ by 1%.

Use benchmarks for that comparison.

---

## 6. Flat vs Cumulative CPU

pprof commonly exposes two important views.

### Flat

Time attributed directly to a function.

Example:

```text
flat = 2.0s
```

means samples were observed executing inside that function itself.

### Cumulative

Time attributed to the function plus callees beneath it.

Example:

```text
cum = 8.0s
```

can indicate:

```text
function does little directly
but calls expensive work
```

Both views matter.

---

## 7. Interpreting Flat Time

High flat CPU may indicate:

- tight computation;
- hashing;
- encoding;
- atomic retry;
- runtime scanning;
- copying.

But a function name is not a root cause.

Example:

```text
memmove
```

can mean:

- slice growth;
- string conversion;
- serialization;
- buffer assembly.

Follow call paths.

---

## 8. Interpreting Cumulative Time

High cumulative time is useful for architecture-level localization.

Suppose:

```text
Handler.Process
  cum 45%
```

but the flat cost is tiny.

The function is an entry point into expensive work.

Inspect descendants before rewriting the wrapper.

---

## 9. Call Graphs

Call graphs reveal how cost reaches a hotspot.

A useful investigation asks:

```text
who calls this?
↓
under what workload?
↓
why does it call so often?
```

Optimizing the callee may be less valuable than reducing the number of calls.

---

## 10. CPU Profile and Runtime Functions

Runtime functions often appear prominently.

Examples:

- GC work;
- allocation;
- map operations;
- synchronization;
- scheduler activity.

Do not conclude:

> The runtime is the problem.

Translate runtime cost back into application behavior.

Examples:

```text
GC CPU
→ allocation / pointer graph / live heap

mutex runtime cost
→ contention topology

map runtime cost
→ access frequency / representation
```

---

## 11. Heap Profile

The heap profile is primarily useful for live memory.

Current Go `runtime/pprof` documentation describes it as sampling allocation sites for live heap objects, with the default pprof view focused on live bytes.

This answers:

> What allocation sites are responsible for memory that remains live?

This is a retention question.

---

## 12. Allocs Profile

The `allocs` profile emphasizes cumulative allocation activity.

It answers:

> Which code paths have allocated the most over time?

This can reveal churn that does not appear in a live heap profile.

A function may allocate huge cumulative volume while retaining almost nothing.

---

## 13. Heap vs Allocs

Use them for different questions:

```text
Heap
→ What remains?

Allocs
→ What churns?
```

Example:

```text
temporary JSON buffers
```

may dominate allocs but disappear from heap.

A cache may dominate heap while barely allocating after warmup.

---

## 14. Bytes vs Objects

Memory profiles can be interpreted by:

- bytes;
- object counts.

Large byte-heavy allocations and tiny object-heavy allocation patterns have different costs.

Examples:

```text
few huge buffers
→ bandwidth / RSS / retention

millions of tiny nodes
→ allocator/object/GC graph overhead
```

Inspect both dimensions when needed.

---

## 15. Memory Profiles Are Sampled

Heap profiling uses sampling.

Small objects and low-frequency allocations may be underrepresented individually.

Profiles are more reliable for identifying large trends than for exact accounting.

Use benchmark allocation metrics for controlled per-operation counts.

---

## 16. Mutex Profile

The mutex profile tracks contention on mutex-like synchronization.

An important interpretation detail is that samples correspond to the holder that caused others to wait, commonly attributed around unlock/end-of-critical-section paths.

Therefore the profile helps answer:

> Which critical sections are causing other goroutines to wait?

It does not merely report the location of `Lock()` calls.

---

## 17. Contention Time Can Exceed Wall Time

Suppose one goroutine holds a lock for one second while five others wait.

The system experiences approximately five goroutine-seconds of blocked waiting.

Contention profiles can therefore report aggregate delay greater than elapsed wall time.

That is expected.

---

## 18. Mutex Sampling

Mutex profiling is event sampled.

Its overhead and fidelity depend on the configured sampling fraction.

Do not turn maximum-detail contention profiling on permanently without considering production overhead.

---

## 19. Block Profile

The block profile tracks time goroutines spend blocked on synchronization operations such as:

- mutexes;
- RWMutex;
- channels;
- select;
- WaitGroup;
- condition variables.

It answers a different question from CPU:

> Where are goroutines not making progress because they are waiting?

---

## 20. Block vs Mutex

Use:

```text
mutex profile
→ who holds contended locks

block profile
→ where goroutines block
```

For a lock problem, both can be useful.

The waiter location and holder location describe different sides of the contention.

---

## 21. Goroutine Profile

The goroutine profile provides stack traces for current goroutines.

Useful for:

- goroutine-count growth;
- widespread blocked states;
- duplicated background workers;
- stuck pipelines.

A snapshot is only one moment.

Compare over time for suspected leaks.

---

## 22. Go 1.27 `goroutineleak` Profile

Go 1.27 makes the `goroutineleak` pprof profile generally available.

It detects a class of permanently blocked goroutines using reachability reasoning.

Conceptually:

```text
G blocked on synchronization object P
↓
nothing runnable can ever reach/unblock P
↓
G is provably leaked
```

This is stronger than simply asking whether a goroutine has been blocked for a long time.

---

## 23. Goroutine Leak Detection Has Limits

The runtime cannot prove every goroutine leak.

For example, a blocking primitive reachable through global state may remain theoretically unblockable from the runtime's perspective even if the application will never actually do so.

Therefore:

```text
goroutineleak profile empty
```

does not prove:

```text
no goroutine leaks exist
```

It detects a valuable class of provable leaks.

---

## 24. Thread Creation Profile

The thread-create profile can reveal call paths associated with new OS thread creation.

It is less commonly needed than CPU/heap/block profiles, but can help with:

- cgo/native blocking;
- thread explosion;
- unusual scheduler interactions.

---

## 25. Execution Trace

Execution tracing records a timeline of runtime events.

It is especially useful when the problem is temporal:

```text
what happened before this latency spike?
why was this goroutine not running?
which goroutine woke this one?
when did GC/scheduling occur?
```

Profiles aggregate.

Trace preserves ordering.

---

## 26. What Trace Can Reveal

Execution trace can expose:

- goroutine scheduling;
- runnable vs running time;
- blocking;
- syscalls;
- synchronization;
- GC activity;
- user regions/tasks.

This makes it well suited for concurrency and latency investigations.

---

## 27. Trace Is Not a Better CPU Profile

Trace provides richer temporal information but also more data and cognitive cost.

If the question is simply:

> Which function burns CPU?

CPU pprof is usually easier.

Use trace when ordering and scheduling behavior matter.

---

## 28. Flight Recorder

Go 1.25 introduced `runtime/trace.FlightRecorder`.

It maintains a moving recent window of execution trace data.

This solves an important production problem:

```text
symptom detected now
but cause happened seconds earlier
```

The application can snapshot recent trace history after detecting an anomaly.

---

## 29. Why Flight Recording Matters

Traditional tracing often requires deciding to trace before the incident.

For rare P99/P99.9 events, that is difficult.

Flight recording reverses the workflow:

```text
keep bounded recent history
↓
detect slow/error event
↓
snapshot the preceding window
```

This is especially valuable for long-running services.

---

## 30. Flight Recorder Overhead and Memory

Flight recording buffers trace data.

Configuration must balance:

- useful history duration;
- maximum buffered bytes;
- runtime overhead.

Do not configure an unbounded diagnostic buffer.

---

## 31. User Regions and Tasks

Trace APIs can annotate application operations.

Useful labels can turn a raw scheduler timeline into domain context:

```text
request
parse
database stage
encode
```

Instrumentation should help answer real questions rather than annotate every function.

---

## 32. `runtime/metrics`

`runtime/metrics` provides a stable API surface for implementation-defined runtime metrics.

Useful categories can include:

- GC;
- scheduler;
- memory;
- CPU/runtime behavior.

It is appropriate for ongoing observability and hypothesis generation.

---

## 33. Metrics Are Not Profiles

A metric may show:

```text
GC CPU rising
```

but not the exact application allocation site.

Metrics tell you:

> Something changed.

Profiles tell you:

> Where the cost appears.

Use metrics to detect and profiles to localize.

---

## 34. Metric Names Can Evolve

`runtime/metrics` provides a stable access interface, while the set of available implementation-defined metrics may evolve.

Code should discover/validate relevant metric descriptions against the target Go version rather than assume every runtime exposes an identical set forever.

---

## 35. `GODEBUG=gctrace=1`

GC trace output is a useful low-cost diagnostic for understanding collection behavior.

It can reveal:

- GC cadence;
- heap sizes;
- timing;
- CPU contribution.

It is good for quick investigation.

For production observability, structured runtime metrics are often easier to aggregate.

---

## 36. Compiler Diagnostics Are Also Profiling Adjacent

Some suspected costs are compiler decisions:

- heap escape;
- remaining bounds checks;
- missed inlining.

Once a profile localizes a hot function, compiler diagnostics can answer:

> Why does generated code still contain this work?

This creates the chain:

```text
profile
→ hot source
→ compiler diagnostics
→ cost model
```

---

## 37. Labels in CPU Profiles

pprof labels can associate samples with logical work such as:

- tenant;
- endpoint;
- operation class.

This can help answer:

> Which workload causes this hot path?

Use labels carefully, especially when labels may contain sensitive or high-cardinality data.

---

## 38. Production Profiling

Production workloads are valuable because they contain real:

- traffic mix;
- data distribution;
- cache state;
- concurrency;
- dependencies.

But production profiling requires operational safeguards.

Consider:

- access control;
- profile duration;
- endpoint exposure;
- data sensitivity;
- overhead.

Never expose pprof endpoints publicly without protection.

---

## 39. Profile Representativeness

A profile is evidence only for the workload sampled.

A 30-second profile during:

```text
backup job
```

may not represent:

```text
normal request traffic
```

Record:

- timestamp;
- deployment version;
- workload condition;
- traffic level;
- Go version.

---

## 40. Incident Profile vs Normal Profile

Two useful profile types:

### Incident profile

Captures abnormal behavior.

Question:

> Why did this event happen?

### Baseline profile

Captures representative normal behavior.

Question:

> Where does the service normally spend resources?

Both are useful but should not be confused.

---

## 41. Tool Intrusion

Profiling and tracing have overhead.

The act of observing can change the system.

This is especially important for:

- high-rate block profiling;
- detailed traces;
- race detector;
- checkptr;
- external hardware tracing.

Always understand whether the tool materially changes the behavior under investigation.

---

## 42. Race Builds Are Not Performance Baselines

The race detector adds significant instrumentation.

Use it for correctness.

Do not compare:

```text
race-enabled benchmark
```

to normal production performance expectations.

The same applies to other heavy debug instrumentation.

---

## 43. `checkptr` Is Not a Performance Baseline

Pointer checking is valuable for unsafe correctness.

Its overhead makes it unsuitable as the primary performance baseline.

Separate:

```text
correctness validation
```

from:

```text
performance measurement
```

---

## 44. External Hardware Profiling

When pprof shows a hot loop but does not explain why it is slow, hardware PMU tools can reveal:

- cycles;
- instructions;
- cache misses;
- branch misses;
- stalled cycles;
- IPC.

This is an advanced layer.

Use it only when the hypothesis requires hardware evidence.

---

## 45. IPC

Instructions per cycle can help distinguish:

```text
CPU executing useful independent work
```

from:

```text
CPU frequently stalled
```

Low IPC alone is not a diagnosis.

It may result from:

- memory latency;
- dependencies;
- branches;
- synchronization.

Pair hardware counters with source-level hypotheses.

---

## 46. Cache-Miss Evidence

If a data-layout optimization claims to improve cache locality, a benchmark time improvement is useful.

Hardware cache counters can strengthen the mechanism evidence:

```text
same workload
↓
fewer cache misses
↓
higher throughput
```

This moves the claim from correlation toward mechanism confirmation.

---

## 47. NUMA

On multi-socket or NUMA systems, memory placement can affect access latency and bandwidth.

NUMA should be treated as an advanced diagnostic variable, not a default application optimization topic.

First rule out:

- algorithm;
- contention;
- data layout;
- ordinary cache behavior.

---

## 48. Profile Diffing

Comparing profiles before and after a change can show whether cost moved.

Example:

```text
JSON parsing CPU ↓
GC CPU ↑
```

A local optimization may merely relocate cost.

Always inspect the whole profile, not only the intended hotspot.

---

## 49. Profile-Driven Optimization Loop

A useful loop is:

```text
1. Define symptom
2. Collect appropriate profile
3. Identify dominant cost
4. Build mechanism hypothesis
5. Design candidate change
6. Benchmark A/B
7. Re-profile system
8. Verify cost actually moved as expected
```

This connects localization with causality.

---

## 50. Profile Quality Checklist

Before trusting a profile, ask:

- Is the workload representative?
- Is the profile long enough?
- Is the system warmed up?
- Is traffic stable?
- Is instrumentation overhead acceptable?
- Is this the correct profile type?
- Does the sampled version match the code being analyzed?

---

## 51. Related Official Sources

- `runtime/pprof`: https://pkg.go.dev/runtime/pprof
- `net/http/pprof`: https://pkg.go.dev/net/http/pprof
- `runtime/trace`: https://pkg.go.dev/runtime/trace
- Flight Recorder: https://go.dev/blog/flight-recorder
- Go 1.27 goroutine leak profile: https://go.dev/doc/go1.27
- `runtime/metrics`: https://pkg.go.dev/runtime/metrics
- Go diagnostics: https://go.dev/doc/diagnostics

---

## 52. Engineering Perspective

Profiling is successful when it replaces:

> I think this code looks expensive.

with:

> Under this workload, this call path accounts for this measured cost.

That is the point where optimization can begin.
