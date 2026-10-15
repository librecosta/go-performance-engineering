# sync.Pool

[English](../../04-memory-and-gc/sync-pool.md) | 简体中文

## 1. 目的

`sync.Pool` 用于 temporary reusable object。

典型候选：

- temporary buffer；
- encoder/decoder；
- scratch object；
- formatting state。

它不是 persistent cache。

## 2. Pool 为什么可能有帮助

没有 reuse：

```text
request
→ allocate scratch
→ use
→ scratch dies
→ repeat
```

使用 Pool：

```text
Get
→ use
→ reset
→ Put
```

可能减少：

- allocation；
- allocation bytes/sec；
- GC cycle；
- allocator CPU。

## 3. Semantics 故意很弱

Pool entry 可以在任何时间消失。

程序必须始终正确处理：

```text
Get
→ pool empty
→ allocate new object
```

Pool 不能成为 essential state 唯一 owner。

## 4. Pool 不是 Cache

Cache 对 retention 有某种承诺。

Pool 没有。

不要用 `sync.Pool` 保存：

- session；
- durable lookup；
- 必须存在的 object。

## 5. Reset Before Reuse

Pooled object 带有旧 state。

例如：

```go
buf := pool.Get().(*bytes.Buffer)
buf.Reset()
```

Custom object 也必须清理必要 state。

否则可能意外保留：

- previous request data；
- large slice；
- large graph pointer。

## 6. Capacity Retention

Slice 即使：

```text
len=0
```

也可能：

```text
cap=huge
```

Pool 很容易因此保留 oversized backing array。

这是最重要的 Pool hazard 之一。

## 7. Drop Oversized Object

常见 policy：

```go
if cap(buf) <= maxPooled {
    pool.Put(buf[:0])
}
```

含义：

```text
common-size buffers
→ reuse

outlier buffers
→ let GC reclaim
```

Threshold 不能 blindly copy。

## 8. Tiny Cheap Object 不一定值得 Pool

现代 small allocation fast path 很快。

Pool operation 也有 overhead，并增加 ownership complexity。

Rare tiny object 可能直接 allocation 更简单、更快。

## 9. Large Scratch Buffer

更适合 pool 的对象通常：

- allocation frequent；
- size relatively stable；
- reset cheap；
- retained normal capacity fits budget。

收益可能来自同时避免 allocation 和 repeated zeroing/growth。

## 10. Pool 与 GC

`sync.Pool` 与 GC behavior 集成，但 application 只应该依赖 documented semantics，而不是 private per-P implementation。

这也是为什么它适合 opportunistic reuse。

## 11. Pool 与 Contention

Global Pool 内部做了优化，但重度使用仍可能影响 concurrency/cache。

不能假设 pooling 自动提高 scalability。

## 12. Custom Pool 的风险

自己实现 freelist/pool 可能引入：

- global atomic；
- lock contention；
- false sharing。

Custom allocator 可能比原本 small allocation 更差。

## 13. Pool 与 Lock-Free Reuse

Lock-free data structure 中，node reuse 可能增加 ABA。

因此 allocation optimization 会改变 concurrency correctness assumption。

## 14. Pointer-Heavy Object

Reuse pointer-rich object 时，旧 pointer 可能保留 graph。

Reset 时可能需要：

```go
obj.Big = nil
obj.Items = obj.Items[:0]
```

是否保留 capacity 要由 policy 决定。

## 15. Sensitive Data

Buffer reuse 可能残留旧内容。

跨 tenant/security boundary 时，可能需要显式 clear，而不只是 `[:0]`。

性能不能违反 confidentiality。

## 16. Benchmark Pool

Compare：

```text
without pool
vs
with pool
```

使用 realistic：

- object size；
- concurrency；
- burst；
- outlier capacity。

Measure：

```text
ns/op
B/op
allocs/op
GC CPU
RSS
```

只看到 allocs/op 降低不够。

## 17. Pool Lifecycle

最好的 pooled object 通常：

```text
short logical lifetime
repeated shape
cheap reset
```

Irregular ownership/long lifetime object 通常不适合。

## 18. 常见误解

### "`sync.Pool` 是 object cache"

错误。

### "Pooling 一定省 memory"

错误，可能 retention 更多。

### "0 alloc/op 一定更好"

错误。

### "一个 benchmark 有收益就全部 pool"

错误。

## 19. 官方资料

- `sync.Pool`: https://pkg.go.dev/sync#Pool
- Go GC guide: https://go.dev/doc/gc-guide
- Runtime allocator: https://go.dev/src/runtime/malloc.go

## 20. 工程视角

Pool 是 churn-control mechanism。

正确问题是：

> repeated temporary allocation 是否真的昂贵？这类 object 能否在不保留过多 state/memory 的前提下安全 reuse？
