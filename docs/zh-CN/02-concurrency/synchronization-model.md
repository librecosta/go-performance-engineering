# 同步模型

[English](../../02-concurrency/synchronization-model.md) | 简体中文

## 1. 并发性能的核心是 Shared State

并发 primitive 经常被错误地排成：

```text
atomic
mutex
channel
```

然后讨论谁“更快”。

真正需要回答的问题是：

> 哪些 state 被共享？谁可以修改？参与者需要多频繁地协调？

一个更有用的模型是：

```text
shared mutable state
        ↓
coordination
        ↓
cache coherence / waiting / retry
        ↓
throughput and latency
```

## 2. Synchronization 有多种成本

同步可能通过这些路径消耗 CPU：

- atomic instruction；
- cache-line ownership transfer；
- spinning；
- retry；
- goroutine park/wakeup；
- scheduler work。

它也会消耗 latency：

- waiting behind lock；
- single-writer queueing；
- batching delay。

所以 primitive 本身只是成本的一部分。

## 3. Contention

Contention 发生在多个参与者需要同一个资源，并且需求时间发生重叠时。

一个 uncontended Mutex 可以非常便宜。

同一个 Mutex 如果有很多 goroutine 高频进入长 critical section，就可能成为主瓶颈。

一个有用的直觉：

```text
contention pressure
≈ arrival rate × critical-section duration
```

这不是精确公式，但说明为什么缩短 critical section 经常比换 primitive 更有效。

## 4. Blocking、Spinning 与 Retrying

当同步操作不能立即成功时，可以：

### Block / Park

停止占用 CPU，等待条件满足。

### Spin

继续占 CPU，不断检查状态。

### Retry CAS

重新读 state、重新计算、再次提交。

不同策略适合不同等待长度和 contention pattern。

短等待有时值得有限 spin；长等待通常更适合 park；高 retry rate 会让 lock-free 代码浪费 CPU。

## 5. Ownership

Synchronization 存在的根本原因，是多个 actor 共享 mutable state。

如果把 ownership 变成独占，很多同步可以直接消失。

例如：

```text
global mutable map
→ sharded maps

shared counter
→ local counters + merge

mutable config
→ immutable snapshot + atomic pointer

many writers
→ single writer
```

因此 ownership design 往往比 primitive selection 更强。

## 6. Communication vs Shared Memory

Mutex 保护 shared state。

Channel 可以传递 value，也可以传递 ownership。

两者是不同模型。

### Shared State

```text
G1 ─┐
G2 ─┼→ same object
G3 ─┘
```

需要 synchronization。

### Ownership Transfer

```text
G1 owns object
   ↓ send
G2 owns object
```

可以显著减少 shared mutation。

## 7. Fairness vs Throughput

同步算法经常在 fairness 与 throughput 之间权衡。

严格 handoff 能减少 starvation，但增加 scheduler/handoff overhead。

允许已经运行中的 goroutine 抢到资源，可能提高 throughput，却让旧 waiter 更不公平。

Go standard synchronization primitives 内部本身就包含这种工程权衡。

## 8. Tail Latency

Average synchronization cost 很小，并不意味着尾延迟也小。

例如：

```text
most Lock calls: immediate
rare Lock calls: wait milliseconds
```

平均值会掩盖 rare expensive wait。

并发性能应该同时观察：

- throughput；
- CPU；
- P50；
- P99；
- fairness。

## 9. Scheduler Interaction

Goroutine 由 Go runtime scheduler 管理。

一个 goroutine park 后：

```text
G blocked
↓
P may run another G
```

这与“Mutex wait 就立刻让 OS thread sleep”的简单模型不同。

Runtime scheduling 是 synchronization behavior 的组成部分。

## 10. Scaling Curve

同步修改必须跨多个 concurrency level 测量：

```text
1 worker
2 workers
4 workers
8 workers
16 workers
32 workers
```

很多时候，真正有价值的是曲线形状，而不是某一个 ns/op 数字。

## 11. 工程原则

不要问：

> 哪个 synchronization primitive 最快？

应该问：

> 这份 state 到底需要怎样的 coordination？能否在优化 primitive 之前先减少 coordination？
