# Heap Model

[English](../../04-memory-and-gc/heap-model.md) | 简体中文

## 1. Heap 不是一个数字

当有人说：

> 程序用了 4 GiB memory。

它可能指：

- live Go objects；
- Go heap reservation/capacity；
- runtime metadata；
- goroutine stack；
- free pages retained for reuse；
- mmap；
- cgo/native memory；
- process RSS。

性能工程必须把这些概念分开。

## 2. Live Heap

Live heap 是 GC 后仍然 reachable 的 object memory。

```text
roots
 ↓
reachable objects
 ↓
live heap
```

它直接影响：

- minimum memory requirement；
- GC work；
- heap-growth target。

如果程序真的有 20 GiB live working set，单纯更频繁 GC 不可能把它“清掉”。

## 3. Heap Capacity vs Live Heap

Runtime 通常拥有比 live object bytes 更多的 heap memory。

原因包括：

- span 内 free slot；
- retained reusable page；
- GC headroom；
- fragmentation。

所以：

```text
live heap
<
runtime heap footprint
```

是正常现象。

## 4. Scannable Heap

一个非常关键的区分：

```text
live heap
vs
scannable heap
```

1 GiB `[]byte` backing array 没有 pointer。

1 GiB `[]*Node` 包含大量 reference。

字节数相似，GC tracing cost 可以完全不同。

## 5. Pointer Density

Pointer-bearing Go value 不只是显式 `*T`：

- string；
- slice；
- map；
- channel；
- interface；
- closure/function state；

都可能包含 Go pointer。

所以 struct 没有 `*SomeType` 也不等于 pointer-free。

## 6. Object Graph Shape

Flat array 与 pointer graph 的 tracing locality 不同。

Pointer-heavy graph 可能：

- 更多 edge；
- 更差 memory locality；
- 更多 independent object metadata。

这会把 GC cost 与 CPU cache behavior 连接起来。

## 7. Span Occupancy

假设 span：

```text
live dead live dead live
```

dead slot 可以被 Go reuse。

但只要某些 page 上仍有 live object，整块 physical page 不一定能立即返还 OS。

这就是 fragmentation 的来源之一。

## 8. Internal Fragmentation

Requested object 小于实际 size-class slot，就会有 slot 内浪费。

这是 allocator 用来换取 fast fixed-size allocation 的 trade-off。

大规模 long-lived object 时可能影响 footprint。

## 9. Page-Level Fragmentation

即使 live bytes 很少，如果它们散在很多 partially occupied span/page 中，大区域也不容易完全释放。

所以：

```text
HeapAlloc falls
```

不保证：

```text
RSS falls immediately
```

## 10. Runtime-Reusable Memory

GC reclaim 后的 memory 可能已经能给新的 Go allocation 复用，即使 OS 仍把 page 视为 resident。

这是有价值的。

Reusing owned heap 往往比释放到 OS 再重新获取更便宜。

因此 steady-state server 不应该强迫每次 GC 后都最大化 RSS release。

## 11. RSS

RSS 是 OS 看到的 resident process pages。

它可能包括：

- Go heap；
- Go stacks；
- runtime structures；
- code/data；
- mmap；
- native/cgo memory。

因此：

```text
RSS ≠ heap live bytes
```

High RSS 不自动等于 Go heap leak。

## 12. Virtual Memory

Process 可能 reserve 大量 virtual address space，但没有对应同量 physical residency。

这在 large sparse mapping、mmap、historical heap ballast 等场景很重要。

Virtual size 与 RSS 必须分开。

## 13. Heap Metric 要对应问题

### "谁在分配？"

看 cumulative allocation。

### "谁在保留？"

看 live/inuse heap。

### "为什么 RSS 不降？"

调查：

- retained heap pages；
- fragmentation；
- scavenging；
- stacks；
- mmap/cgo。

不同问题需要不同证据。

## 14. Heap Object vs Stack Object

Stack value 通常不成为 independent heap object。

因此 escape analysis 可以直接影响：

- allocator traffic；
- object count；
- GC metadata。

## 15. Pointer-Free Bulk Storage

例如：

```go
type Node struct {
    Left  uint32
    Right uint32
    Value uint64
}

nodes []Node
```

可以让一个大 backing array pointer-free。

潜在收益：

- lower scan work；
- fewer heap objects；
- better locality；
- better page density。

Representation design 经常比 allocator micro-tuning 更强。

## 16. Large Live Byte Buffers

Pointer-free 不等于 memory 免费。

10 GiB `[]byte` 虽然 GC scan 便宜，仍然消耗：

- RSS；
- bandwidth；
- cache；
- paging risk。

降低 GC scan 不等于降低 physical capacity。

## 17. Heap Growth Headroom

Go 会故意允许 GC cycle 之间的 heap growth。

更多 headroom：

```text
more memory
→ fewer GC cycles
```

更少 headroom：

```text
less memory
→ more GC cycles
```

这是 GC pacing 的核心 CPU/memory trade-off。

## 18. Memory Ownership

Logical size 与 ownership size 不一定相同。

例如：

```text
slice len=100
cap=16 MiB
```

逻辑只有 100 element，却仍拥有整个 backing array。

或者 tiny view 保留巨大 source buffer。

Retention analysis 必须看 ownership，而不是只看 len。

## 19. Version Sensitivity

稳定概念：

- live heap；
- reachability；
- pointer scan；
- reusable heap memory；
- RSS separation。

具体 metric 与 internal organization 会演进。

## 20. 工程视角

不要把“memory”当成一个 scalar。

先判断问题是哪一种：

```text
live objects?
allocation churn?
runtime heap capacity?
retained backing storage?
fragmentation?
RSS?
external memory?
```

然后才能选择正确机制。
