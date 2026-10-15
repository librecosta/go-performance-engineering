# Cache Hierarchy

## 1. Why Caches Exist

CPU cores execute instructions much faster than DRAM can provide data.

Without caches, processors would spend much of their time waiting for memory.

Modern systems therefore use a hierarchy:

```text
Registers
   ↓
L1 cache
   ↓
L2 cache
   ↓
Last-level cache
   ↓
DRAM
```

The exact sizes and latencies vary by architecture and CPU generation.

The engineering lesson is more important than the exact numbers:

> Accessing data already close to the core is much cheaper than fetching it from distant memory.

---

## 2. Cache Lines

Caches do not normally transfer individual Go fields.

They move data in fixed-size blocks called cache lines.

On common server CPUs, a cache line is often 64 bytes.

If a program accesses:

```go
x := obj.Counter
```

the processor may fetch a full line containing `Counter` and neighboring data.

This has two consequences.

### Positive

Nearby useful data may already arrive for free.

### Negative

Unrelated data that shares the same line can interact through coherence.

---

## 3. L1, L2, and LLC

### L1

L1 is small and very fast.

A hot working set that fits in L1 can execute extremely efficiently.

### L2

L2 is larger and slower, but still much cheaper than DRAM.

### Last-Level Cache

The last-level cache is often shared among multiple cores or core groups.

It can reduce DRAM traffic, but shared capacity also creates competition between workloads.

---

## 4. Cache Hits and Misses

A cache hit means the required line is already present in the relevant cache.

A miss requires fetching it from a lower level.

The cost of a miss depends on where the data is eventually found.

Conceptually:

```text
L1 miss
 ↓
L2?
 ↓
LLC?
 ↓
DRAM?
```

The same Go load instruction may therefore have very different latency.

---

## 5. Spatial Locality

Spatial locality means nearby data tends to be accessed together.

Example:

```go
type Point struct {
    X float64
    Y float64
}

points := []Point{...}
```

Iterating through `points` accesses contiguous memory.

One cache-line fetch can bring several `Point` values close to the CPU.

This is one reason slices of values can be very efficient.

---

## 6. Temporal Locality

Temporal locality means recently accessed data is likely to be used again.

Example:

```go
state := lookup(id)

processHeader(state)
processBody(state)
updateMetrics(state)
```

If these operations happen close together, the relevant state may remain hot in cache.

If the program scans a very large unrelated data set between accesses, the state may be evicted.

---

## 7. Working Set

The working set is the data actively needed by a computation.

A smaller working set has a better chance of fitting in faster cache levels.

This leads to several important techniques:

- compact data structures;
- hot/cold splitting;
- avoiding unnecessary metadata in hot paths;
- processing data in cache-friendly batches.

The objective is not always minimum total memory.

The objective may be minimum **hot** memory.

---

## 8. Cache Coherence

Each CPU core can have private cache copies.

When multiple cores access shared memory, the processor must maintain a coherent view.

Reads can often share a line efficiently.

Writes require stronger coordination.

A core that modifies a line generally needs ownership that prevents other cores from simultaneously modifying stale copies.

This creates cache-line movement.

---

## 9. True Sharing

True sharing occurs when multiple cores genuinely modify the same data.

Example:

```go
counter.Add(1)
```

from many goroutines.

All cores compete for the cache line containing the counter.

No amount of padding around the counter can eliminate the fundamental sharing.

The solution is often architectural:

- sharding;
- batching;
- local accumulation;
- single-writer aggregation.

---

## 10. False Sharing

False sharing occurs when cores modify different variables that happen to occupy the same cache line.

Example:

```go
type Counters struct {
    A atomic.Uint64
    B atomic.Uint64
}
```

If separate cores repeatedly modify `A` and `B`, the whole line may bounce between cores even though the variables are logically independent.

False sharing is a physical-layout problem.

Potential fixes include:

- separating fields;
- cache-line padding;
- redesigning shard layout.

---

## 11. Cache Pollution

Fetching data into cache can evict other useful data.

Large streaming operations can therefore reduce the effective cache available to hot control structures.

This is why hot/cold separation and working-set design matter.

It also explains why low-level runtimes sometimes use specialized streaming memory instructions for large transfers.

Application code should normally prefer standard library/runtime primitives rather than reimplementing these details.

---

## 12. Caches and Go Data Structures

Different Go representations create different cache behavior.

### Slice of Values

```go
[]T
```

usually stores values contiguously.

### Slice of Pointers

```go
[]*T
```

stores pointers contiguously, but the pointed-to objects may be scattered.

### Map

Map access is less sequential and may involve multiple memory locations.

### Linked Structures

Linked lists and pointer-heavy trees can create dependent random loads.

No structure is always superior.

The correct choice depends on access pattern and semantics.

---

## 13. Cache Optimization Is Access-Pattern Optimization

A common mistake is to optimize object size while ignoring how data is accessed.

A smaller object with random access may still perform worse than a larger contiguous structure.

The key questions are:

- Which fields are actually hot?
- Are objects accessed sequentially?
- How many cache lines are touched per operation?
- Are multiple cores writing nearby fields?
- Is the working set larger than the useful cache capacity?

These questions connect source-level design to hardware behavior.
