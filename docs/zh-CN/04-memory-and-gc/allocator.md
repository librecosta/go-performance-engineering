# Go Memory Allocator

[English](../../04-memory-and-gc/allocator.md) | 简体中文

## 1. 为什么 Allocation 值得研究

Heap allocation 经常被简单理解成“慢”。

这个模型太粗糙。

现代 Go 为常见 small allocation 做了大量优化：per-P allocation state、size class、span、cache 等机制，使大多数 small allocation 不需要碰 global lock，也不需要每次进入 OS。

真正应该问的是：

> 分配发生多频繁？每次多少 bytes？对象活多久？这些 allocation 后续制造了多少 GC work？

Allocation 成本可以拆成：

```text
allocation CPU
+
zeroing / initialization
+
object metadata
+
heap growth
+
GC frequency
+
GC scan work
```

一个便宜的 allocation 可能完全无关紧要。

每秒几百万次 allocation 即使单次很便宜，也可能变成系统级成本。

## 2. Runtime Allocation Hierarchy

简化模型：

```text
goroutine
   ↓
current P
   ↓
mcache
   ↓
mcentral
   ↓
mheap
   ↓
operating system
```

每层的目标都是让 common path 尽可能 local，把昂贵 global work 摊薄。

### mcache

`mcache` 与 P 关联，保存不同 size class 的可分配 span。

常见 small allocation 可以直接从 current P 的 cached span 取 slot，避免 global lock。

### mcentral

当当前 span 没有 free slot 时，可以从对应 size class 的 central structure 获取新 span。

### mheap

`mheap` 以 page 粒度管理整个 Go heap。

### Operating System

Runtime 只有在现有 heap capacity 不够时才需要向 OS 获取更多 memory。

这远比 application allocation 本身低频。

## 3. Small Allocation 与 Size Class

Go 会把 small object round 到一组 size class。

概念上：

```text
request N bytes
    ↓
choose size class
    ↓
allocate one slot from span
```

它用少量 internal fragmentation 换取：

- fast allocation；
- fixed-size slot management；
- efficient reuse；
- simpler metadata。

具体 threshold 与 size-class table 属于 runtime implementation detail，不应该成为应用层常量。

## 4. Internal Fragmentation

如果请求 65 bytes，但实际使用更大的 size-class slot，slot 中剩余部分就是 internal fragmentation。

单个 object 没关系。

数百万 long-lived object 时，小差异可能积累成明显 footprint。

但不要为了理论上的 size-class boundary 重排普通 struct，除非 heap profile 证明它真的重要。

## 5. mspan

`mspan` 表示一组 page，可用于某个 small-object size class，或者 large allocation。

Small object span 可以概念化为：

```text
span
┌────┬────┬────┬────┬────┐
│obj │obj │free│obj │free│
└────┴────┴────┴────┴────┘
```

Span occupancy 与后续 fragmentation/page reclamation 密切相关。

## 6. Large Allocation

超过 small-allocation threshold 的 object 会走不同路径，通常以 page 粒度管理。

例如：

```go
buf := make([]byte, 4<<20)
```

如果在 hot request path 高频发生，比一个孤立 tiny object 更值得调查。

Large allocation 会更快消耗 heap headroom，也更容易影响 RSS 与 pooling policy。

## 7. Tiny Allocator

Go 对非常小、pointer-free 的 object 还有 specialized tiny allocation path，可以把多个 tiny noscan object 放入一个小 block，减少 bookkeeping。

关键条件是：

> object 不包含 Go pointer。

这再次说明 pointer-free representation 同时影响 allocator 与 GC。

具体 tiny threshold 同样是 implementation detail。

## 8. Pointer-Free vs Pointer-Containing

例如：

```go
make([]byte, n)
```

backing array pointer-free。

而：

```go
make([]*Node, n)
```

包含 Go pointer。

两者都分配 bytes，但后者会产生 GC-visible reference，需要参与 tracing。

所以 allocation bytes 不能单独描述 GC cost。

## 9. Zeroing

Go 保证 newly allocated memory 具有正确 zero value。

Runtime/compiler 因此必须保证 memory appropriately cleared。

Large object 的 zeroing 本身可能成为 measurable cost。

Reuse 大 scratch buffer 有时因此有收益：

```text
avoid allocation
+
avoid repeated zeroing/growth
```

但 reuse 也可能 retention 过大，所以仍是 trade-off。

## 10. Modern Small-Allocation Optimization

现代 Go 持续改进 very small allocation fast path。

这意味着旧 microbenchmark 不应当作为永久常数。

稳定结论应该是：

> 不要把“一个 small allocation 的成本”写成跨版本固定事实。

真正值得长期关注的是 allocation rate、object count、lifetime 与 downstream GC cost。

## 11. Allocation Count vs Allocation Bytes

例如：

```text
10,000 allocations/s × 1 KiB
≈ 10 MiB/s
```

与：

```text
1,000,000 allocations/s × 8 B
≈ 8 MiB/s
```

bytes/sec 接近，但 object count 完全不同。

前者偏向 large-object byte traffic，后者偏向 allocator/object bookkeeping。

两个指标都要看。

## 12. Fast Path 不等于 Free

Small allocation 可以非常快，但它仍然创建 heap object。

之后可能参与：

- reachability；
- marking；
- sweeping；
- span occupancy；
- fragmentation。

因此消除 hot allocation 的收益可能比一次 `mallocgc` CPU 更大。

## 13. Preallocation

例如：

```go
items := make([]Item, 0, expected)
```

可以避免：

```text
allocate
copy
grow
allocate
copy
grow
```

收益包括 fewer allocation 与 fewer copied bytes。

但过度 preallocation 会增加 retained memory。

Capacity 应来自 realistic distribution。

## 14. Append-Style API

例如：

```go
func Encode(v Value) []byte
```

与：

```go
func AppendEncode(dst []byte, v Value) []byte
```

后者让 caller 控制 storage reuse。

可能减少 allocation/copy，但 API ownership 更显式，也更复杂。

## 15. Object Reuse

Reuse 的形式包括：

- caller-owned buffer；
- reusable scratch；
- `sync.Pool`；
- slab/index storage。

但 reuse 会改变 lifetime/retention。

一个被长期保留的 32 MiB scratch buffer 可能比重复分配普通 64 KiB buffer 更差。

## 16. Allocation Optimization Hierarchy

当 allocation 已被证明是热点，可以优先按下面思路升级：

```text
Can the value stay on stack?
        ↓
Can allocation disappear through API/lifetime design?
        ↓
Can capacity be preallocated?
        ↓
Can storage be caller-owned/reused?
        ↓
Would pool/slab be justified?
```

直接跳到 Pool 经常会掩盖更简单的 lifetime 问题。

## 17. What to Measure

常用证据：

```text
allocs/op
B/op
allocation bytes/sec
object count
heap profile
allocs profile
GC CPU
```

### `allocs/op`

每个 operation 创建多少 heap allocation？

### `B/op`

每个 operation 分配多少 heap bytes？

### allocs profile

哪些 stack 持续制造 allocation traffic？

### heap profile

哪些 object 仍然 live？

## 18. Version Sensitivity

稳定概念：

- allocation rate matters；
- object count matters；
- pointer-free 与 pointer-containing 不同；
- reuse 会交换 churn 与 retention。

具体 threshold、size class、fast path 必须以目标 Go version 为准。

## 19. 相关官方资料

- Go runtime allocator source: https://go.dev/src/runtime/malloc.go
- Go release notes: https://go.dev/doc/devel/release
- Go GC guide: https://go.dev/doc/gc-guide

## 20. 工程视角

Allocator 应被理解为一个高度优化的 memory-management pipeline。

正确问题不是：

> 如何避免所有 allocation？

而是：

> 哪些 heap allocation 产生了可测量的总系统成本？删除或摊薄它们的最简单方式是什么？
