# mmap

[English](../../05-runtime-boundary/mmap.md) | 简体中文

## 1. mmap 改变了什么

传统 read path：

```text
file
 ↓
kernel page cache
 ↓
read syscall
 ↓
Go buffer
```

Memory mapping 则把 file-backed page 映射到 process virtual address space：

```text
file-backed pages
       ↓
process virtual address space
       ↓
direct memory access
```

这可以减少 explicit userspace read buffer/copy。

## 2. mmap 是 OS Memory Mechanism

mmap 不属于普通 Go heap optimization。

它涉及：

- page fault；
- page cache；
- TLB；
- virtual memory；
- file lifetime；
- mapping lifetime。

Go 只是访问 mapped bytes。

## 3. Mapping 不等于全部 Resident

Mapping 一个 100 GiB file，不等于立刻占 100 GiB RSS。

Page 通常在访问时才 resident。

所以：

```text
virtual mapping size
```

与：

```text
resident memory
```

必须分开。

## 4. Page Fault

首次访问 page 可能 fault。

如果 page 不在 memory，还需要 storage IO。

因此 sequential warm scan 与 random cold scan 行为完全不同。

## 5. Page Cache

File-backed mapping 通常仍经过 OS page cache。

Zero-copy 只是在 application 层少一个 explicit buffer/copy，并不意味着 storage stack 完全没有数据移动。

## 6. Sequential Access

Sequential mapping access 常常受益于：

- predictable page faults；
- OS readahead；
- hardware prefetch；
- good TLB/page locality。

## 7. Random Access

Huge mapping random access 可能产生：

- page fault；
- TLB miss；
- cache miss；
- storage IO。

mmap 不自动比 buffered read 快。

## 8. Mapping Lifetime

Go view 只能在 mapping 仍 valid 时使用：

```text
mmap
 ↓
views
 ↓
all views stop
 ↓
munmap
```

反过来就是 use-after-unmap。

## 9. Zero-Copy String View

如果在 mmap 上构造 `unsafe.String`：

```text
string lifetime
≤
mapping lifetime
```

必须严格成立。

Unmap 后 string 直接变成 invalid pointer view。

## 10. Ownership

Mapped memory 不属于 Go heap。

Mapping object/resource 应该拥有 lifetime。

更安全的 API 会封装：

```go
type Mapping struct {
    data []byte
}
```

并限制 Close 后继续使用 view。

## 11. Mutable Mapping

Writable mapping 让 shared state 跨越：

- Go code；
- page cache/file；
- possibly other process。

它的 concurrency/durability semantics 比普通 Go slice 复杂得多。

## 12. Read-Only Mapping

Read-only mapping 更容易推理。

典型用途：

- immutable index；
- static asset；
- large binary dataset。

也更适合 read-only zero-copy view。

## 13. File Truncation / External Mutation

Backing file 可能被外部修改/truncate。

不同 OS behavior 不同，访问失效 region 可能触发严重 fault。

持有 Go slice 并不意味着 backing file lifetime 已固定。

## 14. Offset Alignment

Mapping API 经常要求 aligned offset。

Wrapper 可以内部：

```text
map aligned base
+
slice to logical region
```

不要把这种 OS detail 泄露到整个 application。

## 15. RSS Accounting

Mapped resident page 会影响 process RSS，但不属于 ordinary Go heap。

所以：

```text
Go heap profile small
```

同时：

```text
process RSS large
```

对 mmap-heavy 程序是正常现象。

## 16. GOMEMLIMIT

GOMEMLIMIT 不是 arbitrary mapped memory 的 hard cap。

使用巨大 mapping 的程序必须做 process-level memory planning。

## 17. GC Interaction

GC 看到引用 mapped bytes 的 Go header/slice，但不会把 mapped data 当作普通 heap object graph 扫描。

这使 huge read-only dataset 在 GC scan 上很便宜，但 physical memory 成本仍然存在。

## 18. 不要在 Mapping 中存 Raw Go Pointer

Persistent/mapped format 应使用：

- offset；
- index；
- encoded ID；

而不是 raw Go heap pointer。

这更 relocatable，也避免 GC/lifetime 问题。

## 19. Offset-Based Data Structure

mmap 很适合：

```text
base address
+
offset
→ record
```

这种 representation。

它与 heap 中 pointer→index 的思路相似。

## 20. Copy 仍然可能更好

从 huge mapping 中 copy small hot working set 到 compact Go memory，有时反而更快：

- better locality；
- no repeated page faults；
- simpler lifetime。

mmap 与 copy 并不是二选一。

## 21. Warm vs Cold Benchmark

mmap benchmark 必须区分：

```text
cold pages
vs
warm page cache
```

否则第二次跑可能只是 memory benchmark，而不是 storage benchmark。

## 22. OS Readahead

Sequential access 可能触发 readahead，random access 可能击穿它。

Benchmark 应记录 OS/kernel context。

## 23. Huge Mapping

64-bit process 有大量 virtual address space，但 page-table/TLB cost 仍然存在。

Virtual size 不能等同 physical memory。

## 24. TLB

Large random mapping 可能明显增加 TLB pressure。

这通常通过 data representation/access pattern 优化，而不是 Go pointer trick。

## 25. Prefetch / Advice

OS 可能提供 access-pattern hint/advice。

这些属于 platform-specific optimization，只应在 page-fault evidence 明确时使用。

Correctness 不应该依赖 hint 被采纳。

## 26. Synchronization

Immutable mapping 多 reader 可以几乎不需要同步。

Writable mapping 则需要同时考虑 Go logical invariant 与 OS/file visibility semantics。

## 27. Unmapping

最简单安全 lifecycle：

```text
Open
↓
Use
↓
Stop workers
↓
Close/Unmap
```

Reference counting/borrowed API 可以做，但复杂度更高。

## 28. Finalizer 不是 Lifetime Control

不能依赖 finalizer 精确决定 unmap 时间。

Mapping correctness 应使用 explicit Close/Unmap。

## 29. Go API Choice

Go standard library 没有统一 portable high-level mmap abstraction。

Unix 项目常使用 `golang.org/x/sys/unix` 或 platform wrapper。

OS-specific code 应隔离。

## 30. Cross-Platform Difference

Linux、BSD/macOS、Windows mapping semantics 存在差异。

声称 portable 的 package 必须做 platform-specific implementation/testing。

## 31. Error Handling

Mapping 可能因为：

- alignment；
- permission；
- address-space/resource limit；
- file state；

失败。

它是 resource acquisition operation，需要完整错误处理与 cleanup。

## 32. 适用场景

较适合：

- large immutable file；
- database/index pages；
- static lookup table；
- large random-read dataset。

不太适合：

- tiny file；
- short-lived data；
- map 完马上全部 copy；
- unclear mutable ownership。

## 33. Performance Evidence

Measure：

- throughput；
- latency；
- page fault；
- RSS；
- CPU；
- storage IO；
- warm/cold behavior。

不要只在 fully warm mapping 上 benchmark `data[i]`，然后宣称 storage performance 更好。

## 34. 相关资料

- `golang.org/x/sys/unix`: https://pkg.go.dev/golang.org/x/sys/unix
- `unsafe`: https://pkg.go.dev/unsafe
- target OS mmap documentation

## 35. 工程视角

mmap 把 explicit buffer ownership 转换成 virtual-memory ownership。

它可能减少 copy 与 Go-heap pressure，但代价转移到：

```text
page faults
page cache
TLB
mapping lifetime
OS behavior
```

只有 data representation 与 access pattern 也适合这个模型时，mmap 才真正有价值。
