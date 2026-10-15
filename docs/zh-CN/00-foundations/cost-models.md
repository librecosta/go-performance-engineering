# 成本模型

[English](../../00-foundations/cost-models.md) | 简体中文

## 1. 为什么需要 Cost Model

性能优化只有在我们知道“改变了什么成本”时才有意义。

没有成本模型，很容易把相关性当成因果。

例如：

```text
CPU profile shows atomic.Add
```

并不能证明 atomic implementation 本身低效。

真实成本可能来自 cache-line contention。

类似地：

```text
heap allocations increased
```

也不自动意味着 allocator CPU 是问题。

真正的问题可能是 allocation rate 推高了 GC frequency。

Cost model 的作用，就是把模糊现象变成可验证机制。

## 2. CPU Cost Model

一个简化的 CPU 成本模型包括：

```text
instructions
+
dependency chains
+
pipeline stalls
+
cache misses
+
branch misprediction
+
memory latency
```

源码“操作数更少”并不自动意味着机器执行更快。

现代 CPU 可以在一个 cycle 中执行多个彼此独立的 instruction。

如果 cycles 很高但 instruction count 接近，可能存在：

- cache miss；
- branch miss；
- data dependency；
- synchronization；
- memory stall。

同一个 Go load instruction，如果数据在 L1 和在 DRAM，物理成本可以差很多。

## 3. Cache Cost Model

CPU 以 cache line 为单位搬运数据。

可以把过程粗略理解成：

```text
address
  ↓
cache line
  ↓
core-local cache
```

性能受到这些因素影响：

- line 是否已经在 cache；
- 相邻数据是否也有用；
- 其他 core 是否在写同一 line；
- line 是否不断被 eviction。

由此产生两个核心概念：

### Locality

好的 locality 减少昂贵 memory traffic。

### Coherence

shared write 需要不同 core 协调 cache-line ownership。

这正是 atomic contention 和 false sharing 的物理基础。

## 4. Memory Cost Model

内存成本远不只是 allocation bytes。

需要同时考虑：

```text
allocation count
allocation bytes
working-set size
retained memory
fragmentation
memory bandwidth
locality
```

大量 tiny allocation 可能增加 allocator 与 GC object overhead。

高 bytes/sec 会迅速消耗 heap headroom，并提高 GC cycle frequency。

Retention 则可能让逻辑上很小的对象长期占有很大的 backing storage。

一些 workload 的瓶颈甚至是 memory bandwidth，而不是 instruction throughput。

## 5. GC Cost Model

Go GC 可以从这些维度理解：

```text
allocation rate
live heap
scannable heap
pointer density
roots
memory-limit pressure
```

### Allocation Rate

高 allocation rate 会快速耗尽 GC cycle 之间的 heap headroom。

### Live Heap

长生命周期对象必须跨 GC 保持存在，决定最低 memory footprint，并影响 tracing work。

### Scannable Heap

pointer-free memory 与 pointer-rich object graph 的 GC 成本不同。

### Pointer Density

更多 Go pointer 意味着更多可能需要检查的 edge。

### Object Graph Shape

flat representation 往往比深度 linked graph 更有 locality。

## 6. Concurrency Cost Model

并发会引入顺序代码中不存在的成本：

```text
shared state
   ↓
synchronization
   ↓
coherence / waiting / retry
```

重要变量包括：

- lock hold time；
- arrival rate；
- number of contenders；
- atomic RMW frequency；
- retry rate；
- goroutine parking；
- scheduling；
- ownership topology。

Mutex 并不会因为存在就自动昂贵。

一个有用的直觉近似是：

```text
contention pressure
≈ arrival rate × critical-section duration
```

## 7. Atomic Cost Model

atomic Load、Store、RMW 与 CAS 的成本不同。

关键区别不是简单的：

```text
atomic vs mutex
```

而是：

```text
read-only shared access
vs
shared write ownership
```

多个 core 高频 increment 一个 shared counter，会反复争夺同一个 cache line。

这是一种硬件层序列化。

## 8. Compiler Cost Model

Compiler 只有在能证明安全时才能删除工作。

例如：

```text
cannot prove bounds
→ retain bounds check

cannot prove lifetime
→ heap allocation

cannot determine dynamic receiver
→ indirect interface dispatch

cannot inline
→ less context for later optimization
```

因此很多 compiler optimization 本质上是 proof problem。

源码不仅表达行为，也表达 optimizer 能否推导出的事实。

## 9. Abstraction Cost Model

Abstraction 可能带来：

- indirect call；
- interface conversion；
- boxing；
- less inlining；
- weaker escape analysis；
- allocation。

但 compiler 也可能完全消除这些成本。

所以 abstraction cost 应从 generated code 与 measured behavior 判断，而不是从语法猜测。

## 10. Operating-System Cost Model

Runtime 还会与这些系统机制交互：

- virtual memory；
- page；
- syscall；
- thread scheduling；
- file descriptor；
- page cache；
- mmap；
- network stack。

因此：

```text
RSS
≠
Go heap
```

以及在大量 cgo/mmap 场景下：

```text
GOMEMLIMIT
≠
hard process RSS cap
```

## 11. Latency Cost Model

一个请求的 latency 可能花在：

```text
running
waiting for CPU
waiting for lock
waiting for channel
helping GC
waiting for syscall
waiting for network
```

这也是为什么 CPU profile 无法单独诊断所有 latency 问题。

## 12. Cost Interaction

成本经常相互影响。

一个 pointer-heavy representation 可能同时增加：

- allocation count；
- GC scan；
- cache miss；
- TLB pressure；
- memory footprint。

反过来，一个 compact representation 可能一次性改善多个层面。

高价值优化往往来自 representation 或 ownership 变化，而不是局部 instruction trick。

## 13. 正确的问题

不要只问：

> 哪个技术更快？

应该问：

> 当前 workload 的 dominant cost 是什么？哪种 representation 或 execution model 能以可接受的 trade-off 去掉这个成本？

这就是性能工程的基础。
