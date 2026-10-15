# RWMutex

[English](../../02-concurrency/rwmutex.md) | 简体中文

## 1. 目的

`RWMutex` 允许：

- 多个 concurrent reader；
- 一个 exclusive writer。

它适合 read-side work 足够大，以至于 reader parallelism 能产生真实收益的 workload。

## 2. 常见误解

经常看到：

```text
read-heavy
→ RWMutex
```

这不完整。

Reader 也需要 synchronization 与 bookkeeping。

如果 read section 极短，这些 overhead 可能比并行收益更大。

## 3. Reader Bookkeeping

Reader 不是免费进入。

Implementation 需要维护 reader/writer coordination。

因此 read lock 本身也可能操作 shared synchronization state。

高 core count 下，这份 state 也可能成为 hotspot。

## 4. Writer Arrival

正确的 RWMutex 必须保证 writer 最终获得 exclusive access。

如果新 reader 可以永远进入，writer 就可能 starvation。

因此 writer arrival 会改变后续 reader admission behavior。

## 5. Read-Critical-Section Duration

例如：

```go
rw.RLock()
v := cache[key]
rw.RUnlock()
```

如果这段非常短，reader synchronization overhead 可能占比很高。

如果 reader 在 read lock 内进行相对重的 independent work，并行价值就更高。

## 6. Write Frequency

频繁 writer 会降低 reader concurrency 的价值。

一个：

```text
90% reads
10% writes
```

的 workload，在不同：

- read duration；
- write duration；
- burstiness；
- core count；

下可能完全不同。

比例本身不够。

## 7. Mutex 可能更快

普通 Mutex 可能在这些情况下胜出：

- critical section 很短；
- contention 中等；
- reader bookkeeping 自己变成 shared hotspot；
- write 足够频繁，本来也会打断 reader parallelism。

重要 hot path 应直接 benchmark Mutex vs RWMutex。

## 8. Alternatives

Read-mostly configuration 可以用 immutable snapshot 消除 read lock。

可 partition state 可以 sharding。

derived metrics 可以 local accumulation。

RWMutex 是工具之一，不是“读多写少”的最终答案。

## 9. 工程原则

选择 `RWMutex` 的理由应该是：

> concurrent read critical section 已被证明具有价值。

而不是：

> 代码里 read 数量比 write 多。
