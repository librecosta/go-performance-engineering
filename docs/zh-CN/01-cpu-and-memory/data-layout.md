# 数据布局

[English](../../01-cpu-and-memory/data-layout.md) | 简体中文

## 1. Logical Structure 与 Physical Structure

Go type 表达逻辑数据关系。

CPU 看到的是物理 memory。

例如：

```go
type User struct {
    ID      uint64
    Active  bool
    Name    string
    Profile *Profile
}
```

程序员看到 field。

Runtime/hardware 看到：

- size；
- alignment；
- padding；
- pointer；
- cache-line placement。

对于大量 hot object，physical layout 可以成为一等设计问题。

## 2. Alignment 与 Padding

Field 必须满足 alignment requirement。

Compiler 可能在 field 之间或 struct 末尾插入 padding。

因此 field order 会影响总 size。

更小 struct 可以提高：

- objects per cache line；
- objects per page；
- cache capacity；
- allocator density。

但只有规模足够大时才有工程意义。

为两个 object 省 8 bytes 没意义；为数千万 object 省 8 bytes 可能很重要。

## 3. Cache-Line Placement

由不同 core 独立写入的 field 如果共享一条 line，仍然会互相影响。

这就是 false sharing。

例如：

```go
type WorkerStats struct {
    Requests atomic.Uint64
    Errors   atomic.Uint64
}
```

如果不同 worker 高频写不同 field，物理邻接可能成为问题。

Padding 能隔离 line，但会增加 memory。

因此它更适合：

```text
few
hot
shared
write-heavy
```

类型。

## 4. Pointer Structure vs Index Structure

例如：

```go
type Node struct {
    Left  *Node
    Right *Node
    Meta  *Meta
}
```

会带来：

- pointer chasing；
- GC scan；
- independent heap objects。

如果 node 存在一个 contiguous array 中，可以考虑：

```go
type Node struct {
    Left  uint32
    Right uint32
}
```

这可能降低多个成本。

代价是显式 index 与 sentinel semantics。

## 5. Array of Structures

AoS：

```go
type Particle struct {
    X, Y   float32
    VX, VY float32
    Life   float32
}

particles []Particle
```

memory 近似：

```text
X Y VX VY Life | X Y VX VY Life | ...
```

适合一次 operation 使用 object 大多数字段的 workload。

## 6. Structure of Arrays

SoA：

```go
type Particles struct {
    X, Y   []float32
    VX, VY []float32
    Life   []float32
}
```

field 被分别连续存储。

当 workload 是“一次处理同一字段的很多 object”时可能更好。

潜在优势：

- denser hot data；
- sequential access；
- 不读取 unused field，降低 bandwidth。

代价：

- representation 更复杂；
- 多个 slice；
- index/length invariant 更难维护。

## 7. 如何选择 AoS / SoA

真正的问题是：

> Access pattern 是什么？

AoS 更适合：

```text
for each object:
    use most fields
```

SoA 更适合：

```text
for one field:
    process many objects
```

不存在绝对优胜者。

## 8. Hot/Cold Split

假设：

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

network hot path 只用：

```text
FD
State
Flags
```

可以拆成：

```go
type Connection struct {
    FD    int
    State uint32
    Flags uint32
    Cold  *connectionCold
}
```

即使多了一个 pointer/allocation，hot working set 也可能显著缩小。

目标是 cache locality，不是单纯追求最少 allocation。

## 9. Pointer Density 与 GC

Pointer-bearing field 会影响 GC scanning。

Pointer-free region 通常比 dense pointer graph 更便宜。

因此一个 layout change 可能同时影响：

```text
CPU cache
+
GC
```

这也是 pointer→index 能获得跨层收益的原因。

## 10. Pointer Field Ordering

在某些 Go object layout 下，把 pointer-bearing field 放在前面，可能减少 collector 需要描述/扫描的 pointer-containing prefix。

这是 runtime-version-sensitive behavior。

因此不能把它变成通用代码风格。

只有在 scan-heavy、high-volume type 上通过 benchmark 验证后才值得采用。

## 11. Size Class

Go allocator 会把 small object 放入 size class。

一个很小的 struct size 变化有时会跨越不同 size-class boundary，从而放大 memory footprint 差异。

普通代码不要围绕这些 threshold 设计。

只有 heap profile 证明 high-volume type 确实受影响时才值得考虑。

## 12. Compact Numeric Type

如果 value range 允许，把字段从 64-bit 缩小到 32-bit 可能减少 struct size。

潜在收益：

- better cache density；
- better TLB coverage；
- lower memory footprint。

但是 semantic range 永远优先于 density。

不能为了省 memory 破坏 correctness。

## 13. Layout 与 API Design

Representation 改变可能泄露到 API semantics，例如 pointer→index 会影响：

- identity；
- lifetime；
- mutation；
- error handling。

尽量把底层 representation 隐藏在 stable API 后面。

## 14. Maintainability

不寻常 layout 很容易被“清理”。

例如：

```go
// Child links are indices intentionally. Keeping node storage pointer-free
// reduces GC scanning and pointer chasing during bulk traversal.
type node struct {
    left  uint32
    right uint32
}
```

重要 invariant 必须被记录。

## 15. 工程原则

Data layout 是多个性能层的交汇点：

```text
struct size
    ↓
cache density
    ↓
page density
    ↓
TLB
    ↓
memory bandwidth

pointer layout
    ↓
GC scan
    ↓
pointer chasing
```

对于 high-volume hot data，representation 本身就是 algorithm 的一部分。
