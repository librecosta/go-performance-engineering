# The ABA Problem

## 1. What ABA Means

ABA occurs when a value changes:

```text
A
↓
B
↓
A
```

A compare-and-swap operation only sees that the current value is `A`.

It does not know that the state changed in between.

---

## 2. Stack Example

Suppose:

```text
head
 ↓
A → B → C
```

Goroutine G1 reads:

```text
old = A
next = B
```

Before G1 performs CAS, G2:

```text
pop A
pop B
push A
```

Now:

```text
head
 ↓
A → C
```

G1 resumes and checks:

```text
head == A
```

The comparison succeeds, even though the structure is no longer the same state G1 originally observed.

---

## 3. ABA Is About Identity and History

ABA is often described as an allocator problem:

```text
object freed
same address reused
```

That can happen, but ABA does not require allocation reuse.

The exact same object can be:

```text
removed
reinserted
```

and create the same logical problem.

Therefore:

> GC does not automatically eliminate ABA.

---

## 4. Why CAS Cannot Detect History

CAS compares the current machine value with an expected value.

It does not encode:

- version;
- mutation history;
- generation.

If the machine representation is identical, CAS considers it unchanged.

---

## 5. Tagged or Versioned State

A common defense stores:

```text
(value, version)
```

instead of only:

```text
value
```

Example:

```text
(A, 17)
```

After removal and reinsertion:

```text
(A, 18)
```

An old CAS expecting `(A,17)` now fails.

---

## 6. Tagged Pointer Idea

Low-level algorithms may pack:

- pointer/address information;
- version/tag information;

into a single atomic word.

This can make CAS sensitive to reuse.

The exact technique depends on:

- address width;
- alignment;
- architecture;
- runtime assumptions.

Such representations are highly implementation-sensitive.

---

## 7. Go Runtime as a Study Case

The Go runtime contains internal lock-free structures that use tagged/versioned state.

These are valuable for understanding algorithm design.

They should not automatically be copied into application code.

Runtime internals may rely on:

- non-public invariants;
- non-GC-visible representations;
- architecture-specific assumptions.

---

## 8. GC and ABA

Go's GC helps with memory lifetime.

A normal Go pointer held by a goroutine keeps an object reachable.

This reduces use-after-free problems.

But logical ABA remains possible if the same object identity returns to the structure.

---

## 9. Pooling

Object pools and freelists increase reuse.

This can increase the probability that old identities appear again.

A lock-free data structure that introduces `sync.Pool` must re-evaluate ABA assumptions.

---

## 10. Testing ABA

ABA bugs are difficult to catch with ordinary unit tests.

Useful strategies include:

- targeted orchestrated interleavings;
- stress tests;
- model-based testing;
- invariant checks.

Race detection alone cannot prove ABA safety.

---

## 11. Engineering Principle

Whenever CAS correctness depends on:

> "If the value is unchanged, the state is unchanged."

ask whether the value can transition away and later return.

If yes, ABA must be considered explicitly.
