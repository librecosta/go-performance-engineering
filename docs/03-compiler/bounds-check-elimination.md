# Bounds Check Elimination

## 1. Why Bounds Checks Exist

Go requires safe indexed access.

For:

```go
v := s[i]
```

the program must panic if:

```text
i < 0
or
i >= len(s)
```

This safety guarantee normally requires a bounds check.

But if the compiler can prove the index is valid, emitting another check is unnecessary.

Removing such redundant checks is bounds-check elimination (BCE).

---

## 2. BCE Is a Proof Problem

The compiler must establish:

```text
0 <= i < len(s)
```

on every path reaching the access.

If the proof is incomplete, the check must remain.

Therefore BCE depends heavily on control flow and dominance.

---

## 3. Range Loops

Example:

```go
for i := range s {
    sum += s[i]
}
```

The semantics of `range` already establish that `i` is a valid index.

This gives the compiler strong proof information.

---

## 4. Explicit Length Guard

Example:

```go
func decode(b []byte) uint32 {
    if len(b) < 4 {
        return 0
    }

    return uint32(b[0]) |
        uint32(b[1])<<8 |
        uint32(b[2])<<16 |
        uint32(b[3])<<24
}
```

On the successful path:

```text
len(b) >= 4
```

Therefore the indexed reads are provably safe.

This pattern is both readable and optimization-friendly.

---

## 5. Dominating Maximum-Index Check

Sometimes a parser performs several fixed indexed reads.

A common proof pattern is:

```go
_ = b[7]

x0 := b[0]
x1 := b[2]
x2 := b[7]
```

The first access proves:

```text
len(b) >= 8
```

If it dominates later accesses, redundant checks may disappear.

The value is intentionally unused.

This is a non-obvious optimization and should be commented if retained for performance.

---

## 6. Why `_ = b[N]` Is Safe

This pattern does not bypass bounds safety.

If `b[N]` is invalid, it still panics according to normal Go semantics.

The optimization only consolidates proof.

It is fundamentally different from unsafe pointer arithmetic that removes safety checks entirely.

---

## 7. Loop Shapes

BCE can depend on loop structure.

Simple loops with clear index relationships are easier to prove than complicated index mutation.

For example:

```go
for i := 0; i < len(s); i++ {
    use(s[i])
}
```

has an obvious relation.

More complex arithmetic may make the proof harder.

---

## 8. Slice Operations

Slicing also requires bounds validation.

Expressions such as:

```go
s[a:b]
```

must satisfy ordering and length constraints.

Compiler proof can eliminate redundant slice-bound checks just as it can eliminate index checks.

---

## 9. Inlining Can Enable BCE

Caller:

```go
if len(b) < 8 {
    return
}
```

Helper:

```go
func read7(b []byte) byte {
    return b[7]
}
```

After inlining, the caller's length proof becomes available around `b[7]`.

Thus:

```text
inlining
→ BCE
```

---

## 10. Diagnosing BCE

Go provides compiler diagnostics for residual bounds checks.

A common command is:

```bash
go build -gcflags='-d=ssa/check_bce/debug=1' ./...
```

Important interpretation:

```text
Found IsInBounds
```

indicates that a bounds-check operation remains.

It does not mean:

> The check was successfully eliminated.

This detail is easy to misread.

---

## 11. Assembly Evidence

If needed, assembly can confirm whether index-check and panic paths survive.

Possible tools:

```bash
go build -gcflags='-S' ./...
```

or:

```bash
go tool objdump ...
```

Assembly is useful when diagnostics alone do not answer the question.

---

## 12. BCE and Parsers

BCE is particularly relevant to:

- binary protocol decoders;
- image codecs;
- serialization libraries;
- packet parsers;
- compression code.

These workloads often perform many fixed-offset reads.

A small number of redundant checks repeated at very high frequency can become measurable.

---

## 13. BCE and Application Code

Ordinary business logic rarely needs special BCE shaping.

Readable length guards are usually enough.

Do not add obscure proof tricks to cold code.

The maintenance cost can exceed the runtime benefit.

---

## 14. Unsafe Is Not the First Solution

If bounds checks remain in a hot path, the escalation should be:

```text
clear safe control flow
   ↓
compiler diagnostics
   ↓
BCE-friendly source structure
   ↓
verify again
```

Only specialized low-level code should consider unsafe indexing after proving that safe forms cannot achieve the required performance.

---

## 15. Version Sensitivity

BCE behavior depends on compiler improvements.

A workaround needed in one Go version may become unnecessary later.

Therefore a non-obvious BCE trick should be revalidated after toolchain upgrades.

If the compiler can now remove the checks without the trick, simplify the source.

---

## 16. Proof Repository Relationship

A minimal BCE proof can demonstrate:

- baseline source;
- proof-shaped source;
- compiler diagnostics;
- assembly;
- benchmark.

Its purpose should be:

> Demonstrate that the compiler can eliminate the redundant checks.

Not:

> Claim a universal percentage speedup.

---

## 17. Common Misconceptions

### "Every index access has a runtime check"

False.

### "The compiler always removes loop bounds checks"

Not always.

### "`Found IsInBounds` means BCE succeeded"

False.

### "Unsafe is required for fast parsers"

False.

---

## 18. Engineering Perspective

Bounds-check elimination is a good example of a broader compiler principle:

> Safety checks disappear when the compiler can prove the safety condition from program structure.

The optimization target is not "fewer safety guarantees".

It is "fewer redundant runtime checks".
