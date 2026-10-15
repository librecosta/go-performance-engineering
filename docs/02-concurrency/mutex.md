# Mutex

## 1. What a Mutex Provides

A mutex establishes exclusive access to a critical section.

```go
mu.Lock()
update()
mu.Unlock()
```

Its semantic value is simple:

> Only one goroutine executes the protected mutation at a time.

Performance depends almost entirely on contention and hold time.

---

## 2. Uncontended Mutex

An uncontended Go mutex is designed to be cheap.

The fast path is user-space synchronization.

A mutex does not imply:

```text
system call
kernel sleep
```

on every lock.

This is why replacing an uncontended mutex with complicated atomics often produces little or no real benefit.

---

## 3. Contended Mutex

When the lock is already held, the runtime must decide how to wait.

Possible behavior includes:

- short spinning;
- queueing;
- goroutine parking;
- wakeup/handoff.

The exact implementation evolves with Go versions, but the design goal is stable:

> Make short contention cheap while avoiding wasting CPU on long waits.

---

## 4. Spinning

If a critical section is expected to end very soon, a short spin can be cheaper than parking and rescheduling.

But spinning consumes CPU.

Therefore an unlimited loop such as:

```go
for !mu.TryLock() {
}
```

is usually worse than letting the runtime manage waiting.

---

## 5. Parking

When waiting is not expected to be short, parking the goroutine avoids burning CPU.

The scheduler can run another goroutine on the available execution resource.

This is one reason a mutex can outperform a CAS retry loop under heavy contention.

Losers stop competing.

---

## 6. Critical-Section Length

Consider:

```go
mu.Lock()
value := expensiveTransform(input)
cache[key] = value
mu.Unlock()
```

Only the cache mutation may require exclusivity.

Moving expensive work outside the lock can dramatically reduce contention.

```go
value := expensiveTransform(input)

mu.Lock()
cache[key] = value
mu.Unlock()
```

This is often more important than replacing the primitive.

---

## 7. Lock Convoys

If many goroutines wait behind a lock whose critical section is long or frequently reacquired, the system can form a queue.

Throughput becomes serialized around the lock.

Symptoms may include:

- low scalability;
- high mutex profile;
- long P99 wait;
- CPU not fully utilized.

---

## 8. Fairness

A perfectly fair mutex is not always the fastest.

Handoff and scheduling can cost more than allowing an already-running goroutine to acquire the lock.

Go's mutex implementation balances throughput and starvation behavior.

This is an example of why synchronization primitives contain non-obvious engineering decisions.

---

## 9. Starvation Control

If new contenders repeatedly win, an old waiter could theoretically wait too long.

Modern mutex implementations include strategies to prevent pathological starvation.

The details are runtime implementation, not application API.

The lesson is:

> A standard mutex already contains adaptive contention logic that a hand-written CAS loop usually does not.

---

## 10. Mutex vs Atomic

Atomic operations work well for small independent state.

A mutex works well for compound invariants.

Example:

```go
type State struct {
    mu      sync.Mutex
    balance int64
    version uint64
}
```

If balance and version must change together, a mutex expresses the invariant directly.

Two atomics do not automatically create an atomic pair.

---

## 11. Mutex vs RWMutex

A read-heavy workload does not automatically require `RWMutex`.

If read critical sections are extremely short, reader bookkeeping may cost more than simply using a mutex.

Measure both under realistic concurrency.

---

## 12. Diagnosing Mutex Problems

A mutex is only a performance problem if contention is measured.

Useful observations include:

- mutex profile;
- block profile;
- execution trace;
- throughput scaling;
- critical-section duration.

A code review alone cannot prove contention.

---

## 13. Engineering Principle

Use a mutex when it expresses the invariant clearly.

Optimize the sharing pattern and critical-section duration before replacing it with a more complex synchronization mechanism.
