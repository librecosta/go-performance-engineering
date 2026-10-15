# RWMutex

## 1. Purpose

`RWMutex` allows:

- multiple concurrent readers;
- one exclusive writer.

It is useful when read-side work is substantial enough that concurrent readers provide real benefit.

---

## 2. The Common Misconception

A common rule of thumb says:

```text
read-heavy
→ RWMutex
```

This is incomplete.

Reader operations still require synchronization and bookkeeping.

For very short read sections, that overhead can outweigh the benefit of parallel readers.

---

## 3. Reader Bookkeeping

Readers cannot simply enter for free.

The implementation must track reader/writer coordination.

This means a read lock can involve shared synchronization state.

At high core counts, that shared state itself may become a hotspot.

---

## 4. Writer Arrival

A correct RWMutex must ensure writers eventually acquire exclusive access.

If new readers could enter forever, a writer could starve.

Therefore writer arrival changes how subsequent readers are admitted.

This creates latency behavior that differs from a simple "many readers are free" model.

---

## 5. Read-Critical-Section Duration

Consider:

```go
rw.RLock()
v := cache[key]
rw.RUnlock()
```

If this takes only a tiny amount of work, the reader synchronization overhead may be significant.

But if each reader performs substantial independent computation while holding the read lock, parallelism may be valuable.

---

## 6. Write Frequency

Frequent writers reduce the usefulness of reader concurrency.

Writers require exclusive access.

A workload with:

```text
90% reads
10% writes
```

may behave very differently depending on:

- read duration;
- write duration;
- burstiness;
- number of cores.

Percentages alone are not enough.

---

## 7. Mutex Can Be Faster

A plain mutex may win when:

- critical sections are short;
- contention is moderate;
- reader-side bookkeeping becomes a shared hotspot;
- writes are frequent enough to serialize readers anyway.

Therefore `RWMutex` should be benchmarked against `Mutex` for important hot paths.

---

## 8. Alternatives

For read-mostly configuration, immutable snapshots can remove read locking entirely.

For partitionable state, sharding can reduce contention.

For derived metrics, local accumulation can avoid shared reads/writes.

RWMutex is one tool, not the default end state.

---

## 9. Engineering Principle

Choose `RWMutex` because concurrent read critical sections provide measured value, not because the code contains more reads than writes.
