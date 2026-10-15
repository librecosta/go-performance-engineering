# 内存局部性

[English](../../01-cpu-and-memory/memory-locality.md) | 简体中文

## 1. Locality 是性能属性

Memory locality 描述程序的访问模式与现代 memory system 的工作方式是否匹配。

好的 locality 可以复用：

- cache line；
- page translation；
- prefetched data。

差的 locality 会增加：

- cache miss；
- TLB miss；
- memory latency；
- stalled execution。

## 2. Contiguous Data

Go 的 slice of values 是获得好 locality 的常见方式：

```go
items := []Item{...}
```

backing array 把 element 连续存储。

顺序遍历：

```go
for i := range items {
    consume(items[i])
}
```

会形成可预测的 address stream，CPU 可以在处理当前 line 时提前 prefetch 后续 line。

## 3. Hardware Prefetch

现代 CPU 会观察访问 pattern。

类似：

```text
address
address + stride
address + 2*stride
address + 3*stride
```

这种规律地址更容易被预测。

hardware 可以提前请求未来 cache line，从而隐藏部分 memory latency。

这意味着：

> 即使数据集大于 L1/L2，只要访问足够规律，也可能保持很高吞吐。

## 4. Memory-Level Parallelism

Processor 可以让多个彼此独立的 memory request 同时在途。

例如：

```text
load A
load B
load C
```

如果彼此没有 dependency，等待时间可以重叠。

而 pointer chain 通常做不到这一点。

## 5. Pointer Chasing

例如：

```go
type Node struct {
    Value uint64
    Next  *Node
}
```

遍历时：

```text
load Node0
 ↓
read Next0
 ↓
now Node1 address is known
 ↓
load Node1
```

下一次 load 地址需要等待上一次结果。

这会限制 prefetch 与 memory-level parallelism。

## 6. `[]T` vs `[]*T`

### `[]T`

可能的优势：

- contiguous values；
- fewer allocations；
- fewer pointers；
- better cache locality；
- pointer-free 时更少 GC scan。

可能的成本：

- large value copy；
- slice growth 时移动数据；
- object identity 不方便。

### `[]*T`

可能的优势：

- stable identity；
- pointer copy 很便宜；
- 适合 shared/polymorphic object。

可能的成本：

- independent heap objects；
- pointer chasing；
- more GC-visible pointers；
- locality 较差。

选择取决于 workload 和 semantics。

## 7. Flat Structure

pointer-heavy tree 有时可以改成 index：

```go
type Node struct {
    Left  uint32
    Right uint32
    Value uint64
}

type Tree struct {
    Nodes []Node
}
```

逻辑仍然是 tree，但 storage 变得 compact/contiguous。

潜在收益：

- fewer heap objects；
- less pointer scanning；
- better locality；
- smaller working set。

代价是更显式的 index 管理。

## 8. Working-Set Locality

一个 hot loop 可能只使用大对象中的少数字段。

例如：

```go
type Connection struct {
    FD       int
    State    uint32
    Flags    uint32
    Username string
    Headers  map[string]string
    Debug    *DebugInfo
}
```

packet path 可能只需要：

```text
FD
State
Flags
```

hot/cold split 可以缩小 hot working set。

即使它引入一个额外 pointer，也可能整体更快。

因此：

> fewer allocations 不等于 better CPU locality。

## 9. Access Order 很重要

同样 Big-O 的两个算法可以因为 cache behavior 差异而表现不同。

例如：

```text
process all fields for object 1
process all fields for object 2
```

与：

```text
process field A for all objects
process field B for all objects
```

分别偏向 object locality 和 field locality。

这就是 AoS/SoA 选择的基础。

## 10. Sequential vs Random Access

Sequential access 常常 bandwidth-friendly。

Random access 常常 latency-sensitive。

因此：

```text
sequential 1 GiB scan
```

可能比：

```text
random 100 MiB pointer graph
```

更快，即使访问总字节更多。

访问模式经常比数据量本身更重要。

## 11. Locality 与 GC

Flat pointer-free representation 可能同时减少：

- cache miss；
- pointer chasing；
- object count；
- GC scan work。

这也是 data-oriented transformation 经常能跨层获得收益的原因。

## 12. Locality 依赖 Workload

为一种 traversal 优化的结构，可能伤害另一种 traversal。

例如：

- SoA 很适合 bulk numeric processing；
- AoS 很适合一次处理完整 object。

不能只从理论决定 layout，必须测量真实 access pattern。
