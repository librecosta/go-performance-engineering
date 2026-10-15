# Inlining

[English](../../03-compiler/inlining.md) | 简体中文

## 1. Inlining 做了什么

Inlining 把 callee body 展开到 call site。

例如：

```go
func add(a, b int) int {
    return a + b
}

func work() int {
    return add(1, 2)
}
```

概念上：

```go
func work() int {
    return 1 + 2
}
```

最直观的收益是去掉 call overhead。

但这通常不是最重要的收益。

## 2. Inlining 是 Enabling Optimization

Inlining 让 caller context 进入 callee body。

之后 compiler 可能看到：

- constant argument；
- caller-side length guarantee；
- concrete receiver；
- shorter lifetime；
- dead branch。

于是继续触发：

- escape improvement；
- BCE；
- constant propagation；
- DCE。

因此：

> Inlining 的主要价值经常是它打开了后续优化。

## 3. Constant Propagation Example

```go
func enabled(flags uint32) bool {
    return flags&1 != 0
}

func process() {
    const flags = 0

    if enabled(flags) {
        expensive()
    }
}
```

inline 后 `flags&1` 是 known constant。

Compiler 可以把整个 `expensive()` path 删除。

收益远大于“一次 CALL”。

## 4. Inlining 与 Escape Analysis

Helper：

```go
func makePoint() *Point {
    return &Point{X: 1}
}
```

单看 helper，pointer 从 callee 逃出。

但如果 inline 到立即消费的 caller，compiler 可能重新证明 object 只需要 local lifetime。

## 5. Inlining 与 Bounds Check

例如 helper：

```go
func byte7(b []byte) byte {
    return b[7]
}
```

caller：

```go
func parse(b []byte) byte {
    if len(b) < 8 {
        return 0
    }

    return byte7(b)
}
```

inline 后 caller 的 `len(b) >= 8` proof 直接包围 `b[7]`，可能触发 BCE。

## 6. Inlining Heuristic

Inlining 会增加 code size。

Compiler 因此使用 heuristic，而不是无限展开。

考虑因素包括：

- function complexity；
- call-site context；
- profile information；
- expected benefit；
- code growth。

这些 heuristic 会随 Go release 变化。

不要写死：

> 小于 N 个 node 一定 inline。

这是 implementation detail。

## 7. Diagnostics

可以用：

```bash
go build -gcflags='-m=2' ./...
```

观察：

```text
can inline
cannot inline
```

以及原因。

## 8. Inlining 也可能增加 Code Size

重复展开 callee 会增加 binary/code footprint，可能带来：

- instruction-cache pressure；
- binary size；
- register pressure。

所以 more inlining 不是绝对更好。

## 9. Hot Helper

Small helper 可以保持源码结构清晰，同时由 compiler 消除 call overhead。

这说明 abstraction 与 optimization 不必天然冲突。

## 10. Giant Function 不一定更快

Manual flatten function 可能导致：

- larger instruction footprint；
- worse heuristic；
- register pressure；
- poorer maintainability。

不要为了“省 call”把整个 hot path 写成一个巨大函数。

## 11. Interface 与 Inlining

Indirect interface call 无法像 known direct call 那样直接 inline。

Devirtualization 可以先把它变 direct call，再尝试 inline：

```text
interface call
   ↓
devirtualization
   ↓
direct call
   ↓
inlining
```

## 12. PGO 与 Inlining

PGO 告诉 compiler 哪些 call site 真正 hot。

于是 compiler 可以把 optimization budget 更多投入实际高频路径，而不需要手工删除 abstraction。

## 13. Benchmark Interpretation

同一个 function 在 microbenchmark 和 real application 中可能生成不同 machine code，因为 caller context 不同：

- constant；
- inline；
- escape；
- devirtualization。

所以 benchmark 应尽量模拟真实 call pattern。

## 14. Disable Inlining 作为实验

暂时禁用 inlining 可以帮助判断一个结果是否依赖它。

这种 flag 是 diagnostic experiment，不是 production configuration。

## 15. 常见误解

### "Inlining 只是省 CALL"

不完整。

### "所有 tiny function 永远 inline"

不是语言保证。

### "手工 merge function 一定更快"

错误。

### "Interface 永远阻止优化"

不一定，devirtualization/PGO 可能解决。

## 16. 工程视角

Inlining 最好理解成 **context propagation**。

它让 caller 已知事实流入 callee body，从而可能继续删除 allocation、check、branch 和 dispatch。

所以正确问题不是：

> 要不要手工去掉这个函数调用？

而是：

> Compiler 是否已经拥有足够上下文来优化这个 hot call path？
