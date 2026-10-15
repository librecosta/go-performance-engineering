# Unsafe

[English](../../05-runtime-boundary/unsafe.md) | 简体中文

## 1. 为什么存在 `unsafe`

Go 的 memory-safety model 很强：

- pointer type 受检查；
- slice 有 bounds check；
- object lifetime 由 runtime 管理；
- GC 理解普通 Go reference；
- package boundary 限制 implementation detail。

`unsafe` 用于这些抽象无法满足的场景，例如：

- OS interface；
- mmap；
- serialization/parser internals；
- FFI；
- runtime implementation；
- carefully controlled zero-copy view。

`unsafe` 的意义是：

> 把一部分 compiler/runtime 原本替你保证的 correctness contract 转移给程序员。

它不是“性能开关”。

只有当某个 safe abstraction cost 已被证明显著时，unsafe 才可能成为候选。

## 2. 你绕过了什么 Safety

根据具体操作，unsafe 可能绕过或削弱：

- type identity；
- alignment；
- object bounds；
- immutability；
- ownership；
- lifetime；
- GC visibility。

所以每一段 unsafe code 都必须由程序员重新建立对应 invariant。

例如 zero-copy `[]byte` → `string` 必须自己保证 backing bytes 在 string lifetime 内不可变。

## 3. `unsafe.Pointer`

`unsafe.Pointer` 是 typed pointer 之间的桥梁。

它允许普通 Go type system 拒绝的 conversion。

但它仍然属于 Go pointer/lifetime model。

它不是任意 integer address。

## 4. `uintptr` 不是 GC Reference

非常重要：

> `uintptr` 是 integer，不是 Go pointer。

例如：

```go
p := &obj
u := uintptr(unsafe.Pointer(p))
```

程序员知道 `u` 数值上像地址。

GC 不把它视为 live reference。

如果没有其他 normal Go pointer：

```text
object
 ↓
only integer address remains
```

object 仍然可能失去 reachability。

所以不能把 Go pointer 长期藏在 `uintptr` 中当作 lifetime mechanism。

## 5. Pointer → Integer → Pointer

历史上常见：

```go
unsafe.Pointer(uintptr(p) + offset)
```

这种 arithmetic 必须严格遵守 pointer-validity rules。

不能把 pointer 转成 `uintptr` 存起来很久，之后再假设 object 还活着。

问题不只是地址是否变化，而是 GC visibility 已经丢失。

## 6. `unsafe.Add`

现代 Go 提供：

```go
unsafe.Add(ptr, offset)
```

表达 pointer arithmetic。

它比历史 `uintptr` arithmetic 更明确，但仍然 unsafe。

调用者必须保证：

- allocation boundary；
- alignment；
- representation；
- lifetime。

## 7. `unsafe.Sizeof`

`unsafe.Sizeof` 可以用来研究：

- struct size；
- padding；
- compact representation；
- layout experiment。

但它不等于 total memory footprint。

它不会自动包含：

- pointed-to object；
- slice backing array；
- map storage；
- runtime metadata。

## 8. `unsafe.Alignof`

`unsafe.Alignof` 返回 alignment requirement。

把一个 valid `*byte` 直接解释成 `*uint64` 并不自动安全。

Alignment assumption 在不同 architecture 上尤其重要。

## 9. `unsafe.Offsetof`

`unsafe.Offsetof` 返回 struct field offset。

它适合 interoperability 与 layout validation。

不要因为能计算 offset，就用 pointer arithmetic 替代普通 field access。

## 10. `unsafe.Slice`

`unsafe.Slice(ptr, len)` 从 pointer + length 建立 Go slice view。

适合：

- native buffer；
- mmap region；
- low-level system API。

但 slice 不获得独立 ownership。

Underlying memory 必须在整个 slice lifetime 内保持有效。

## 11. Historical Giant-Array Trick

旧代码常用：

```go
(*[1 << 30]T)(unsafe.Pointer(ptr))[:n:n]
```

这类方式应该主要作为历史理解。

现代 public unsafe API 更清楚，也减少 representation assumption。

## 12. `unsafe.SliceData`

```go
ptr := unsafe.SliceData(buf)
```

可以拿到 slice backing storage 的 data pointer。

Pointer validity 仍然依赖 slice/backing allocation lifetime。

Raw pointer 不是 ownership transfer。

## 13. `unsafe.String`

`unsafe.String(ptr, len)` 可以构造 string view。

这是现代 zero-copy `[]byte` → `string` 的基础。

关键 safety contract：

> backing bytes 在 returned string reachable 期间必须保持不可变。

## 14. `unsafe.StringData`

`unsafe.StringData` 暴露 string data pointer。

它可以用于 read-only native access。

但它不会把 string storage 变成 writable memory。

写入会违反 string semantics。

## 15. Reflect Header Trick

历史代码经常操作：

- `reflect.StringHeader`；
- `reflect.SliceHeader`。

现代代码应优先使用 dedicated `unsafe` API。

Header trick 容易：

- 隐藏 lifetime；
- 产生 non-GC-visible pointer；
- 依赖 representation detail；
- 制造 invalid alias。

## 16. Alignment

amd64 对某些 unaligned access 比较宽容，不代表 Go 跨平台 contract 也允许任意 misalignment。

Low-level reinterpretation 必须考虑 target architecture。

## 17. Bounds

Unsafe pointer arithmetic 可以绕过 slice bounds protection。

错误 pointer 可能指向：

- another object；
- unmapped memory；
- runtime metadata。

Bounds 必须来自可信 representation。

## 18. Aliasing

Unsafe conversion 可以让不同 Go value 引用相同 bytes：

```text
[]byte
  └────┐
       ↓
same backing bytes
       ↑
string ┘
```

Type system 原本允许程序员分别推理 mutation。

Aliasing 改变了这个 assumption。

## 19. Lifetime

Unsafe pointer/view 绝不能比 underlying storage 活得更久。

尤其要注意：

- pooled buffer；
- mmap；
- C memory；
- stack/local storage；
- manually managed region。

Address 数值正确，不代表 storage 仍然有效。

## 20. Ownership

Review unsafe code 时最重要的问题：

> 谁拥有这些 bytes？

可能是：

- Go heap object；
- caller；
- pool；
- mmap；
- C allocator；
- OS resource。

然后继续问：

> owner 什么时候可以 mutate / release？

如果回答不清晰，unsafe optimization 就还没准备好。

## 21. GC Visibility

Typed Go pointer 会告诉 GC object reachable。

`uintptr`/integer representation 不会。

Runtime 内部可能故意使用 non-GC-visible pointer encoding，但它拥有额外 private invariant。

普通 application 不应复制这种机制。

## 22. `runtime.KeepAlive`

`runtime.KeepAlive` 可以让 object 在某个 program point 前保持 reachable。

适合某些：

- finalizer/cleanup；
- syscall；
- native call；

的 lifetime pattern。

但它不能让原本非法的 unsafe pointer operation 变合法。

## 23. Unsafe 与 Escape Analysis

Unsafe 可能改变 compiler 追踪 pointer flow 的能力。

Low-level directive 甚至可以显式声明 noescape。

这类机制是 correctness contract，不是普通 optimization hint。

## 24. Unsafe 与 Race Detector

Unsafe aliasing 可以产生普通 data race。

例如 string view 指向 pooled buffer，而另一个 goroutine reuse/mutate 它。

Race detector 可能捕获部分问题，但不能证明完整 lifetime contract。

## 25. `checkptr`

`checkptr` 类 instrumentation 能检测一些 invalid pointer arithmetic/conversion。

它是 correctness tool，不是 performance baseline。

## 26. `go vet`

`go vet` 可以识别部分危险 unsafe pattern。

通过 vet 不等于 unsafe correctness 已证明。

Lifetime/ownership bug 仍可能存在。

## 27. Cross-Architecture Testing

Unsafe code 必须特别警惕：

- pointer width；
- alignment；
- endianness；
- unaligned load support；
- OS ABI；
- architecture instruction behavior。

支持哪些 architecture，就应该尽量在哪些 architecture 上测试。

## 28. Encapsulation

Unsafe surface 应尽可能小。

健康结构可能是：

```text
internal/zerocopy/
internal/native/
internal/mmap/
```

而不是 unsafe scattered throughout business logic。

## 29. API Naming

Aliased-view API 如果对 caller 有重要 lifetime 影响，应让 contract 可见。

例如：

```go
func UnsafeBytesToString(b []byte) string
```

如果 helper 完全内部化，也至少要在 implementation comment 中保存 invariant。

## 30. Performance Evidence

Unsafe 只有在 safe abstraction cost 真正显著时才值得。

Measure：

- copied bytes；
- allocation；
- CPU；
- bandwidth；
- end-to-end impact。

Microbenchmark 100× 不代表 system 也有价值。

## 31. Maintainability Threshold

Unsafe 的 correctness/maintenance cost 更高，因此 evidence threshold 也必须更高。

概念上：

```text
expected system gain
        ↓
must exceed
        ↓
correctness + maintenance + portability risk
```

## 32. 不应该做什么

不要为了：

- 去掉 compiler 本来就能安全消除的 bounds check；
- cold path 少一次 tiny copy；
- 访问 private runtime state；
- 强迫 object stack allocation；
- 展示“技巧”；

而使用 unsafe。

## 33. Public Unsafe vs Runtime Internal

Public `unsafe` API 虽然危险，但有文档 contract。

`//go:linkname` private runtime symbol 属于更高风险层，因为额外增加 compatibility risk。

## 34. 官方资料

- `unsafe`: https://pkg.go.dev/unsafe
- `runtime.KeepAlive`: https://pkg.go.dev/runtime#KeepAlive
- compiler directives: https://go.dev/src/cmd/compile/doc.go

## 35. 工程视角

Safe Go 把 memory ownership 大量交给 type system/compiler/runtime。

Unsafe 则让程序员自己负责。

它的价值来自删除一个被测量的 abstraction cost。

代价是你必须重新承担那个 abstraction 原本保护的 invariant。
