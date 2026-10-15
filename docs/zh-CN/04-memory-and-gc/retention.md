# Memory Retention

[English](../../04-memory-and-gc/retention.md) | 简体中文

## 1. Allocation 与 Retention 是不同问题

Allocation 问：

> 创建了多少 memory？

Retention 问：

> Memory 保持 reachable 多久？

低 allocation rate 也可以有高 memory，因为保留过多。

高 cumulative allocation 也可以保持很小 live heap，因为 object 很快死亡。

两个问题需要不同证据。

## 2. Backing Array

Slice 是 backing array 上的 descriptor。

概念上包含：

```text
pointer
length
capacity
```

一个 tiny slice 可以保持 huge backing array reachable。

例如：

```go
func first100(buf []byte) []byte {
    return buf[:100]
}
```

如果 `buf` 是 100 MiB，返回的 100-byte view 可能保留整个 allocation。

## 3. Intentional Copy

可以故意 copy：

```go
func first100(buf []byte) []byte {
    return append([]byte(nil), buf[:100]...)
}
```

这样 only small allocation 保留，huge source 可以被 reclaim。

这是“fewer allocations 永远更好”的反例。

## 4. Slice Capacity Retention

```text
len = 0
cap = 32 MiB
```

仍然拥有 32 MiB backing array。

```go
buf = buf[:0]
```

不会释放 capacity。

这对 reuse 很有用，也可能产生 long-lived memory。

## 5. Reusable Buffer

Worker-local buffer 适合正常容量稳定的 workload。

但 outlier request 可能把它扩到非常大。

因此经常需要 policy：

```text
if capacity unusually large
→ drop buffer
```

## 6. `sync.Pool` Retention

Pool 中的 giant buffer 也会提高 footprint。

常见思路：

```go
if cap(buf) <= maxPooled {
    pool.Put(buf[:0])
}
```

Threshold 必须来自 workload。

## 7. String / Substring

String 同样引用 backing data。

Parser 对巨大 source 建立 tiny long-lived token view 时，也要判断是：

- alias source；
- copy into compact storage。

关键是 source/view lifetime relation。

## 8. Cache

Cache 本来就是 intentional retention。

Heap profile 中 cache policy 与 leak 看起来可能很像。

必须问：

- bounded 吗？
- TTL/LRU/size policy？
- value 是否异常大？
- key cardinality 是否无限增长？
- memory pressure 是否反馈到 eviction？

没有明确 ownership policy 的 cache 很容易变成“永久保留”。

## 9. Map

删除 map entry 会移除 logical reference，但 internal capacity 不会必然按 entry count 同比例 shrink。

Long-lived map 如果曾达到巨大 peak，后面很 sparse，仍可能保留 memory。

Phase change 后重建 map 有时合理，但应 measurement-driven。

## 10. Object Graph Retention

一个 root 可以保留整个 graph：

```text
cache entry
   ↓
session
   ↓
history
   ↓
large buffers
```

Heap profile 不应该只责怪最大 leaf allocation。

要理解谁保持它 reachable。

## 11. Closure / Goroutine

Long-lived goroutine 或 closure 可能意外保留 large data。

Goroutine leak 因此经常也是 retention problem。

## 12. Channel

Buffered channel 会保留 queued value。

Large channel capacity 在 consumer stall 时可以成为明显 retention source。

Queue size 本身是 memory-design decision。

## 13. Cleanup / Finalization

等待 cleanup/finalization 的 object 可能拥有比预期更长 effective lifetime。

需要 deterministic release 的资源应该 explicit close/release。

## 14. Retention vs Fragmentation

Reference 已删除，但 RSS 不降，可能是 fragmentation/page release，而不是 reachability retention。

因此 heap inuse 与 process RSS 应分别看。

## 15. Heap Profile vs Allocs Profile

### Heap / inuse

回答：

> 什么还 live？

### Allocs / alloc_space

回答：

> 什么曾经制造大量 churn？

Retention 问题通常从 heap/inuse 开始。

## 16. Copy vs Zero-Copy

Zero-copy 减少：

- allocation；
- memcpy；
- bandwidth。

但可能通过保留 huge source 增加 retained memory。

真正比较的是：

```text
copy cost
vs
retained-memory cost
vs
ownership complexity
```

## 17. Phase-Oriented Program

例如：

```text
load data
↓
build index
↓
discard source
↓
steady state
```

如果 temporary data 已 unreachable，但 RSS 仍高，下一步可能是 scavenging/page release，而不是 leak。

## 18. What to Measure

关注：

```text
heap inuse bytes
heap object count
RSS
large retention paths
slice capacities
cache sizes
goroutine count
mmap/native memory
```

Phase transition 前后对比很有价值。

## 19. 工程视角

Memory retention 本质是 ownership problem。

最有价值的问题通常不是：

> 哪行代码 allocate 了它？

而是：

> 哪个 live reference 仍然负责让这块 storage 保持 reachable？
