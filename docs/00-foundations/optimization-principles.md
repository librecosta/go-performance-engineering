# Optimization Principles

## 1. Measure Before Optimizing

A code pattern is not automatically a bottleneck.

The following are only candidates:

- allocation;
- mutex;
- interface;
- copy;
- branch;
- bounds check;
- atomic operation;
- pointer traversal.

Optimization begins when measurement shows that one of these contributes meaningfully to the system objective.

---

## 2. Remove Work Before Making Work Faster

The cheapest work is work that does not happen.

Compiler optimizations embody this principle:

```text
heap allocation
→ stack allocation

bounds check
→ eliminated

function call
→ inline

constant branch
→ removed
```

Application design can do the same:

```text
shared write
→ local accumulation

many lock operations
→ one batch

many heap nodes
→ one flat backing array
```

Before optimizing an operation, ask whether the operation can disappear.

---

## 3. Optimize the Bottleneck, Not the Smell

A mutex in code is not evidence of lock contention.

An interface is not evidence of dispatch overhead.

An allocation is not evidence of GC trouble.

Treat "performance smells" as investigation prompts, not conclusions.

---

## 4. Prefer Representation Changes Over Micro-Tricks

Many major improvements come from changing the representation:

```text
pointer graph
→ index-based storage

large mixed struct
→ hot/cold split

single shared counter
→ sharded counters

temporary object tree
→ slab/arena-like storage
```

These changes reduce fundamental costs rather than shaving instructions around them.

---

## 5. Make the Compiler's Job Easy

The compiler can remove work only when it can prove safety.

Source structure can communicate useful invariants:

- length guards;
- clear control flow;
- concrete types;
- simple hot helpers;
- predictable ownership.

Do not use unsafe tricks merely because the compiler initially failed to optimize something.

First ask whether the same information can be expressed safely.

---

## 6. Treat Memory Layout as an Algorithmic Decision

Data layout determines:

- cache behavior;
- TLB behavior;
- bandwidth;
- pointer chasing;
- GC scan work.

For large hot data structures, layout can matter as much as algorithm complexity.

---

## 7. Reduce Sharing Before Optimizing Synchronization

High contention often indicates a shared-state design problem.

Possible transformations:

```text
global counter
→ local counters + aggregation

global map
→ shards

shared mutable state
→ immutable snapshot

multi-writer state
→ single writer
```

Changing a mutex to an atomic does not remove true sharing.

---

## 8. Use Memory to Save CPU Deliberately

Many performance strategies spend memory:

- caching;
- pooling;
- larger GC targets;
- preallocation.

This can be correct.

But memory must be treated as a budget, not as free capacity.

Measure both:

```text
CPU saved
memory retained
```

---

## 9. Do Not Worship Zero Allocation

Zero allocation is useful in some hot paths, but it is not the ultimate metric.

A one-allocation implementation may be faster, clearer, and safer.

Likewise, an intentional copy may allow a large backing buffer to be released.

The objective is total system cost.

---

## 10. Do Not Worship Lock-Free

Lock-free algorithms provide a progress guarantee, not a speed guarantee.

Under contention they may create:

- retry storms;
- cache-line traffic;
- starvation;
- higher CPU consumption.

A well-designed mutex can outperform a naive CAS loop.

---

## 11. Do Not Worship Zero-Copy

Zero-copy removes data movement but introduces:

- shared ownership;
- aliasing;
- lifetime coupling;
- retention risks.

If the copied data is small or the copy is not hot, the complexity may not be justified.

---

## 12. Preserve Optimization Intent

Some optimized code looks worse than the naive version.

Examples:

- unused bounds proof;
- intentional copy;
- padding;
- unusual field order;
- index instead of pointer;
- pool capacity limit.

When the reason is not obvious from local syntax, preserve it through comments, tests, benchmarks, or design notes.

This is not optional documentation overhead.

It protects the optimization from accidental regression.

---

## 13. Optimize for the Target Environment

Performance depends on:

- Go version;
- GOARCH;
- CPU;
- operating system;
- workload;
- concurrency level.

Do not treat benchmark results from another environment as universal truth.

Implementation-sensitive claims must be revalidated after major Go toolchain upgrades.

---

## 14. Prefer Stable Contracts Over Runtime Internals

Go runtime source is valuable for understanding mechanisms.

It is not automatically a public API.

Prefer:

```text
public language/runtime contract
```

over:

```text
linkname/private runtime behavior
```

unless maintaining a low-level specialized library where the compatibility burden is explicit.

---

## 15. Use the Lowest-Level Tool Necessary

Start with high-level evidence.

Escalate only when required:

```text
metrics / profile
→ benchmark
→ compiler diagnostics
→ SSA / assembly
→ perf / PMU
```

A cache-coherence investigation should not begin with assembly if a scaling benchmark already shows the issue clearly.

---

## 16. Validate at Multiple Levels

A local benchmark proves a local effect.

A system benchmark proves whether that effect matters.

A mature validation path is:

```text
microbenchmark
→ component
→ service
→ production/canary
```

The broader the change, the broader the validation required.

---

## 17. Keep Guardrails

An optimization can improve one metric while degrading another.

Examples:

```text
throughput +10%
P99 +50%
```

or:

```text
CPU -15%
RSS +2×
```

Whether this is acceptable depends on the system objective.

Always define guardrails.

---

## 18. Stop When the Value Is Too Small

Performance engineering includes deciding not to optimize.

Reject a change when:

- the hotspot is negligible;
- the theoretical gain is tiny;
- complexity is disproportionate;
- system-level benefit does not appear;
- correctness or maintainability risk is too high.

A technically faster implementation is not automatically a better implementation.
