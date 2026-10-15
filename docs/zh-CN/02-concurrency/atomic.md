# Atomic 操作

[English](../../02-concurrency/atomic.md) | 简体中文

## 1. Atomic 的含义

Atomic operation 提供不可分割的 memory operation 和同步语义。

常见类型：

- load；
- store；
- swap；
- add；
- compare-and-swap。

Atomic 不等于 free。

它意味着 concurrent access 在 memory model 下具有明确语义。

## 2. Read vs Write

Atomic load 在多个 core 只读 shared state 时通常比较便宜。

write 或 read-modify-write 更贵，因为需要协调 cache-line ownership。

所以：

```text
atomic.Load
```

和：

```text
atomic.Add
```

属于完全不同的性能类别。

## 3. Read-Modify-Write

例如：

```go
counter.Add(1)
```

需要原子地读取旧值并发布新值。

如果所有 core 都更新一个 shared counter，它们会争夺同一 cache line。

虽然没有 software lock，硬件仍然需要序列化 ownership。

## 4. True Sharing

```go
var requests atomic.Uint64
```

如果所有 worker 高频更新：

```text
Core 0 writes
Core 1 writes
Core 2 writes
...
```

line 不断在 core 之间移动，scalability 会快速恶化。

## 5. Sharded Counter

常见 transformation：

```text
one global counter
→ many local/sharded counters
```

每个 worker 写不同 memory location。

只在读取/汇总时聚合。

它用：

```text
more expensive reads/aggregation
```

换取：

```text
much cheaper writes
```

对 write-heavy metrics 经常非常合适。

## 6. Immutable Snapshot

Atomic 特别适合 read-mostly state。

例如：

```go
type Store struct {
    config atomic.Pointer[Config]
}
```

Writer 构建完整 immutable config，然后一次 publish。

Reader：

```go
cfg := store.config.Load()
```

可以避免 read lock 和 reader-side RMW。

## 7. Compare-and-Swap

CAS 只有在 current value 等于 expected value 时才提交。

概念上：

```text
load old
compute new
CAS(old, new)
```

低 contention 时可以很高效。

高 contention 时，大量 goroutine 会计算出立即被丢弃的 candidate。

## 8. Retry Amplification

假设 32 个 goroutine 同时竞争一个 CAS。

一个成功，31 个失败。

如果立刻 retry：

```text
1 useful update
31 failed attempts
```

失败工作可能比成功工作更多，还会继续制造 cache traffic。

这就是 retry amplification。

## 9. Atomic 不会自动 Compose

两个 atomic：

```go
balance atomic.Int64
version atomic.Uint64
```

不是一个 compound atomic state。

Reader 可能看到：

```text
new balance
old version
```

如果多个 field 属于一个 invariant，应考虑：

- Mutex；
- immutable snapshot；
- carefully packed state。

## 10. Memory Ordering

Atomic 同时包含 memory-ordering semantics。

不同 architecture 上 generated instruction 可能不同。

Application code 应根据 Go memory model 推理，而不是根据某一款 CPU 的 instruction sequence。

## 11. Atomic vs Mutex

Atomic 更适合：

- state 可以由一个简单 atomic operation 表示；
- invariant 很小；
- contention 低或 read-mostly。

Mutex 更适合：

- multiple fields 一起变化；
- critical section 复杂；
- retry 会重复昂贵 work。

选择首先是 semantic decision，其次才是 performance decision。

## 12. Diagnosing Atomic Hotspot

CPU profile 中 hot atomic 可能意味着：

- true sharing；
- false sharing；
- retry loop；
- synchronization frequency 太高。

真正解决方案可能是：

- sharding；
- batching；
- local accumulation；
- ownership redesign。

单纯替换 atomic primitive 通常解决不了 topology。

## 13. 工程原则

Atomic 去掉的是 software locking，不是 hardware coordination。

当很多 core 写同一个位置时：

> cache coherence 就成为那把“锁”。
