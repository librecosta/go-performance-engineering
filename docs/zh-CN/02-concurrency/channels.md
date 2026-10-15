# Channel

[English](../../02-concurrency/channels.md) | 简体中文

## 1. Channel 是 Communication Primitive

Channel 不是“更高级的锁”，也不能简单比较成比 Mutex 快或慢。

它同时表达：

- communication；
- synchronization；
- queueing；
- blocking；
- ownership transfer。

因此 Channel 的性能与语义无法分开讨论。

## 2. Shared-State Model

使用 Mutex 时：

```text
G1 ─┐
G2 ─┼→ shared mutable state
G3 ─┘
```

同步负责保护同一对象。

## 3. Ownership-Transfer Model

Channel 可以表达：

```text
producer owns value
        ↓ send
consumer owns value
```

通过移动 responsibility，程序可能显著减少 shared mutation。

这通常比 send/receive 单次 ns/op 更重要。

## 4. Buffered Channel

Buffered channel 是 bounded queue。

它可以暂时解耦 producer/consumer。

收益：

- absorb burst；
- reduce direct handoff；
- provide backpressure。

成本：

- queue memory；
- full 时 producer wait；
- queueing latency。

## 5. Unbuffered Channel

Unbuffered send 需要 matching receiver rendezvous。

这适合 synchronization 本身就是协议的一部分。

它与 lock-protected queue 不是同一种语义。

## 6. Backpressure

Channel 可以故意阻塞 producer。

这经常是 feature，而不是 problem。

没有 backpressure，压力可能转移到：

- unbounded memory；
- goroutine explosion；
- downstream overload。

性能工程必须考虑 system stability，而不只是 throughput。

## 7. Channel vs Mutex

如果问题是：

> 多个 goroutine 需要协调访问 shared state。

Mutex 通常更直接。

如果问题是：

> Work 或 ownership 应该在 participant 之间移动。

Channel 更自然。

不要为了“channel 风格”强行把简单 map 包装成 message passing，也不要用 Mutex 隐藏天然 pipeline。

## 8. Single-Writer Architecture

Channel 经常用于 single-writer：

```text
workers
   ↓ events
aggregator goroutine
   ↓
owns mutable aggregate state
```

Aggregator 内部可以无锁修改 private state。

成本从 shared synchronization 移到 queueing/message passing。

## 9. Batching

Consumer 可以一次 drain 多个 item，然后 batch processing。

可能减少：

- per-item locking；
- syscall frequency；
- aggregation overhead。

代价是 individual latency 可能增加。

## 10. Channel Contention

Channel 自己也可能 contended。

症状：

- many blocked senders；
- many blocked receivers；
- throughput collapse；
- scheduler activity。

它不是天然免疫 shared coordination cost。

## 11. Diagnosing Channel Performance

应该问：

- channel 是否经常 full / empty？
- producer 还是 consumer 更慢？
- buffer 是在吸收 burst，还是只是在隐藏 overload？
- single consumer 是 deliberate serialization 吗？
- queue 能否 sharding？
- batching 是否减少 per-item overhead？

## 12. 工程原则

因为 communication / ownership semantics 选择 Channel。

不要根据单操作 benchmark 在 Channel 与 Mutex 之间机械选边。
