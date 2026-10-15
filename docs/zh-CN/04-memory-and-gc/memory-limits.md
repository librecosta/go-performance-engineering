# Memory Limit、RSS 与 Scavenging

[English](../../04-memory-and-gc/memory-limits.md) | 简体中文

## 1. 为什么需要 Memory Limit

GOGC 控制 proportional heap growth。

Production system 还经常受到：

- container limit；
- VM size；
- process budget；

这样的 absolute memory constraint。

Go 通过：

- `GOMEMLIMIT`；
- `debug.SetMemoryLimit`；

提供 runtime soft memory limit。

## 2. GOMEMLIMIT 不是 GOGC

### GOGC

控制 normal heap-growth ratio。

### GOMEMLIMIT

控制 absolute soft memory budget 下的 pressure。

两者互补。

## 3. Soft Runtime Memory Budget

Memory limit 约束的是 Go runtime 管理/理解的 memory accounting。

它不能简单解释成：

```text
hard process RSS cap
```

因为 process 还可能有：

- mmap；
- cgo/native allocation；
- external library；
- kernel/OS buffer。

## 4. Container Headroom

如果 container hard limit 是 8 GiB，直接把：

```text
GOMEMLIMIT = 8 GiB
```

通常太激进，因为没有给非 Go runtime memory 留空间。

Headroom 大小取决于：

- cgo；
- mmap；
- stack growth；
- native library；
- observability agent；
- workload。

没有 universal percentage。

## 5. Limit 会成为 GC Pressure Signal

当普通 GOGC pacing 会逼近 memory limit 时，runtime 会增加 GC pressure。

```text
heap grows
↓
approaches limit
↓
collector becomes more aggressive
```

这是 CPU 换 memory。

## 6. Limit 太低

如果 live set 本身接近 limit：

```text
GC
↓
little memory dies
↓
allocation
↓
limit pressure
↓
GC again
```

会进入 thrashing。

症状：

- GC CPU 高；
- cycle 频繁；
- assist 多；
- application progress 低。

真正修复是：

- reduce live set；
- increase budget；
- change representation。

## 7. GOGC 与 GOMEMLIMIT 一起工作

Far below limit 时，GOGC 决定 normal pacing。

接近 limit 时，absolute budget 成为 dominant constraint。

这是：

```text
preferred growth policy
+
maximum soft budget
```

两个维度。

## 8. Disable Ordinary GOGC

可以 `GOGC=off` 或等效 runtime setting，同时保留 memory limit。

这会让 limit 成为主要 GC trigger。

某些固定 budget workload 可能受益，但必须真实 load test。

## 9. RSS

RSS 是 OS resident process memory。

包括的不止 Go heap：

```text
Go heap
Go stacks
runtime metadata
binary/code
mmap
cgo/native memory
```

所以 high RSS 不直接证明 Go heap leak。

## 10. Heap Reclaim vs RSS

GC reclaim dead object 后，slot 可以 reuse。

但 page 可能仍 mapped/resident，因为：

- runtime 预期 reuse；
- span 部分 occupied；
- scavenger 尚未 release。

所以 heap live 与 RSS 会出现 gap。

## 11. Scavenger

Scavenger 把可释放的 physical page 返还 OS。

概念上：

```text
objects die
↓
sweep frees slots
↓
pages become reclaimable
↓
scavenger releases backing
↓
RSS may fall
```

它与 tracing GC 是不同阶段。

## 12. Runtime 为什么保留 Free Memory

保留 free heap 能让后续 allocation 直接 reuse，而不必重新向 OS 请求。

Steady-state server 经常因此更高效。

最大化即时 RSS release 不一定是最佳策略。

## 13. `debug.FreeOSMemory`

`debug.FreeOSMemory` 会 force GC，并尽量把 free memory 返还 OS。

它更适合清晰 phase boundary：

```text
startup/indexing
↓
large temporary heap
↓
steady state
```

不适合做成周期性“内存清理 timer”。

## 14. Fragmentation

如果 page 内：

```text
live
free
live
free
```

memory 可以给 Go reuse，却不一定能作为整页 release。

如果：

```text
HeapAlloc ↓
RSS stays high
```

fragmentation 是一个可能原因。

## 15. Large Object Release

Large allocation 完全死亡后有时更容易形成可释放 page region。

但 large-object churn 仍会产生 RSS volatility。

## 16. mmap

mmap memory 不属于普通 Go heap allocation。

它可以显著影响：

- RSS；
- virtual memory；
- page fault。

GOMEMLIMIT 不是它的完整 accounting/control system。

## 17. cgo / Native Allocator

C/native library 可能 allocation 大量 Go runtime 不知道的 memory。

因此 cgo-heavy service 需要 process-level memory budget。

## 18. Observability

Memory dashboard 应区分：

```text
process RSS
Go heap live
Go heap goal
released heap
allocation rate
GC CPU
external/native memory if available
```

一个 graph 无法解释所有 memory behavior。

## 19. OOM Prevention

GOMEMLIMIT 可以降低 OOM 风险，但不能保证永不 OOM。

尤其当：

- live Go heap 超预算；
- native memory 独立增长；
- mmap dominates；
- container/kernel accounting 不同。

它是 control mechanism，不是 absolute safety guarantee。

## 20. Memory Budget Design

实际预算应该拆成：

```text
Go runtime budget
+
external/native budget
+
operational safety margin
```

比例来自实际 process observation。

## 21. Historical Ballast

Heap ballast 曾经通过人为增大 live heap，解决：

> 机器有很多 memory，但 GOGC 因 live heap 小而 GC 太频繁。

GOMEMLIMIT 能更直接地表达：

> 允许使用可用 memory 降低 GC pressure，但不要超过这个 soft budget。

所以 ballast 现在更适合作为历史背景。

## 22. 官方资料

- GC guide: https://go.dev/doc/gc-guide
- `runtime/debug`: https://pkg.go.dev/runtime/debug
- Runtime scavenger: https://go.dev/src/runtime/mgcscavenge.go

## 23. 工程视角

Memory limit 是 capacity planning 的一部分。

正确问题不是：

> GOMEMLIMIT 应该写多少？

而是：

> Go runtime、native/mmap、true live set 分别需要多少 memory？在压力下可以接受多少 GC CPU？
