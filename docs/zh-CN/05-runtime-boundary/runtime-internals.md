# Runtime 与 Compiler Boundary

[English](../../05-runtime-boundary/runtime-internals.md) | 简体中文

## 1. 为什么 Runtime Internal 很诱人

Go runtime 内部包含高度优化机制：

- scheduler；
- allocator；
- GC；
- synchronization；
- per-P state；
- lock-free structure；
- stack management。

性能工程师很容易问：

> Application 能不能直接用同样技巧？

有时 public API 已经提供了对应能力。

更多时候，runtime code 依赖 application 不具备的 private invariant。

理解 runtime 很重要。

复制 runtime private implementation 是另一个风险等级。

## 2. Public Contract vs Private Mechanism

Public API 有 compatibility contract。

Private runtime symbol 没有。

例如 `sync.Pool` 内部可能使用 per-P state，不等于 application 应该 `linkname runtime.procPin` 自己实现一套。

Standard library/runtime 可以同步修改 internal contract，third-party library 不行。

## 3. `runtime.KeepAlive`

Compiler liveness 不等于 lexical scope。

某个 wrapper object 在源码中还“存在”，但 compiler 可能已经判断以后没有 semantic use。

如果它的 cleanup/finalizer 会释放 native resource，可能需要：

```go
runtime.KeepAlive(obj)
```

把 reachability 延长到指定 point。

## 4. Lexical Scope vs Liveness

Function 还没 return，不代表 variable 对 compiler 来说一定 live。

Low-level lifetime-sensitive code 必须理解这个差异。

## 5. KeepAlive 不做什么

`KeepAlive` 不会：

- 修复非法 pointer arithmetic；
- 把 `uintptr` 变成 GC reference；
- 修复 use-after-unmap；
- 提供 synchronization。

它只控制 reachability。

## 6. Cleanup 与 Finalization

GC-triggered cleanup/finalizer 是 nondeterministic。

需要 deterministic lifetime 的资源仍应该 explicit：

```text
Close
Unmap
Release
```

Cleanup 适合作为 defensive fallback。

## 7. `runtime.Pinner`

Pinner 是 public runtime lifetime mechanism。

它提醒我们：

> 有 public contract 时，不要依赖当前 GC implementation accident。

Native code 要 retain Go address，就用 documented pinning model。

## 8. Compiler Directives

性能相关 directive 包括：

- `//go:noescape`；
- `//go:nosplit`；
- `//go:linkname`。

它们安全级别完全不同，不能统称为“compiler hint”。

## 9. `//go:noescape`

通常用于没有 Go body 的 function declaration，例如 assembly implementation。

它告诉 escape analysis：

> pointer argument 不会通过这个 function 逃到 heap/return。

这可能让 caller value 保持 stack lifetime。

## 10. noescape 是 Contract

如果 implementation 实际上保存 pointer：

```text
compiler believes no escape
↓
object stays stack
↓
function returns
↓
external pointer still exists
↓
memory corruption
```

所以 noescape 是 human-provided correctness fact。

## 11. `#cgo noescape`

cgo 也有类似 directive。

风险逻辑完全相同。

## 12. `//go:nosplit`

Normal Go function entry 包含 stack-growth safety check。

`//go:nosplit` 让 compiler 省略普通 stack-overflow check。

Runtime 在某些无法安全 grow stack 的底层位置需要它。

## 13. `nosplit` 不是 `go:fast`

普通 application 为了少一个 check 使用 `//go:nosplit` 是危险的。

它可能导致 stack overflow/corruption。

这是 runtime correctness primitive，不是 hot-path speed hint。

## 14. `//go:linkname`

`//go:linkname` 可以把 local declaration 绑定到另一个 package 的 symbol，包括 unexported/private symbol。

它突破：

- package encapsulation；
- compatibility boundary；
- 某些 type-safety expectation。

Compiler 要求 import `unsafe`，正说明其风险级别。

## 15. Private Runtime Symbol

一些 library 历史上用 `linkname` 访问：

- `runtime.procPin`；
- scheduler helper；
- private hash/allocation internals。

这意味着主动承担：

```text
Go upgrade
↓
private contract changes
↓
build failure / semantic break / corruption
```

## 16. `runtime.procPin`

`procPin` 是 runtime internal mechanism，用于把 goroutine 暂时固定到当前 P，并暴露 P identity。

它让 per-P state 成为可能。

但它不是普通 public application API。

## 17. 为什么 Per-P 可能很快

Per-P state 可以减少：

- global lock；
- shared cache line；
- contention。

概念上：

```text
P0 → local state 0
P1 → local state 1
P2 → local state 2
```

这是很好的 design idea。

不代表 private `procPin` 是合适的第三方实现方式。

## 18. 优先 Public Alternative

在 runtime hack 前，先考虑：

- sharding；
- worker-local state；
- `sync.Pool`；
- batching；
- explicit ownership。

通常可以获得大部分收益，同时保留 compatibility。

## 19. Runtime Noescape Trick

Runtime source 历史上可能使用 obscure pointer relation 的技巧，让 escape analysis 看不到某些关系。

Application 模仿这种行为非常危险。

如果 pointer 真 escape，而 compiler 被“骗”成 noescape，就会直接损坏 memory safety。

## 20. Lying to Compiler

Compiler optimization fact 同时也是 correctness fact。

人为让 compiler 错误相信 noescape，潜在收益只是少一次 allocation，风险是 dangling pointer/memory corruption。

普通 application 不应接受这种 trade-off。

## 21. Tagged Pointer

Runtime lock-free structure 可能把：

```text
pointer identity
+
version/tag
```

压进一个 atomic word。

用于 ABA 等问题。

但它可能依赖：

- pointer width；
- address range；
- alignment；
- non-GC-visible encoding。

属于 runtime-level assumption。

## 22. GC-Visible vs Encoded Address

如果 address 存在：

```text
uintptr / uint64
```

GC 不认为它是 reference。

Runtime 可以用其他 invariant 保证 lifetime，普通 heap object 通常没有。

## 23. Runtime Lock-Free Structure

Runtime code 很适合学习：

- CAS；
- ABA；
- tagging；
- backoff；
- per-P state。

但它不是 application template。

Runtime 控制 memory placement、GC interaction 与 scheduler assumption。

## 24. Backoff

Runtime lock-free algorithm 里的 spin/backoff constant 与 specific workload/hardware 相关。

复制一个数字到不同 workload 不会复制同样性能模型。

## 25. Intrinsic

某些 standard library/runtime function 可能被 compiler 特殊识别并替换成 optimized machine code。

Third-party code 更应该依赖 public API，让 toolchain 自己管理 intrinsic，而不是依赖 private compiler contract。

## 26. Assembly

Assembly 适合 specialized library，并且：

- hot loop 已证明；
- compiler 无法生成所需 instruction；
- architecture-specific maintenance 可接受。

成本：

- per-arch maintenance；
- ABI/toolchain change；
- reduced portability；
- harder review/testing。

## 27. Assembly 与 `//go:noescape`

Compiler 无法完整检查 assembly body，因此常需要 human 提供 noescape fact。

这是 legitimate low-level use，但仍必须 correctness review。

## 28. `//go:noinline`

可用于：

- compiler/runtime constraint；
- benchmark experiment；
- debugging。

普通 performance code 不应为了“快”而阻止 inline。

## 29. 不要固化 Compiler Heuristic

不要长期依赖：

```text
this function always inlines
this private symbol always exists
this threshold never changes
```

除非项目明确绑定 specific Go toolchain。

## 30. Standard Library 的 Privileged Boundary

`sync`、runtime 等 standard package 可以共享 private contract。

这些 contract 不属于 Go compatibility promise。

读源码时永远要问：

> 这是 public mechanism 还是 privileged internal mechanism？

## 31. `internal` Package

Go 自己的 `internal/...` 也是明确“不提供通用外部 contract”的信号。

可以研究，不应该依赖。

## 32. Runtime Source Archaeology

阅读 runtime source 非常有价值，因为它告诉你：

- runtime 认为哪些 cost 值得优化；
- primitive 怎么实现；
- version-sensitive behavior 在哪里。

最有价值的输出通常是：

```text
a better cost model
```

而不是：

```text
copy this private code
```

## 33. Version Upgrade Risk

Private runtime dependency 在 upgrade 时可能导致：

- build failure；
- semantic change；
- corruption；
- performance regression。

这属于主动接受的 toolchain-maintenance burden。

## 34. Encapsulation

如果低层 library 确实必须用 runtime-specific mechanism，应隔离：

```text
internal/runtimehack/
```

并提供 safe higher-level API 与 explicit version tests。

## 35. Comments

Directive/private dependency 必须记录：

- 为什么成立；
- 哪个 implementation 被覆盖；
- 什么 future change 会让它失效；
- 为什么 public alternative 不够。

## 36. Evidence Threshold

层次越低，证据门槛越高：

```text
safe source optimization
→ modest proof

public unsafe API
→ stronger proof

assembly/compiler contract
→ benchmark + correctness

private runtime linkname
→ exceptional justification
```

## 37. Correctness Tooling

可以组合：

- unit test；
- race；
- checkptr；
- architecture CI；
- fuzzing；
- stress；
- assembly inspection。

没有单一工具可以证明 correctness。

## 38. KeepAlive 与 Native Call

`KeepAlive` 的典型意义是保证 wrapper object 在 raw field/native use 完成前保持 reachable。

核心事实：

```text
last lexical-looking use
≠
last compiler-visible lifetime
```

## 39. KeepAlive 与 Unsafe

`KeepAlive` 不能替代 unsafe pointer validity rules。

Lifetime extension 与 pointer validity 是两个不同 invariant。

## 40. 官方资料

- compiler directives: https://go.dev/src/cmd/compile/doc.go
- `runtime.KeepAlive`: https://pkg.go.dev/runtime#KeepAlive
- `runtime.Pinner`: https://pkg.go.dev/runtime#Pinner
- runtime source: https://go.dev/src/runtime/
- runtime lfstack: https://go.dev/src/runtime/lfstack.go

## 41. 工程视角

Runtime internal 最值得带回 application 的通常是 design principle：

```text
reduce sharing
reduce allocation
preserve locality
make lifetime explicit
```

而不是 runtime 为实现这些原则所使用的 private hook。
