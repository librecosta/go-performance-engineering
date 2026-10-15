# Escape Analysis

## 1. Stack vs Heap Is Usually a Compiler Decision

Go source code often does not explicitly choose whether an object lives on the stack or heap.

Example:

```go
p := &Point{X: 1, Y: 2}
```

Taking an address does not automatically imply heap allocation.

The real question is:

> Can the compiler prove that the pointed-to object does not need to outlive the stack lifetime that owns it?

Escape analysis answers that question.

---

## 2. Why Escape Analysis Exists

Stack allocation is attractive because lifetime is simple.

Conceptually:

```text
function starts
   ↓
stack frame exists
   ↓
local objects live
   ↓
function returns
   ↓
frame no longer needed
```

Heap objects need more general lifetime management.

The garbage collector must track whether they remain reachable.

Therefore keeping suitable values on the stack can reduce:

- heap allocation;
- GC object count;
- scan work;
- allocator metadata work.

---

## 3. Fundamental Safety Constraint

A pointer to stack storage must not remain usable after that stack storage is no longer valid.

Conceptually invalid:

```text
stack object
  ↓ pointer escapes
caller keeps pointer
  ↓
stack lifetime ends
```

The compiler therefore moves storage to the heap when necessary.

---

## 4. Returning a Pointer

Example:

```go
func newPoint() *Point {
    p := Point{X: 1}
    return &p
}
```

At the source level, `p` belongs to `newPoint`.

But the returned pointer remains usable after the function returns.

If `newPoint` remains an independent call boundary, the object cannot simply disappear with its local stack frame.

Heap allocation is the natural safe implementation.

---

## 5. Inlining Can Change the Result

Now:

```go
func newPoint() *Point {
    return &Point{X: 1}
}

func usePoint() int {
    p := newPoint()
    return p.X
}
```

If `newPoint` is inlined, the compiler sees the allocation in the caller:

```go
func usePoint() int {
    p := &Point{X: 1}
    return p.X
}
```

The pointer may no longer need to escape.

This is why escape analysis and inlining are tightly connected.

---

## 6. Address-Taking Is Not the Same as Escaping

Example:

```go
func work() int {
    p := Point{X: 1}
    inspect(&p)
    return p.X
}
```

If `inspect` does not store the pointer beyond the call, `p` may remain on the stack.

Thus:

```text
address taken
```

is not equivalent to:

```text
heap allocated
```

---

## 7. Pointer Receivers Do Not Imply Heap Allocation

Example:

```go
func (p *Point) XValue() int {
    return p.X
}
```

Calling a pointer-receiver method on a local value does not automatically force heap allocation.

The receiver pointer can point to stack storage as long as its lifetime remains valid.

---

## 8. Parameter Leakage

A parameter can "leak" when information derived from it escapes through:

- a return value;
- heap storage;
- a closure;
- global state.

Example:

```go
func identity(p *Point) *Point {
    return p
}
```

The function does not allocate a new object.

But it communicates that the incoming pointer can escape through the return value.

Escape diagnostics often describe this as parameter leakage.

---

## 9. Closures

Closures can extend variable lifetime.

Example:

```go
func counter() func() int {
    n := 0

    return func() int {
        n++
        return n
    }
}
```

The returned closure uses `n` after `counter` returns.

The captured state therefore needs storage with a longer lifetime.

---

## 10. Local Closures Can Be Different

Example:

```go
func work() int {
    n := 0

    inc := func() {
        n++
    }

    inc()
    return n
}
```

If the closure does not escape, the compiler may keep the captured state local.

Again, source syntax alone does not determine placement.

---

## 11. Interfaces and Escape

Interface conversion can affect escape behavior.

Example:

```go
func box(v Value) any {
    return v
}
```

If the interface value escapes, the concrete value representation may also require longer-lived storage.

The exact result depends on:

- value size;
- compiler version;
- inlining;
- use site.

Therefore interface-related allocation should be measured rather than assumed.

---

## 12. Dynamic Calls Reduce Information

If the compiler cannot see or resolve a callee, it may need conservative assumptions.

Inlining and devirtualization can improve the information available to escape analysis.

This is another example of optimization chaining.

---

## 13. Large Objects

Even a value that does not logically escape may be unsuitable for stack placement if it is very large.

Stack growth and frame-size considerations impose practical limits.

Therefore:

```text
does not escape
```

does not mean:

```text
guaranteed stack allocation
```

It means the lifetime analysis does not require heap placement.

---

## 14. Diagnosing Escape

A common compiler diagnostic:

```bash
go build -gcflags='-m=2' ./...
```

can report information such as:

```text
moved to heap
does not escape
leaking param
can inline
cannot inline
```

More detailed compiler output can help explain propagation paths.

The important use is not simply counting "moved to heap".

It is understanding why a hot allocation escapes.

---

## 15. Benchmark Evidence

Compiler diagnostics explain the decision.

Benchmarks answer whether it matters.

Useful signals include:

```text
allocs/op
B/op
ns/op
```

An escape change that removes one tiny allocation from a cold path may have no system value.

---

## 16. API Design Can Influence Escape

APIs that return newly allocated objects often encourage heap lifetime.

Append-style APIs can let callers provide storage.

Example:

```go
func Encode(v Value) []byte
```

versus:

```go
func AppendEncode(dst []byte, v Value) []byte
```

The second form gives callers more control over reuse and allocation.

This is not automatically superior, but it can reduce allocation in hot paths.

---

## 17. Pooling Is Not the First Escape Fix

When a hot object escapes unnecessarily, the first question should be:

> Can the lifetime or API be changed so the object does not need the heap?

Only after that should pooling be considered.

A pool still leaves the object inside heap/GC-managed storage.

Eliminating the heap object entirely can be stronger.

---

## 18. Escape Analysis and Unsafe Tricks

Low-level code can provide explicit no-escape contracts to the compiler for assembly or external implementations.

These contracts are dangerous if false.

They are correctness assertions, not tuning hints.

Application code should not attempt to deceive escape analysis merely to force stack allocation.

---

## 19. Common Misconceptions

### "new always allocates"

False.

### "& always allocates"

False.

### "pointer receiver means heap"

False.

### "does not escape means always stack"

False.

### "one allocation is always a problem"

False.

The only useful question is:

> Does this allocation occur in a measured hot path, and what downstream cost does it create?

---

## 20. Engineering Perspective

Escape analysis connects source-level lifetime with runtime memory management.

The most important performance insight is:

> Removing a heap allocation does more than save allocator CPU. It can remove an object from the entire heap/GC lifecycle.

That is why lifetime-aware API design can be one of the highest-value compiler-level optimizations.
