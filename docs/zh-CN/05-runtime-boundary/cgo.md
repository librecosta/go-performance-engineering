# cgo

[English](../../05-runtime-boundary/cgo.md) | 简体中文

## 1. cgo 改变了什么

cgo 让 Go 调 C，也允许 C 与 Go 交互。

它同时跨越：

```text
Go type system
Go GC
Go scheduler
C ABI
native ownership
```

所以成本不仅是 CALL。

Runtime 需要维持 Go correctness model，而 foreign environment 不理解 Go pointer/goroutine。

## 2. 什么时候适合 cgo

常见原因：

- native system library；
- existing C/C++ SDK；
- codec；
- database library；
- crypto；
- hardware/runtime integration。

第一性能问题应该是：

> 这个 boundary 是否必要？应该以什么 granularity crossing？

而不是先问怎么让 cgo call 更快。

## 3. Boundary Overhead

cgo call 一般比普通 Go call 贵。

可能涉及：

- execution-state transition；
- scheduler/runtime invariant；
- pointer rule；
- callback preparation。

但 Go release 会持续优化 overhead，所以旧 benchmark 不能当固定常数。

## 4. Call Granularity

一个常见错误是为 tiny work 高频 crossing：

```text
for each byte:
    call C
```

Boundary overhead 很容易 dominate。

更合理：

```text
batch many operations
↓
one native call
```

## 5. Amortization

C function 做 10 ms heavy work，boundary overhead 几乎可忽略。

C function 只做 20 ns tiny work，boundary 可能成为主成本。

因此要看 work-per-call ratio。

## 6. Go Pointer 与 C

C 不参与 Go GC reachability。

如果 C 在不符合规则的情况下长期保存 Go pointer，会破坏 lifetime/GC invariants。

Go 因此定义严格 pointer-passing rules。

## 7. Temporary Borrow

一个 Go pointer 只在当前 cgo call duration 内给 C 使用，比“C 长期保存”简单得多。

重点是：

> C 是否在 call 返回后继续 retain pointer？

## 8. Retained Go Pointer

如果 C 必须在 call 结束后仍保留 Go address，需要 explicit pinning lifetime。

现代 Go 提供 `runtime.Pinner`：

```text
Go object
 ↓
Pin
 ↓
C retains pointer
 ↓
C stops using
 ↓
Unpin
```

## 9. `runtime.Pinner`

Pinner 提供 documented contract：native code 在 pin lifetime 中可以 retain address。

不要因为当前 Go heap 通常 non-moving，就依赖 implementation accident。

## 10. Nested Pointer

Pin outer object 不会自动让内部所有 Go pointer 都合法供 C 长期 dereference。

Foreign-visible Go memory graph 应尽可能简单。

## 11. `runtime/cgo.Handle`

很多时候 C 根本不需要真实 Go address。

它只需要一个 opaque identity，稍后再交回 Go。

可以：

```text
Go value
 ↓
NewHandle
 ↓
opaque handle
 ↓
C stores handle
 ↓
Go receives handle
 ↓
Value()
 ↓
Delete()
```

## 12. Handle Lifetime

`cgo.Handle` 在 `Delete` 前会占 runtime registry resource。

所以必须 explicit lifecycle。

不 Delete 就是 leak。

## 13. Go Value Containing Pointers

如果 Go value 自身含很多 pointer，把 raw memory 暴露给 C 很危险。

Handle 能让真实 object graph 留在 Go side。

## 14. C Memory Used by Go

反方向：

```text
C allocates buffer
↓
Go creates view
```

可以用 unsafe view。

关键 invariant：

```text
C must not free/realloc while Go view exists
```

Go GC 不管理这块 memory。

## 15. Native Allocation 与 RSS

C allocation 不属于 ordinary Go heap。

所以 Go heap metrics 可能很小，process RSS 却很高。

cgo-heavy service 必须看 process/native memory。

## 16. Callback Direction

C 调 exported Go function 时，runtime 需要额外 boundary preparation。

Per-item callback 往往昂贵。

如果可以返回 batch，通常更容易优化。

## 17. `#cgo nocallback`

Current cgo 支持 `#cgo nocallback`，用于明确保证某 C function 永远不会 callback Go。

它让 runtime/toolchain 跳过部分 callback preparation。

这是严格 contract，不是 casual hint。

如果声明错误，程序行为可能失败/panic。

## 18. `#cgo noescape`

`#cgo noescape` 告诉 compiler：

- 该 C function 不保存传入 Go pointer；
- 不把它回传给 Go。

这可以改善 escape decision。

如果 contract 是假的，可能导致 dangling stack pointer、memory corruption 或 crash。

## 19. noescape 是 Correctness Assertion

它的意思不是：

> 请尽量快。

而是：

> 我保证 foreign implementation 满足这个 lifetime fact。

Compiler 会据此做 stack/heap decision。

## 20. Native Threading

C library 可能拥有自己的 native thread/concurrency model。

性能可能涉及：

- OS scheduling；
- thread affinity；
- callback interaction；
- thread count。

不能只把 cgo 理解成 function call overhead。

## 21. Blocking Native Call

Long C call 会 block 当前 native thread。

Go scheduler 可以让其他 goroutine 在其他 thread 上运行，但大量 blocked cgo 仍会影响：

- OS thread count；
- scheduler behavior；
- resource usage。

## 22. cgo 与 GC

Go-owned 与 C-owned memory 是两套 lifetime model：

```text
Go-owned
→ GC managed

C-owned
→ native/manual managed
```

跨 boundary 的 bridge 必须显式。

## 23. Copy vs Native View

Copy C data 到 Go memory 的成本：

- allocation；
- memcpy。

收益：

- independent lifetime；
- GC ownership；
- simpler concurrency；
- no dangling native view。

只有 copy 真实昂贵且 lifetime 足够清晰时，zero-copy native view 才更合理。

## 24. String Conversion

C string 与 Go string representation/lifetime 不同。

Hot native loop 中重复 per-item conversion 可能产生明显 overhead。

Batching 或 stable representation 往往更重要。

## 25. Error Handling

Native error 可能来自：

- return code；
- errno；
- callback；
- allocated error object。

Performance wrapper 可以减少不必要 conversion/allocation，但 correctness/clarity 仍优先。

## 26. Batching

最通用的 cgo optimization 经常就是：

```text
N Go→C transitions
```

变成：

```text
1 transition
N units of work inside C
```

## 27. Avoid Chatty API

如果 C API 有大量 tiny getter，可以在 wrapper 可控时考虑一次 native call 填充 result structure，减少 crossing。

前提是 ABI/ownership 清晰。

## 28. Benchmark cgo

拆开比较：

```text
boundary overhead
native work
copy/conversion overhead
```

Benchmark：

- empty/minimal C call；
- real representative call；
- batched call；
- copy vs shared buffer。

## 29. Toolchain Version

记录：

- Go version；
- OS；
- architecture；
- C compiler；
- optimization flags；
- native library version。

cgo overhead 会随版本变化。

## 30. Cross Compilation

cgo 依赖 target C toolchain/ABI，会增加 cross compilation/deployment complexity。

这是技术选型的一部分维护成本。

## 31. Race / Sanitizer Boundary

Go race detector 不自动理解所有 native synchronization。

C side 可能需要自己的 sanitizer/testing。

Go race clean 不证明 C side race-free。

## 32. Encapsulation

建议把 cgo 封在小 package：

```text
internal/native/
    cgo wrapper
    pointer/lifetime policy
        ↓
ordinary safe Go API
```

不要让 foreign lifetime semantics 扩散到整个项目。

## 33. Comment 与 Contract

`#cgo noescape` / `nocallback` 等必须解释为什么 C implementation 满足 contract。

Native library future change 可能让 assumption 失效。

## 34. 官方资料

- cgo: https://pkg.go.dev/cmd/cgo
- `runtime.Pinner`: https://pkg.go.dev/runtime#Pinner
- `runtime/cgo.Handle`: https://pkg.go.dev/runtime/cgo

## 35. 工程视角

最有效的 cgo optimization 通常不是 pointer trick，而是设计一个粗粒度、明确的 boundary：

```text
few crossings
clear ownership
simple pointer graph
batch work
```

Crossing 越少、lifetime 越清楚，越容易优化与维护。
