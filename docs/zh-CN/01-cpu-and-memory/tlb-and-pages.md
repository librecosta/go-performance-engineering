# TLB 与内存页

[English](../../01-cpu-and-memory/tlb-and-pages.md) | 简体中文

## 1. Virtual Memory

Go 程序访问 virtual address。

OS 通过 page table 把 virtual page 映射到 physical memory。

```text
virtual address
    ↓
page table translation
    ↓
physical memory
```

如果每次 memory access 都完整遍历 page table，成本会非常高。

因此 CPU 会缓存 translation。

## 2. Translation Lookaside Buffer

TLB 缓存 virtual page → physical page 的映射。

TLB hit 让 address translation 很便宜。

TLB miss 则可能触发 page-table walk。

所以 memory locality 同时影响：

- cache；
- address translation。

## 3. Page Locality

Sequential slice scan 中，很多相邻 element 共享相同 page。

一个 page translation 可以被大量访问复用。

随机 pointer-heavy structure 更容易跨越大量无关 page，制造 TLB pressure。

## 4. Object Density Per Page

Compact representation 能让每页容纳更多有效 object。

对象从 32 bytes 降到 16 bytes，理论上可以显著提高 page density。

这可能改善：

- cache density；
- TLB coverage；
- memory bandwidth efficiency。

所以 object size 不只是 RSS 问题。

## 5. Pointer Chasing 与 TLB

Pointer chasing 可能同时叠加：

```text
dependent load
+
cache miss
+
TLB miss
```

大型、分散 allocation 的 graph 尤其容易受影响。

这也是 flat/index representation 有时明显优于 pointer graph 的原因。

## 6. Page Fault

Virtual address space 不意味着所有 page 都已经 resident。

Page 可以按需被物理化。

这与：

- large allocation；
- mmap；
- sparse access；
- historical heap ballast；

都有关。

大 virtual reservation 可以在未触碰时保持较小 RSS。

## 7. Huge Pages

Large page 可以让更少 TLB entry 覆盖更多 memory。

对非常大的 memory workload 可能有帮助。

但代价包括：

- fragmentation；
- allocation behavior；
- latency；
- deployment complexity；
- kernel configuration。

它不应成为默认 Go optimization。

## 8. Transparent Huge Pages

OS 可能自动把 memory promotion 成 huge page。

不同环境配置不同。

依赖 page behavior 的 benchmark 应记录相关 system configuration。

## 9. mmap 与 Page Behavior

mmap 把 file-backed 或 anonymous page 映射进 process address space。

它可能减少 explicit Go heap copy，但性能仍取决于：

- page fault；
- access pattern；
- page cache；
- mapping lifetime；
- working set。

所以 mmap 是 OS memory design，而不只是 API trick。

## 10. TLB Optimization 通常就是 Data-Layout Optimization

Application code 很少直接操作 TLB。

通常的杠杆是：

- compact data；
- contiguous storage；
- fewer scattered allocations；
- predictable traversal。

只有 specialized workload 才应该继续深入 huge page / NUMA tuning。

## 11. 工程原则

面对大型 memory workload，不要只问：

> 使用了多少 bytes？

还应该问：

- 有多少 object？
- 跨多少 page？
- access 是 sequential 还是 random？
- representation 是否 compact？
- 是否不断跨 unrelated page？

Memory size 与 memory topology 是两个不同维度。
