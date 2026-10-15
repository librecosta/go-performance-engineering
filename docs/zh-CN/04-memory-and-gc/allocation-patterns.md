# Allocation Patterns

[English](../../04-memory-and-gc/allocation-patterns.md) | 简体中文

## 1. Allocation Pattern 比“是否分配”更重要

一次性创建一个 100 MiB long-lived structure，和每秒创建 100 MiB temporary objects，是完全不同的性能模型。

关键维度：

```text
size
frequency
lifetime
pointer content
reuse
```

## 2. Allocation Churn

Churn 表示反复创建短生命周期 heap object。

例如：

```go
for _, item := range items {
    tmp := make([]byte, 4096)
    process(tmp, item)
}
```

这些 buffer 可能很快死亡，live heap 不大，但 bytes/sec 很高。

它会快速消耗 heap headroom，提高 GC frequency。

## 3. Short-Lived 不等于 Free

Temporary object 仍然需要：

```text
allocation
+
heap growth
+
later GC accounting
```

短 lifetime 让它容易回收，但不会抹掉创建成本。

## 4. Allocation Rate

一个有用的近似：

```text
available heap headroom
        ÷
allocation bytes/sec
        ≈
time until next GC pressure
```

这解释了为什么 live heap 几乎不变，GC frequency 却能明显增加。

## 5. Burst Allocation

有些 workload 呈现：

```text
steady state
↓
large batch arrives
↓
temporary allocation spike
↓
back to normal
```

它可能造成：

- transient GC pressure；
- assist；
- RSS spike；
- P99 spike。

Average allocation rate 会掩盖 burst。

## 6. Steady Allocation

稳定小 allocation stream 更容易让 pacer 预测。

优化时要区分 workload 是：

- steady；
- bursty；
- phase-oriented。

## 7. Preallocation

Slice grow 会产生：

```text
new backing allocation
+
copy old contents
```

如果 approximate final size 已知：

```go
out := make([]Item, 0, expected)
```

可以减少 allocation/copy。

## 8. Over-Preallocation

例如通常 100 item，却每次：

```go
make([]Item, 0, 1_000_000)
```

会用 retained memory 换掉一些 allocation。

Capacity 应反映 realistic distribution，而不是 theoretical maximum。

## 9. Map Capacity

Map 也可以在 approximate size 已知时给 hint，以减少 growth/reorganization。

同样存在 oversizing trade-off。

## 10. Append-Style API

```go
func Marshal(v Value) []byte
```

把 output allocation ownership 放在 callee。

```go
func AppendMarshal(dst []byte, v Value) []byte
```

允许 caller reuse。

这在 performance-oriented encoding API 中很常见。

## 11. Caller-Owned Scratch

```go
buf := make([]byte, 0, 4096)

for ... {
    buf = buf[:0]
    buf = encode(buf, value)
}
```

可以避免 repeated allocation。

但 buffer 最大 capacity 会被长期保留，所以需要 size discipline。

## 12. Capacity Poisoning

正常 request 只需要 64 KiB，但一个 outlier 把 reusable buffer 扩到 32 MiB。

如果一直保留：

```text
one outlier
→ permanent larger footprint
```

这是 capacity poisoning。

Pool 与 worker-local scratch 都要考虑。

## 13. Copy vs Retain

例如：

```go
small := append([]byte(nil), huge[:100]...)
```

它新增一个 tiny allocation，却可能让巨大 backing array 释放。

因此：

```text
allocation count ↑
live retained memory ↓↓↓
```

总体更好完全可能。

## 14. `[]T` vs `[]*T`

`[]*T` 可能意味着：

- one slice allocation；
- many object allocations。

`[]T` 可能把所有 object 放进一个 backing array。

这可以减少：

- allocation count；
- pointer scanning；
- allocator traffic。

但 semantics 仍然优先。

## 15. Flat Index Storage

Graph/tree 可以从 many heap nodes 变成 one/few backing arrays。

潜在收益：

- fewer allocations；
- fewer objects；
- less GC metadata；
- better locality。

## 16. Slab-Like Storage

Application 可以自己在大 slice/slab 中管理 logical object：

```go
type Arena struct {
    Nodes []Node
}
```

当大量 object：

```text
born together
used together
die together
```

时特别适合 bulk reset。

## 17. Same-Lifetime Objects

Slab/arena-like 模式最适合生命周期一致的对象：

- parser AST；
- query plan；
- request graph；
- batch state。

Mixed lifetime 会降低价值。

## 18. `sync.Pool`

`sync.Pool` 允许 temporary heap object opportunistic reuse。

因为 semantics 特殊，会单独讨论：

- [sync.Pool](./sync-pool.md)

## 19. Large Temporary Buffer

Large buffer 值得特殊处理，因为它可能：

- page-granular allocation；
- RSS spike；
- poison pool；
- push memory-limit pressure。

常见策略是：

```text
reuse common sizes
drop unusual outliers
```

而不是 pool every capacity。

## 20. Batch Allocation

例如：

```go
items := make([]Item, n)
```

相比：

```go
items := make([]*Item, n)
for i := range items {
    items[i] = new(Item)
}
```

可能大幅减少 object count。

## 21. Escape-Driven Allocation

一些 allocation 其实是 compiler/lifetime artifact：

- interface escape；
- closure capture；
- opaque callee；
- failed inlining。

加 pool 前先检查能否通过 lifetime/API design 直接删除 allocation。

## 22. Allocation 与 Tail Latency

Burst allocation 可能让 goroutine 承担 GC assist。

即使 STW 很小，request latency 仍可能出现 spike。

所以 allocation optimization 也可能是 latency optimization。

## 23. What to Compare

评估 allocation change 时同时看：

```text
ns/op
B/op
allocs/op
GC CPU
live heap
RSS
P99 latency
```

没有一个单指标足够。

## 24. 工程视角

好的 allocation strategy 应与 object lifetime 对齐：

```text
stack-local
caller-owned
batch-owned
pool-reusable
long-lived heap
```

Ownership/lifetime 越清晰，越容易得到可控性能。
