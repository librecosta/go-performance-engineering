# 缓存层级

[English](../../01-cpu-and-memory/cache-hierarchy.md) | 简体中文

## 1. 为什么 CPU 需要 Cache

现代 CPU core 执行 instruction 的速度远快于 DRAM 提供数据的速度。

如果没有 cache，processor 会花大量时间等待 memory。

因此现代硬件通常形成层级：

```text
Registers
   ↓
L1 cache
   ↓
L2 cache
   ↓
Last-level cache
   ↓
DRAM
```

具体 size 与 latency 会随 architecture 和 CPU generation 变化。

真正稳定的工程结论是：

> 数据离执行 core 越近，访问成本通常越低。

## 2. Cache Line

Cache 通常不会按一个 Go field 的大小搬运数据，而是按固定大小的 cache line。

主流 server CPU 上常见 64-byte cache line。

当程序访问：

```go
x := obj.Counter
```

CPU 可能把包含 `Counter` 的整条 line 一起取入 cache。

这既可能带来 spatial locality，也可能让无关变量因为共享同一 line 而发生 false sharing。

## 3. L1、L2 与 LLC

### L1

很小、很快。

hot working set 如果能放进 L1，执行效率通常非常高。

### L2

容量更大、速度更慢，但仍远快于 DRAM。

### Last-Level Cache

LLC 往往由多个 core 或 core group 共享。

它可以减少 DRAM traffic，但也意味着多个 workload 会争用有限容量。

## 4. Cache Hit 与 Miss

Cache hit 表示所需 line 已经存在于对应 cache level。

Miss 则需要向更低层取数据：

```text
L1 miss
 ↓
L2?
 ↓
LLC?
 ↓
DRAM?
```

同一个 Go load instruction 因此可以产生完全不同的实际 latency。

## 5. Spatial Locality

Spatial locality 表示相邻数据通常也会一起被访问。

例如：

```go
type Point struct {
    X float64
    Y float64
}

points := []Point{...}
```

遍历 `points` 时，多个 `Point` 可以一起进入同一个或相邻 cache line。

这也是 `[]T` 在顺序访问场景中很高效的原因之一。

## 6. Temporal Locality

Temporal locality 表示刚刚访问过的数据很可能马上再次使用。

如果一段 state 在多个紧邻操作中重复使用，它可能一直保持 hot。

如果中间扫描巨大无关数据，原本 hot 的 line 可能被 eviction。

## 7. Working Set

Working set 指一次计算中持续活跃的数据集合。

Working set 越小，就越容易留在更快的 cache level。

因此常见思路包括：

- compact structure；
- hot/cold split；
- hot path 不携带不必要 metadata；
- cache-friendly batching。

目标不一定是最小总 memory，而可能是最小 **hot memory**。

## 8. Cache Coherence

不同 CPU core 可以拥有 private cache copy。

当多个 core 访问 shared memory 时，hardware 必须维护 coherent view。

多个 core 只读通常比较容易共享。

写入则需要更强的 cache-line ownership 协调。

这会造成 cache-line movement。

## 9. True Sharing

True sharing 表示多个 core 真正在修改同一个变量。

例如所有 goroutine 都执行：

```go
counter.Add(1)
```

这时所有 core 都争用 counter 所在的 cache line。

在 counter 周围加 padding 不能消除这种本质 sharing。

常见方向是：

- sharding；
- batching；
- local accumulation；
- single-writer aggregation。

## 10. False Sharing

False sharing 表示不同 core 修改不同变量，但这些变量恰好位于同一条 cache line。

例如：

```go
type Counters struct {
    A atomic.Uint64
    B atomic.Uint64
}
```

逻辑上 A/B 独立，物理上却可能因为共享 line 而互相干扰。

这是 layout problem。

可能的解决方式：

- 分离 field；
- cache-line padding；
- redesign shard layout。

## 11. Cache Pollution

大规模 streaming access 可能把 hot control data 从 cache 中挤出去。

因此 hot/cold separation 和 working-set design 很重要。

某些 runtime/library 会对大块 copy 使用 specialized memory primitive。

Application code 通常应该依赖标准库/runtime，而不是自己实现低级 instruction trick。

## 12. Go Data Structure 与 Cache

### `[]T`

通常让 value 连续存储。

### `[]*T`

pointer 连续，但对象本身可能散落在 heap。

### Map

访问更随机，可能触及多个不同 memory location。

### Linked Structure

linked list / pointer-rich tree 会制造 dependent random load。

没有哪一种结构永远更快。

应该根据 access pattern 与 semantics 选择。

## 13. Cache Optimization 本质是 Access-Pattern Optimization

不要只问 object 有多小。

还要问：

- 哪些 field 真正 hot？
- object 是否 sequential access？
- 一次 operation 触及多少 cache line？
- 多个 core 是否在写相邻 field？
- working set 是否大于 cache capacity？

这些问题才真正把 source-level design 与 hardware behavior 连接起来。
