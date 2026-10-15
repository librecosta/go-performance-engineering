# Memory Locality

## 1. Locality as a Performance Property

Memory locality describes how closely the program's access pattern matches the way modern memory systems deliver data.

Good locality allows the CPU to reuse:

- cache lines;
- page translations;
- prefetched data.

Poor locality increases:

- cache misses;
- TLB misses;
- latency;
- stalled execution.

---

## 2. Contiguous Data

Slices of values are a common source of good locality in Go.

```go
items := []Item{...}
```

The backing array stores elements next to each other.

A sequential loop:

```go
for i := range items {
    consume(items[i])
}
```

has a predictable address stream.

The CPU can often prefetch future lines while processing the current line.

---

## 3. Hardware Prefetch

Modern CPUs observe access patterns.

A regular sequence such as:

```text
address
address + stride
address + 2*stride
address + 3*stride
```

is easier to predict.

The hardware can request future cache lines before the program reaches them.

This hides part of memory latency.

The important engineering lesson is:

> Regular access patterns can make data sets much larger than L1 or L2 perform surprisingly well.

---

## 4. Memory-Level Parallelism

A processor can have multiple independent memory requests in flight.

For predictable independent loads:

```text
load A
load B
load C
```

the CPU may overlap waiting time.

But dependent pointer chains do not provide the same freedom.

---

## 5. Pointer Chasing

Consider:

```go
type Node struct {
    Value uint64
    Next  *Node
}
```

Traversing:

```go
for n != nil {
    use(n.Value)
    n = n.Next
}
```

creates a dependency:

```text
load Node0
 ↓
read Next0
 ↓
now Node1 address is known
 ↓
load Node1
```

The address of the next load may not be known until the previous load completes.

This makes prefetch and memory-level parallelism more difficult.

---

## 6. `[]T` vs `[]*T`

### `[]T`

Potential benefits:

- contiguous values;
- fewer allocations;
- fewer pointers;
- better cache locality;
- lower GC scanning in pointer-free cases.

Potential costs:

- copying large values;
- value movement when slice grows;
- identity semantics may be inconvenient.

### `[]*T`

Potential benefits:

- stable object identity;
- cheap pointer copies;
- useful for polymorphic or shared objects.

Potential costs:

- separate heap objects;
- pointer chasing;
- more GC-visible pointers;
- less predictable locality.

The correct choice depends on workload and semantics.

---

## 7. Flat Structures

A pointer-heavy tree can sometimes be represented as:

```go
type Node struct {
    Left  uint32
    Right uint32
    Value uint64
}

type Tree struct {
    Nodes []Node
}
```

The logical relationship is still a graph/tree, but storage becomes compact and contiguous.

Potential gains include:

- fewer heap objects;
- less pointer scanning;
- better locality;
- smaller working set.

The cost is more explicit index management.

---

## 8. Working-Set Locality

A hot loop may touch only a subset of a large object.

Example:

```go
type Connection struct {
    FD       int
    State    uint32
    Flags    uint32
    Username string
    Headers  map[string]string
    Debug    *DebugInfo
}
```

If packet processing touches only:

```text
FD
State
Flags
```

the other fields increase the physical footprint of the hot representation.

A hot/cold split can improve locality even if it introduces another pointer.

This illustrates an important rule:

> Fewer allocations are not always the same as better CPU locality.

---

## 9. Access Order Matters

Two algorithms with identical Big-O complexity can have different cache behavior.

Example:

```text
process all fields for object 1
process all fields for object 2
```

versus:

```text
process field A for all objects
process field B for all objects
```

The first favors object locality.

The second may favor field locality.

This is the basis of the AoS/SoA choice.

---

## 10. Sequential vs Random Access

Sequential access is often bandwidth-friendly.

Random access is often latency-sensitive.

This distinction matters because:

```text
sequential 1 GiB scan
```

can outperform:

```text
random 100 MiB pointer graph
```

despite touching more total data.

The amount of data is not the only cost.

The access pattern is often more important.

---

## 11. Locality and GC

Memory locality and GC behavior often reinforce each other.

A flat pointer-free representation may simultaneously:

- reduce cache misses;
- reduce pointer chasing;
- reduce object count;
- reduce scan work.

This is why data-oriented transformations can produce multi-layer improvements.

---

## 12. Locality Is Workload-Specific

A structure optimized for one traversal may be worse for another.

For example:

- SoA can be excellent for bulk numeric processing;
- AoS can be excellent when each object is processed as a whole.

Do not optimize layout from theory alone.

Measure the actual access pattern.
