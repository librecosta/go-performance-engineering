# 逃逸分析

[English](../../03-compiler/escape-analysis.md) | 简体中文

## 1. Stack vs Heap 通常是 Compiler Decision

例如：

```go
p := &Point{X: 1, Y: 2}
```

源码取了 address，不等于一定 heap allocate。

真正的问题是：

> Compiler 能否证明这个 object 的 lifetime 不会超过所属 stack lifetime？

Escape analysis 就是回答这个问题。

## 2. 为什么需要 Escape Analysis

Stack allocation 的 lifetime 很简单：

```text
function starts
   ↓
stack frame exists
   ↓
local objects live
   ↓
function returns
   ↓
frame no longer needed
```

Heap object 则需要通用 lifetime management，并参与 GC reachability。

所以把合适 value 留在 stack 可以减少：

- heap allocation；
- GC object count；
- scan work；
- allocator metadata work。

## 3. 核心安全约束

指向 stack storage 的 pointer 不能在 stack storage 生命周期结束后继续可用。

如果 pointer 可以 outlive 当前 frame，storage 就需要更长 lifetime。

## 4. 返回 Pointer

例如：

```go
func newPoint() *Point {
    p := Point{X: 1}
    return &p
}
```

返回后的 pointer 仍然可用。

如果 `newPoint` 是独立 call boundary，`p` 不能简单跟随 local frame 消失。

Heap placement 是自然结果。

## 5. Inlining 可以改变 Escape Result

例如：

```go
func newPoint() *Point {
    return &Point{X: 1}
}

func usePoint() int {
    p := newPoint()
    return p.X
}
```

如果 inline：

```go
func usePoint() int {
    p := &Point{X: 1}
    return p.X
}
```

Compiler 看到完整 caller context 后，可能证明 object 不再需要 escape。

这说明 escape 与 inline 紧密相关。

## 6. Address-Taking 不等于 Escape

```go
func work() int {
    p := Point{X: 1}
    inspect(&p)
    return p.X
}
```

如果 `inspect` 不保存 pointer，`p` 仍可能留在 stack。

所以：

```text
address taken
```

和：

```text
heap allocated
```

完全不是同一件事。

## 7. Pointer Receiver 不等于 Heap

```go
func (p *Point) XValue() int {
    return p.X
}
```

local value 可以安全使用 pointer receiver，只要 receiver pointer 没有超过 local lifetime。

## 8. Parameter Leakage

Parameter 可以通过：

- return value；
- heap storage；
- closure；
- global state；

泄漏到更长 lifetime。

例如：

```go
func identity(p *Point) *Point {
    return p
}
```

函数本身不 new object，但它声明了 incoming pointer 可以从 return path escape。

Compiler diagnostic 可能把这描述为 parameter leakage。

## 9. Closure

Closure 可以延长 captured variable lifetime。

例如：

```go
func counter() func() int {
    n := 0

    return func() int {
        n++
        return n
    }
}
```

返回 closure 后仍要访问 `n`，所以 state 必须活得更久。

## 10. Local Closure 可能不同

```go
func work() int {
    n := 0

    inc := func() {
        n++
    }

    inc()
    return n
}
```

如果 closure 本身不 escape，compiler 可能保持 local lifetime。

还是那句话：源码语法不直接决定 placement。

## 11. Interface 与 Escape

Interface conversion 可能影响 lifetime。

例如：

```go
func box(v Value) any {
    return v
}
```

如果 interface value escape，concrete representation 也可能需要更长 storage。

结果取决于：

- value size；
- Go version；
- inlining；
- use site。

不能只看到 interface 就断言 allocation。

## 12. Dynamic Call 会减少信息

如果 compiler 无法解析 callee，就需要更 conservative assumption。

Devirtualization 与 inlining 能恢复更多信息，从而改善 escape analysis。

## 13. Large Object

即使 lifetime analysis 表示“does not escape”，非常大的 object 也可能因为 stack/frame constraint 不适合 stack placement。

所以：

```text
does not escape
```

不等于：

```text
guaranteed stack allocation
```

它只说明 lifetime 本身不要求 heap。

## 14. Diagnosing Escape

常用：

```bash
go build -gcflags='-m=2' ./...
```

关注：

```text
moved to heap
does not escape
leaking param
can inline
cannot inline
```

更详细输出可以帮助追踪 pointer flow。

目标不是机械统计 “moved to heap”，而是理解 hot allocation **为什么** escape。

## 15. Benchmark Evidence

Compiler diagnostic 解释 decision。

Benchmark 解释这个 decision 是否重要。

常见指标：

```text
allocs/op
B/op
ns/op
```

Cold path 少一个 tiny allocation，可能没有系统价值。

## 16. API Design 会影响 Escape

例如：

```go
func Encode(v Value) []byte
```

与：

```go
func AppendEncode(dst []byte, v Value) []byte
```

第二种让 caller 控制 storage reuse，可能减少 hot-path allocation。

它不一定总是更好，但 lifetime ownership 更显式。

## 17. Pool 不是第一反应

如果 hot object 是因为 lifetime/API shape 导致不必要 escape，优先解决：

> 能否让它根本不需要 heap？

Pool 仍然是在 heap/GC-managed storage 中复用 object。

完全移除 heap object 通常更强。

## 18. Unsafe / noescape Contract

Low-level assembly/FFI 可以向 compiler 显式声明 noescape contract。

如果声明错误，就可能把本应 heap 的 object 放 stack，最终出现 dangling pointer。

这属于 correctness assertion，不是 tuning hint。

## 19. 常见误解

### "`new` 一定 heap allocation"

错误。

### "`&` 一定 heap allocation"

错误。

### "Pointer receiver 一定 heap"

错误。

### "`does not escape` 一定 stack"

错误。

### "一个 allocation 永远是问题"

错误。

正确问题是：

> 这个 allocation 是否处于 measured hot path？它又造成了什么 downstream cost？

## 20. 工程视角

Escape analysis 把 source-level lifetime 与 runtime memory management 连接起来。

去掉一个 heap allocation 的价值不仅是省 allocator CPU。

它还可能把一个 object 从整个 heap/GC lifecycle 中移除。
