# Branch Prediction

## 1. Why Branch Prediction Exists

Modern processors use deep pipelines and speculative execution.

When the CPU encounters:

```go
if cond {
    pathA()
} else {
    pathB()
}
```

it would be expensive to wait until `cond` is fully resolved before fetching future instructions.

Instead, the processor predicts which path will execute.

---

## 2. Correct Prediction

When the prediction is correct, execution continues with little disruption.

Predictable branches can therefore be very cheap.

Example pattern:

```text
false false false false false false
```

or:

```text
true true true true true true
```

The predictor can learn stable behavior.

---

## 3. Misprediction

When the prediction is wrong, speculative work from the wrong path must be discarded.

The processor then resumes from the correct path.

This creates pipeline disruption.

The exact penalty depends on architecture, but the engineering consequence is clear:

> Unpredictable branch outcomes can be much more expensive than predictable ones.

---

## 4. Data Distribution Is Part of Performance

Consider:

```go
for _, x := range values {
    if x >= threshold {
        sum += int(x)
    }
}
```

If data is sorted:

```text
false false false ... true true true
```

the branch may be predictable.

If data is random near 50/50:

```text
T F T F F T T F ...
```

prediction becomes harder.

The same code can therefore have different performance on different input distributions.

---

## 5. Benchmark Implications

Synthetic benchmarks often accidentally create unrealistically predictable branches.

Examples:

- always the same HTTP method;
- always a cache hit;
- always the same parser shape;
- always the first switch case.

Production traffic may have a very different branch distribution.

A benchmark that ignores this can produce misleading conclusions.

---

## 6. Branchless Programming

A branch can sometimes be replaced by arithmetic, masks, or table lookup.

This is often described as "branchless programming".

But:

```text
branchless
≠
automatically faster
```

If the original branch is highly predictable, replacing it may:

- add instructions;
- create new dependencies;
- increase memory access;
- hurt readability.

Branchless transformations must be measured.

---

## 7. Conditional Moves and Compiler Decisions

Compilers may transform simple conditionals into branchless machine instructions when profitable.

Therefore source code should not be rewritten solely to force branchless behavior without checking generated code.

The Go compiler may choose different lowering based on:

- architecture;
- surrounding control flow;
- optimization passes.

---

## 8. Hot and Cold Paths

Programs often have:

```text
common fast path
rare slow path
```

Examples:

- valid input vs error formatting;
- cache hit vs fallback;
- normal packet vs diagnostic logging.

Separating rare cold logic can sometimes reduce hot instruction footprint.

However, this is not a universal rule.

Inlining and code layout decisions are compiler-sensitive and should be verified.

---

## 9. Branch Prediction and State Machines

Protocol parsers and state machines may contain many branches.

Performance depends on:

- distribution of states;
- predictability of transitions;
- data representation;
- code layout.

Flattening a state machine or using a table may help or hurt depending on access behavior.

---

## 10. PMU Evidence

When branch behavior is suspected but source-level profiling is insufficient, hardware counters can provide:

```text
branches
branch-misses
```

These counters should support a hypothesis, not replace it.

A high branch-miss rate is meaningful only in context:

- how hot is the code;
- what input distribution produced it;
- what change reduces it;
- what extra work the change introduces.

---

## 11. Engineering Principle

Do not ask:

> How do I remove branches?

Ask:

> Is branch misprediction a significant cost in this workload, and can the control/data representation make the hot path more predictable without unacceptable complexity?
