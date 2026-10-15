# Performance Engineering

## 1. Performance Engineering Is Not a Bag of Tricks

Performance optimization is often presented as a collection of local transformations:

- replace one allocation;
- remove one branch;
- add a pool;
- use an atomic;
- avoid a copy.

Those techniques can be useful, but they are not performance engineering by themselves.

Performance engineering is the discipline of understanding **where a system spends resources, why it spends them, and which trade-off produces the best measurable outcome**.

The distinction matters because many local optimizations merely move cost elsewhere.

For example:

```text
Zero-copy
  ↓
less memcpy
less allocation
  ↓
more ownership coupling
more lifetime constraints
possible memory retention
```

or:

```text
Lock-free algorithm
  ↓
less blocking
  ↓
more retries
more cache-line contention
more algorithmic complexity
```

An optimization is only useful if the total system trade-off is favorable.

---

## 2. Performance Is a System Property

A Go program does not execute directly from source code.

Its performance emerges from a stack of interacting layers:

```text
Application behavior
        ↓
Architecture and data model
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

A symptom observed in one layer may be caused by another.

### Example: GC CPU

A CPU profile may show:

```text
runtime.scanobject
runtime.gcBgMarkWorker
```

It is tempting to conclude:

> The garbage collector is slow.

But the actual cause may be:

- a larger live heap;
- a higher allocation rate;
- pointer-heavy data structures;
- a memory limit forcing aggressive collection.

The runtime function is where the cost is visible, not necessarily where the problem originates.

### Example: Atomic Hotspot

A profile may show an atomic increment as hot.

The root cause may not be the atomic implementation itself.

It may be:

```text
many cores
   ↓
one shared counter
   ↓
one cache line
   ↓
coherence traffic
```

The right optimization may therefore be sharding or local accumulation rather than replacing the atomic primitive.

---

## 3. Define Performance Before Optimizing

"Make it faster" is not a complete objective.

Performance may mean:

- higher throughput;
- lower median latency;
- lower tail latency;
- less CPU per request;
- lower RSS;
- lower allocation rate;
- better scalability with more cores;
- lower infrastructure cost.

These goals can conflict.

For example:

```text
larger cache
  ↓
higher hit rate
  ↓
lower CPU / latency
  ↓
higher RSS
```

or:

```text
larger GOGC
  ↓
fewer GC cycles
  ↓
lower GC CPU
  ↓
higher heap footprint
```

The engineer must know which dimension matters most.

---

## 4. Throughput, Latency, and Efficiency

### Throughput

Throughput measures how much work is completed per unit time.

Examples:

```text
requests / second
operations / second
MB / second
```

Throughput is often limited by the most saturated resource:

- CPU;
- memory bandwidth;
- lock contention;
- database;
- network.

### Latency

Latency measures how long one operation takes.

Tail latency matters because systems rarely behave uniformly.

A service may have:

```text
P50  = 5 ms
P99  = 40 ms
P99.9 = 200 ms
```

The average does not explain the rare expensive paths.

### Efficiency

Efficiency asks how much resource is consumed for a unit of useful work.

Examples:

```text
CPU seconds / request
bytes allocated / operation
memory / connection
```

A throughput increase that requires twice as much CPU may not be an improvement.

---

## 5. Bottlenecks Are Workload-Dependent

A bottleneck is not an intrinsic property of a function.

It is a relationship between code and workload.

Consider a lock:

```go
mu.Lock()
critical()
mu.Unlock()
```

Under one worker:

```text
no contention
```

Under 64 workers:

```text
heavy contention
```

The same code can move from negligible to dominant.

The same applies to:

- branch prediction;
- cache behavior;
- allocation;
- GC;
- channel buffering;
- memory bandwidth.

Therefore performance conclusions should always include workload assumptions.

---

## 6. Optimization Changes Resource Allocation

Most optimizations move cost between dimensions.

### CPU ↔ Memory

Caching, pooling, and larger GC targets often spend memory to save CPU.

### Memory ↔ Complexity

Compact layouts may save memory but increase indexing or conversion complexity.

### Throughput ↔ Tail Latency

Batching may improve throughput but increase waiting time for individual operations.

### Safety ↔ Control

Unsafe zero-copy can remove copying but transfers lifetime correctness from the language to the programmer.

### Simplicity ↔ Specialized Performance

A generic API may be easier to maintain, while a specialized representation may be faster in one hot path.

The correct choice depends on system priorities.

---

## 7. Amdahl's Law in Practice

An optimization cannot save more time than the fraction of total cost it affects.

If a function consumes 1% of total CPU, making it infinitely fast saves at most approximately 1% of CPU.

This is why profile-guided work matters.

A 10× microbenchmark improvement can still be irrelevant at system level.

Performance engineering should therefore estimate:

```text
hotness
×
possible speedup
=
maximum system gain
```

before accepting complexity.

---

## 8. Local Improvement vs System Improvement

A local benchmark answers:

> Is this implementation faster under this test?

A system benchmark answers:

> Does this change improve the application?

Both matter.

Example:

```text
unsafe bytes→string conversion
microbenchmark: -90% time
```

But if conversion accounts for only 0.2% of request CPU:

```text
system benefit ≈ negligible
```

The local result is valid, but the engineering decision may still be to reject the change.

---

## 9. Evidence-Driven Engineering

A mature performance investigation follows a chain:

```text
Symptom
  ↓
Measurement
  ↓
Cost model
  ↓
Hypothesis
  ↓
Candidate change
  ↓
A/B validation
  ↓
System validation
```

Each step protects against a different failure mode.

### Without Measurement

You optimize a non-bottleneck.

### Without a Cost Model

You may improve the benchmark without understanding why.

### Without A/B Validation

You cannot separate signal from noise.

### Without System Validation

You may optimize a component while degrading the service.

---

## 10. Performance Engineering and Maintainability

Optimized code often looks unusual.

Examples:

```go
_ = b[7]
```

may exist only to give the compiler a dominating bounds proof.

A deliberate copy:

```go
out := append([]byte(nil), input[:n]...)
```

may exist to avoid retaining a large backing array.

Padding may exist only to separate cache lines.

These patterns are vulnerable to "cleanup" refactors.

A performance optimization is therefore incomplete if the reason for the unusual code cannot be recovered.

Important optimizations should preserve intent through:

- comments;
- benchmarks;
- tests;
- proof examples;
- design documentation.

Performance is not separate from maintainability.

---

## 11. Performance Engineering as a Feedback Loop

Performance work is not a one-time phase.

Systems change:

- traffic changes;
- Go versions change;
- CPUs change;
- data sets grow;
- architecture changes.

A healthy process is cyclical:

```text
observe
 ↓
measure
 ↓
understand
 ↓
change
 ↓
validate
 ↓
monitor
 ↓
repeat
```

The objective is not to produce permanently "optimized code".

The objective is to maintain a system whose costs are understood and controlled.
