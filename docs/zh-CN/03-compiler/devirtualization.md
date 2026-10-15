# Devirtualization

[English](../../03-compiler/devirtualization.md) | 简体中文

## 1. Interface Dispatch

例如：

```go
func consume(r io.Reader, buf []byte) {
    _, _ = r.Read(buf)
}
```

Static type 是 `io.Reader`。

Runtime concrete value 可能是：

- `*os.File`；
- `*bytes.Reader`；
- network type；
- custom implementation。

所以 generic interface call 使用 dynamic dispatch。

## 2. Direct vs Indirect Call

Direct call 有 known target。

Interface call 可能通过 dynamic method resolution。

Indirect dispatch 本身有成本，但更大的性能影响经常是：

> compiler 丢失了 concrete callee information。

## 3. Indirect Call 为什么阻断后续优化

Inlining 通常需要 known callee。

如果 target 不确定：

```text
cannot inline target
```

随后还可能弱化：

- escape analysis；
- constant propagation；
- BCE；
- DCE。

所以 interface cost 可能远大于“一次 indirect call”。

## 4. Static Devirtualization

源码虽然使用 interface，但 compiler 有时仍能证明 concrete receiver。

例如某个 call site 中 dynamic type 已知。

这时可以：

```text
interface call
→ direct call
```

这就是 static devirtualization。

## 5. Devirtualization Enables Inlining

最关键的链：

```text
interface call
   ↓
concrete receiver discovered
   ↓
direct call
   ↓
inline candidate
```

之后 caller-specific fact 会进一步暴露。

## 6. Abstraction 不自动昂贵

一个常见反模式是：

> Interface 慢，所以 hot code 里全部删掉。

这可能无谓破坏 architecture。

如果 compiler 已经 devirtualize，runtime cost 可能非常小。

应该从 measured hot path 出发，而不是从 language feature 标签出发。

## 7. Profile-Guided Devirtualization

Static analysis 有时不能唯一确定 receiver。

Production profile 却可能显示：

```text
95% calls → concrete type A
5%        → others
```

PGO 可以为 hot concrete type 构建 direct fast path，同时保留 generic fallback。

具体 machine code 依赖 compiler version。

## 8. Conditional Fast Path

PGO devirtualization 的价值在于：

- abstraction semantics 仍然保留；
- cold receiver type 仍能工作；
- common production path 被 specialize。

这通常优于手工为了性能到处加 type switch。

## 9. Devirtualization 与 Escape

如果 interface method 接收 temporary pointer，而 call target opaque，escape analysis 可能更 conservative。

Devirtualize + inline 后，compiler 能看到 concrete method 的真实行为。

这可能消除看起来与 interface dispatch 无关的 allocation。

## 10. Devirtualization 与 Generics

Generics 和 interface 解决不同 abstraction problem。

Generic lowering 可能涉及 shape/dictionary。

Interface 可能被 devirtualize。

所以：

```text
generics = zero-cost
interfaces = slow
```

这种简单结论都不可靠。

要看 generated code 与 profile。

## 11. Diagnosing Dynamic Dispatch

可以结合：

- CPU profile；
- compiler diagnostics；
- assembly；
- PGO A/B。

要回答：

> 这个 hot path 的 dynamic dispatch 是否真的还存在？它是否阻止了有价值的后续优化？

## 12. Manual Type Switch

可以手工构建 specialized path：

```go
switch x := r.(type) {
case *bytes.Reader:
    ...
default:
    ...
}
```

但代价是：

- source complexity；
- coupling；
- maintenance。

不要只因为“devirtualization 是优化”就手工实现。

## 13. API Boundary

一个健康架构经常是：

```text
generic abstraction at boundary
        ↓
concrete specialized hot internals
```

这样可以同时保留 maintainability 与 hot-loop optimization。

## 14. Version Sensitivity

Devirtualization capability 会持续演进。

今天 indirect 的 call，未来可能自动 direct。

所以 performance-driven interface refactor 在 Go upgrade 后应该重新验证。

## 15. 常见误解

### "Interface call 一定 allocation"

错误。

### "Interface call 最终一定 indirect"

错误。

### "删 interface 一定更快"

错误。

### "PGO 需要手工拆 abstraction"

不需要，compiler specialization 本来就是 PGO 的重要价值。

## 16. 工程视角

Dynamic dispatch 的真实成本经常是：

```text
lost compiler knowledge
```

而不仅是 dispatch instruction。

Devirtualization 的价值是把这份 knowledge 恢复回来，让 optimization chain 重新连接。
