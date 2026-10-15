# 面向所有权的设计

[English](../../02-concurrency/ownership-design.md) | 简体中文

## 1. 最好的 Synchronization 往往是没有 Synchronization

同步存在，是因为 mutable state 被共享。

如果 ownership 变成 exclusive，很多同步可以直接删除。

这是并发优化中最强的一类 transformation。

## 2. Shared Mutable State

典型设计：

```text
G1 ─┐
G2 ─┼→ mutable structure
G3 ─┘
```

需要：

- Mutex；
- atomic；
- channel serialization；
- 其他 coordination。

Concurrency 增长后，shared state 很容易成为 bottleneck。

## 3. Partition Ownership

把：

```text
all keys
  ↓
one map
```

改成：

```text
hash(key)
  ↓
shard 0
shard 1
shard 2
...
```

每个 shard contender 更少。

这是 lock striping/sharding。

代价：

- more structures；
- iteration 更复杂；
- rebalance；
- shard-count decision。

## 4. Local Accumulation

Global counter 是典型 hotspot。

可以让 worker：

```text
worker
  ↓
local count
```

然后周期性：

```text
merge
```

显著减少 synchronization frequency。

代价是 aggregation 延迟。

对 metrics/statistics 通常可以接受。

## 5. Single Writer

Single-writer architecture 让一个 goroutine 独占 mutable state：

```text
workers
   ↓ messages
aggregator
   ↓
owns state
```

Aggregator 内部：

```go
stats[key]++
```

可以不需要 Mutex。

成本从 shared synchronization 变成：

- queueing；
- channel communication；
- possible serialization。

当 mutation 很便宜、coordination 很贵时尤其适合。

## 6. Immutable Snapshot

Read-mostly state 可以构建 immutable version。

Writer：

```text
build new snapshot
↓
publish pointer
```

Reader：

```text
load snapshot
↓
read without mutation
```

这样可以消除 read locking。

代价是：

- rebuild/copy new snapshot；
- transition 期间 old/new version 同时占 memory。

## 7. Batching

不要每个 operation 都：

```text
lock
update one
unlock
```

而是：

```text
prepare many updates
↓
lock
apply batch
unlock
```

收益：

- fewer lock operations；
- less handoff；
- better amortization。

代价：

- temporary buffer；
- delayed visibility；
- higher individual latency。

## 8. Message Passing

Message passing 可以显式表达 ownership transfer，减少模糊 shared mutation。

但 message passing 也有成本：

- allocation；
- channel contention；
- copy；
- queueing。

应该先因为 ownership clarity 选择，然后 benchmark。

## 9. Read Replication

一些 workload 可以复制 read-only state 给各 worker。

这样减少 shared read/synchronization。

代价：

- replication memory；
- update propagation；
- consistency delay。

这是典型：

```text
memory
→ lower coordination
```

## 10. Ownership 与 Cache Locality

Ownership 也影响 hardware。

如果某个 worker 长时间独占一组 state，它们的 cache line 更可能保持 local。

如果很多 worker 不断写同一 state，line 会不断在 core 之间 bounce。

因此 ownership 同时减少：

- logical synchronization；
- physical coherence traffic。

## 11. Ownership Boundary

一个很有价值的设计问题：

> 哪个 goroutine 负责修改这份 state？

如果回答是：

> everyone

就应该重新审视结构。

更好的答案经常是：

- one worker；
- one shard owner；
- one pipeline stage；
- one immutable version。

## 12. 什么时候 Sharing 不可避免

有些 state 本来就是 shared：

- coordination flags；
- global limits；
- connection registry。

目标不是消灭所有 sharing。

目标是最小化 coordination frequency 与 scope。

## 13. Ownership 作为 Optimization Strategy

Primitive-level thinking：

```text
Mutex vs Atomic vs Channel?
```

Ownership-level thinking：

```text
为什么这些 goroutine 必须一起修改同一份 state？
```

第二个问题经常带来更大、更稳定的性能收益。

## 14. 工程原则

当 contention 已经很严重时，不要继续只优化 synchronization primitive。

应该重新设计 ownership topology。
