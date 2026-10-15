# Mutex

[English](../../02-concurrency/mutex.md) | 简体中文

## 1. Mutex 提供什么

Mutex 为 critical section 提供 exclusive access：

```go
mu.Lock()
update()
mu.Unlock()
```

语义很直接：

> 同一时刻只有一个 goroutine 可以执行受保护的 mutation。

性能几乎完全取决于 contention 与 hold time。

## 2. Uncontended Mutex

没有 contention 时，Go Mutex 的 fast path 很便宜。

它不会在每次 Lock 时都直接进入：

```text
system call
kernel sleep
```

所以把一个 uncontended Mutex 换成复杂 atomic，往往没有实际收益。

## 3. Contended Mutex

当 lock 已被持有，runtime 需要决定如何等待。

可能涉及：

- short spinning；
- queueing；
- goroutine parking；
- wakeup / handoff。

具体 implementation 会随 Go version 演进，但设计目标稳定：

> 让短 contention 尽量便宜，同时避免在长等待中浪费 CPU。

## 4. Spinning

如果 critical section 很快就结束，短暂 spin 可能比 park + reschedule 更便宜。

但 spin 会直接消耗 CPU。

因此这种代码：

```go
for !mu.TryLock() {
}
```

通常不如让 runtime 管理 contention。

## 5. Parking

等待时间较长时，parking 可以避免持续烧 CPU。

Scheduler 可以让当前 P 去执行其他 goroutine。

这也是为什么在高 contention 下，Mutex 有时比 CAS retry loop 更高效：失败者停止继续攻击 shared line。

## 6. Critical-Section Length

例如：

```go
mu.Lock()
value := expensiveTransform(input)
cache[key] = value
mu.Unlock()
```

真正需要 exclusive access 的可能只有 map mutation。

把计算移到 lock 外：

```go
value := expensiveTransform(input)

mu.Lock()
cache[key] = value
mu.Unlock()
```

往往比换 primitive 更重要。

## 7. Lock Convoy

如果很多 goroutine 排在一个持有时间较长、或者被频繁重新获取的 lock 后面，就可能形成 convoy。

症状包括：

- scalability 低；
- mutex profile 热；
- P99 wait 高；
- CPU 没有充分利用。

## 8. Fairness

Perfectly fair 的 Mutex 不一定 throughput 最高。

严格 handoff 可能需要额外 scheduling，而允许正在运行的 goroutine 获取 lock 有时更高效。

Go Mutex 内部会在 throughput 与 starvation control 之间做平衡。

## 9. Starvation Control

如果新 contender 永远抢在旧 waiter 前面，就可能发生 starvation。

现代 Go Mutex 会使用相应策略避免这种 pathological behavior。

这也说明 standard Mutex 已经包含很多 adaptive logic，自制 CAS loop 通常没有。

## 10. Mutex vs Atomic

Atomic 更适合 small independent state。

Mutex 更适合 compound invariant。

例如：

```go
type State struct {
    mu      sync.Mutex
    balance int64
    version uint64
}
```

如果 balance/version 必须一起更新，Mutex 能直接表达 invariant。

两个 atomic 并不会自动变成“一次原子状态更新”。

## 11. Mutex vs RWMutex

read-heavy 不自动意味着 `RWMutex` 更快。

如果 read critical section 很短，reader bookkeeping 可能比普通 Mutex 的 serialization 更贵。

必须用 realistic concurrency 比较。

## 12. Diagnosing Mutex Problems

只有 measurement 证明 contention 后，Mutex 才是性能问题。

有用证据包括：

- mutex profile；
- block profile；
- execution trace；
- throughput scaling；
- critical-section duration。

单纯 code review 不能证明 contention。

## 13. 工程原则

当 Mutex 能清晰表达 invariant 时就使用它。

在替换成复杂 primitive 前，先优化 shared-state design 与 critical-section duration。
