# Data Layout

## 1. Logical Structure vs Physical Structure

A Go type expresses logical data relationships.

The CPU sees physical memory.

These views are related but not identical.

Example:

```go
type User struct {
    ID      uint64
    Active  bool
    Name    string
    Profile *Profile
}
```

The programmer sees fields.

The runtime and hardware see:

- size;
- alignment;
- padding;
- pointers;
- cache-line placement.

For hot data structures, physical layout can become a first-class design concern.

---

## 2. Alignment and Padding

Fields must satisfy alignment requirements.

The compiler may insert padding between fields or at the end of a struct.

Field ordering can therefore change total size.

A smaller struct can increase:

- objects per cache line;
- objects per page;
- cache capacity;
- allocator density.

But layout optimization should be driven by scale.

Saving eight bytes on a struct allocated twice is irrelevant.

Saving eight bytes on tens of millions of objects may matter.

---

## 3. Cache-Line Placement

Fields that are written independently by different cores can still interfere if they share a line.

This produces false sharing.

Example:

```go
type WorkerStats struct {
    Requests atomic.Uint64
    Errors   atomic.Uint64
}
```

If separate workers update these fields heavily, physical adjacency may become a problem.

Cache-line padding can isolate them, but padding increases memory usage.

It is therefore appropriate mainly for:

```text
few
hot
shared
write-heavy
```

objects.

---

## 4. Struct of Values vs Struct of Pointers

A struct containing many pointers:

```go
type Node struct {
    Left  *Node
    Right *Node
    Meta  *Meta
}
```

creates:

- pointer chasing;
- GC scan work;
- independent heap objects.

An index-based representation:

```go
type Node struct {
    Left  uint32
    Right uint32
}
```

may reduce those costs when nodes live in a contiguous array.

The trade-off is more explicit indexing and sentinel semantics.

---

## 5. Array of Structures

AoS:

```go
type Particle struct {
    X, Y   float32
    VX, VY float32
    Life   float32
}

particles []Particle
```

Memory:

```text
X Y VX VY Life | X Y VX VY Life | ...
```

AoS is often good when each operation needs most fields of one object.

---

## 6. Structure of Arrays

SoA:

```go
type Particles struct {
    X, Y   []float32
    VX, VY []float32
    Life   []float32
}
```

Memory is grouped by field.

This can be better when processing one field across many objects.

Benefits may include:

- denser hot data;
- better sequential access;
- reduced bandwidth for unused fields.

Costs include:

- more complex representation;
- multiple slices;
- synchronization between lengths/indices.

---

## 7. Choosing AoS or SoA

The correct question is:

> What is the access pattern?

### AoS fits:

```text
for each object:
    use most fields
```

### SoA fits:

```text
for one field:
    process many objects
```

Neither representation is globally superior.

---

## 8. Hot/Cold Splitting

Suppose:

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

The hot network path may use only:

```text
FD
State
Flags
```

A split:

```go
type Connection struct {
    FD    int
    State uint32
    Flags uint32
    Cold  *connectionCold
}
```

can reduce hot working-set size.

This may be worthwhile even if it adds a pointer or allocation.

The optimization target is cache locality, not allocation count.

---

## 9. Pointer Density and GC

Pointer-bearing fields influence GC scanning.

Pointer-free regions are cheaper for the collector to treat than dense graphs of references.

Thus a layout change can affect both:

```text
CPU cache
+
GC
```

This is why pointer→index transformations can produce multi-layer improvements.

---

## 10. Pointer Field Ordering

For some Go object layouts, placing pointer-bearing fields earlier can reduce the portion of an object that needs to be described/scanned as pointer-containing memory.

The exact collector behavior is version-sensitive.

This technique should therefore be treated as a measured layout optimization, not a universal style rule.

---

## 11. Size Classes

The Go allocator groups small objects into size classes.

A small struct-size change can sometimes move an object into a different allocation class.

This can amplify the memory effect of a field-order change.

Do not design ordinary structs around size-class thresholds.

Consider it only for high-volume types where heap profiles show meaningful impact.

---

## 12. Compact Numeric Types

Reducing a field from 64 bits to 32 bits can shrink a structure, but only when the value range permits it safely.

Benefits may include:

- smaller object;
- better cache density;
- better TLB coverage.

The semantic range requirement comes first.

Never trade correctness for density.

---

## 13. Layout and API Design

Representation choices often leak into APIs.

For example, switching pointers to indexes may change:

- identity semantics;
- lifetime;
- mutation;
- error handling.

A good optimization hides representation details behind stable APIs when possible.

---

## 14. Maintainability

Unusual layouts are easy to "simplify" later.

Important invariants should be documented.

Examples:

```go
// Child links are indices intentionally. Keeping node storage pointer-free
// reduces GC scanning and pointer chasing during bulk traversal.
type node struct {
    left  uint32
    right uint32
}
```

A layout optimization is not complete until the next maintainer can understand why it exists.

---

## 15. Engineering Principle

Data layout is where multiple performance layers meet:

```text
struct size
    ↓
cache density
    ↓
page density
    ↓
TLB
    ↓
memory bandwidth

pointer layout
    ↓
GC scan
    ↓
pointer chasing
```

For high-volume hot data, representation is part of the algorithm.
