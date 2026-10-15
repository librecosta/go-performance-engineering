# Zero-Copy

[English](../../05-runtime-boundary/zero-copy.md) | 简体中文

## 1. Zero-Copy 真正改变了什么

Zero-copy 常被描述成：

> 不复制 bytes。

更完整的理解是：

> 不复制意味着两层很可能共享同一份 storage，因此 ownership/lifetime 也被绑定。

普通 copy：

```text
source owns A
copy
destination owns B
```

Zero-copy：

```text
source/view
   ↓
same storage
```

所以它首先是 ownership/lifetime optimization，其次才是 memcpy optimization。

## 2. Ordinary Copy Semantics

例如：

```go
s := string(b)
```

语言语义要求 string 后续表现为 immutable string，不能因为 `b` 的普通 mutation 而改变。

Copy 是实现 ownership independence 的直接方式。

## 3. Zero-Copy `[]byte` → `string`

现代 Go 可以：

```go
unsafe.String(
    unsafe.SliceData(b),
    len(b),
)
```

形成：

```text
        backing bytes
        ↑          ↑
     []byte      string
```

潜在减少：

- allocation；
- memcpy；
- bandwidth。

## 4. Immutability Contract

Returned string 语义上 immutable。

因此 backing bytes 在 string reachable 期间必须保持不变。

这种代码会破坏 contract：

```go
b := getBuffer()
s := unsafe.String(unsafe.SliceData(b), len(b))

b[0] = 'X'
use(s)
```

## 5. Pool Reuse Hazard

非常危险：

```go
buf := pool.Get().([]byte)
fill(buf)

s := unsafe.String(unsafe.SliceData(buf), len(buf))

pool.Put(buf[:0])
return s
```

另一个 goroutine 可能 reuse/mutate 同一 buffer。

后果可能包括：

- corrupted output；
- race；
- unstable map key；
- security bug。

## 6. Ownership Transfer

一个可行设计是：

```text
buffer created
↓
all future writers give up access
↓
string view published
```

这里关键不是“保持 bytes 活着”，而是“放弃 mutation right”。

## 7. Borrowed View

也可以限制 lifetime，例如只在 callback 内暴露 view：

```text
view valid only during callback
```

API design 能大幅降低 ownership complexity。

## 8. `string` → `[]byte`

反方向更危险。

Slice type 暗示 writable memory，而 string 语义是 immutable。

Writable zero-copy string→bytes 应视为高风险，并通常避免。

## 9. Read-Only Native View

如果 native API 只读 string bytes，可以用 `unsafe.StringData` 获取 pointer，并在严格 lifetime contract 下调用。

Native code 不能写。

## 10. Retention 是 Zero-Copy 的另一面

假设 parser 收到 16 MiB input，只想保留 100-byte token。

Zero-copy view 可能让整个 16 MiB source 保持 reachable。

Copy 100 bytes 反而能释放 megabytes。

所以：

```text
zero-copy
```

完全可能用更多 memory。

## 11. Lifetime Ratio

Zero-copy 最适合：

```text
source lifetime ≈ view lifetime
```

例如 request buffer 与 token 都只活在一次 parse 内。

最不适合：

```text
source huge
view tiny
view long-lived
```

## 12. Copy Size

16-byte copy 与 16-MiB copy 不在一个量级。

引入 unsafe 前应该测真实 payload distribution：

```text
p50
p90
p99
```

不要只 benchmark maximum payload。

## 13. Memory Bandwidth

Large copy 会消耗 memory bandwidth。

Streaming/high-throughput workload 中，即使 allocation 不明显，bandwidth 也可能成为 bottleneck。

Zero-copy 因此可能比 alloc metrics 显示的收益更大。

## 14. Cache Effect

Copy 也可能带来好处，例如把数据变成 compact hot buffer，顺便进入 cache。

共享 huge source mapping 则可能产生 scattered access/page fault。

所以：

> zero-copy 不等于 zero memory-system cost。

## 15. OS Zero-Copy

Network/storage 语境下的 zero-copy 可能使用 kernel mechanism，避免 userspace intermediate buffer。

这与 `unsafe.String` 是不同机制，不能混为一类。

## 16. mmap 作为 Zero-Copy

mmap 让程序直接访问 file-backed page，而不是先 read 到额外 Go buffer。

它减少 explicit userspace copy，但仍有：

- page fault；
- TLB；
- cache；
- IO；
- lifetime。

## 17. Native Memory View

C library 给出 native pointer 时，可以用 `unsafe.Slice` 建立 Go view。

但 C owner 必须在 view lifetime 内保持 memory valid。

C free/realloc 后，Go slice 就失效。

## 18. Zero-Copy Serialization

很多时候不需要 unsafe。

例如：

```go
func AppendEncode(dst []byte, v Value) []byte
```

可以直接写 caller-owned destination，减少 intermediate allocation/copy，同时保持 safe ownership。

## 19. Scatter/Gather

如果 API 支持多 buffer write，就不需要把：

```text
header + payload
```

先 concatenate 到临时 buffer。

这也是 safe zero-copy thinking。

## 20. Copy-on-Write

COW 在 read-heavy path 中共享 storage，到 mutation 时再 copy。

它减少 copy，却增加：

- shared-state tracking；
- mutation branch；
- synchronization/lifetime complexity。

Go 没有通用自动 COW container，所以需要自行设计 contract。

## 21. Map Key

Mutable-backed zero-copy string **绝不能**作为稳定 map key 发布。

Map hashing 假设 key value semantics 稳定。

Backing bytes mutation 会制造极难解释的行为。

## 22. Concurrent Readers

多个 reader 可以共享 immutable bytes。

真正危险的是 hidden writer。

一个清晰 invariant 是：

```text
many readers
zero writers
```

无法保证就 copy。

## 23. Public API Boundary

Private helper 可以依赖局部 lifetime invariant。

Public zero-copy API 面对任意 caller，风险更高。

更好的设计可能是：

- callback-scoped borrow；
- documented ownership transfer；
- owner object 封装 source lifetime。

## 24. Benchmark Design

比较：

```text
safe copy
vs
zero-copy
```

并覆盖 realistic：

- byte size；
- lifetime；
- concurrency；
- reuse；
- retention。

指标：

```text
ns/op
B/op
allocs/op
RSS/live heap
system throughput
```

## 25. Microbenchmark Trap

如果 safe conversion 100 ns，unsafe 1 ns，看起来 100×。

但它发生在 1 ms operation 中，系统收益仍接近零。

不要只根据 local ratio 接受 unsafe complexity。

## 26. Retention Benchmark

当 view 可能 long-lived 时，应加入 retention experiment。

让 token 保持 live，比较 retained heap。

这可能揭示 copy 反而更好的场景。

## 27. Correctness Tests

需要覆盖：

- empty；
- small；
- large；
- mutation；
- pool reuse；
- lifetime boundary；
- concurrency。

Benchmark 证明速度，test 证明 ownership model。

## 28. Comment

坏：

```go
// Faster zero-copy.
```

好：

```go
// Returns a zero-copy string view over b.
// b must remain immutable and must not be reused while the string is reachable.
```

Unsafe comment 必须保存 safety contract。

## 29. Version Sensitivity

Public unsafe API 是 documented contract，但 ordinary conversion 的 compiler optimization 可能演进。

长期 workaround 需要随 toolchain 重新验证。

## 30. Decision Model

```text
copy measured hot?
      │
     no → keep safe copy
      │
     yes
      ↓
safe API restructuring possible?
      │
     yes → prefer it
      │
     no
      ↓
ownership/lifetime fully controlled?
      │
     no → copy
      │
     yes
      ↓
zero-copy candidate
      ↓
benchmark + retention + correctness validation
```

## 31. 官方资料

- `unsafe`: https://pkg.go.dev/unsafe
- `runtime.KeepAlive`: https://pkg.go.dev/runtime

## 32. 工程视角

Copy 购买的是 **ownership independence**。

当你删除 copy，就必须用明确 lifetime 与 mutation rules 替代这份 independence。

只有当这些 rule 足够简单，而且 data movement 真正昂贵时，zero-copy 才值得。
