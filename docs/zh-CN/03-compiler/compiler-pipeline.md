# Go Compiler Pipeline

[English](../../03-compiler/compiler-pipeline.md) | 简体中文

## 1. 为什么 Compiler 会影响性能

Go 源码不会一对一映射成 machine instruction。

从：

```go
result := process(input)
```

到最终 CPU 执行之间，compiler 会进行多阶段分析和转换。

可以用下面这个高层模型理解：

```text
Go source
   ↓
parsing and type checking
   ↓
compiler IR
   ↓
devirtualization / inlining
   ↓
escape analysis
   ↓
SSA construction
   ↓
optimization passes
   ↓
architecture-specific lowering
   ↓
register allocation
   ↓
machine code
```

性能工程之所以要理解 compiler，是因为很多 runtime cost 只有在 compiler **无法证明它们不需要存在** 时才会留下来。

例如：

```text
cannot prove index is safe
→ keep bounds check

cannot prove object lifetime
→ allocate on heap

cannot determine concrete receiver
→ keep indirect interface dispatch

cannot inline
→ lose caller context for later optimization
```

因此 compiler-aware performance engineering 很多时候是在解决：

> 如何让重要事实变得可证明？

## 2. Source Code 是语义 Contract，不是 Machine-Code 描述

Go expression 告诉 compiler 必须保留什么行为。

它通常不会规定机器必须如何实现。

例如：

```go
x := new(Foo)
```

并不等于：

> 必须 heap allocate 一个 Foo。

真正的语义是：

> 返回一个指向零值 Foo 的 pointer，并保证该 value 拥有正确 lifetime。

如果 compiler 能证明 lifetime 仅局部存在，storage 可以留在 stack。

类似：

```go
v := s[i]
```

要求的是：

> 如果 `i` 越界就 panic。

它并不要求：

> 在这个 source location 一定发出一个独立 runtime bounds check。

如果安全性已被证明，check 可以消失。

## 3. Front End

Compiler front end 负责把源码理解成合法 Go 程序，主要包括：

- parsing；
- name resolution；
- type checking；
- generic instantiation/lowering；
- internal representation 构建。

在这里 compiler 获得：

- concrete type；
- interface relationship；
- constant；
- control-flow structure。

这些信息会被后续 optimization pass 使用。

## 4. Intermediate Representation

Compiler 通常不会直接在 source syntax 上完成所有优化。

它会把程序转换成 internal representation，使语义/data flow 更容易分析。

IR 把：

```text
程序意味着什么
```

和：

```text
源码恰好怎么写
```

分离开。

不同源码形式可能在 lowering 后变成等价结构。

## 5. Devirtualization 为后续优化打开路径

Interface call 会隐藏 concrete callee。

例如：

```go
func consume(r io.Reader, b []byte) {
    r.Read(b)
}
```

`r.Read` 是 dynamic dispatch。

如果 compiler 能证明某个 call site 中 `r` 总是某个 concrete type，它可以把 interface dispatch 变成 direct call。

这就是 devirtualization。

它重要的不只是省一次 indirect dispatch，而是：

```text
devirtualization
      ↓
direct call
      ↓
inlining
      ↓
more optimization context
```

之后可能继续改善：

- escape analysis；
- constant propagation；
- BCE；
- DCE。

## 6. Inlining 与 Escape Analysis

Inlining 把 callee body 放进 caller。

例如：

```go
func value(p *Point) int {
    return p.X
}

func work() int {
    p := &Point{X: 10}
    return value(p)
}
```

概念上 inline 后：

```go
func work() int {
    p := &Point{X: 10}
    return p.X
}
```

Compiler 现在能直接看到 `p` 在 caller 中的完整 lifetime。

这可能改变 escape decision。

所以 inlining 绝不只是“去掉 CALL”。

它是 enabling optimization。

## 7. Escape Analysis

Escape analysis 追踪 reference flow，判断 value 能否安全留在 stack-managed lifetime。

核心问题不是：

```text
有没有 &
有没有 new
```

而是：

> 这个 value 的 lifetime 是否可能超过所属 stack frame？

如果 pointer 可能在 frame 结束后继续使用，就需要更长寿命 storage，通常是 heap。

详见：

- [逃逸分析](./escape-analysis.md)

## 8. SSA Construction

高层 transformation 后，compiler 会进入 Static Single Assignment（SSA）形式。

例如：

```go
x := 1

if cond {
    x = 2
}

return x
```

可以概念化成：

```text
x0 = 1

if cond:
    x1 = 2

x2 = φ(x0, x1)

return x2
```

每个 SSA value 只有一个 definition。

这样 compiler 更容易分析：

- value 来源；
- constant；
- dominance；
- dead value；
- redundant check。

详见：

- [Static Single Assignment](./ssa.md)

## 9. Machine-Independent Optimization

在 SSA 中可以进行大量与具体 CPU 无关的 transformation：

- constant propagation；
- dead-code elimination；
- branch simplification；
- nil-check elimination；
- bounds-check elimination；
- value rewriting。

共同主题是：

> 已经能从程序语义证明的工作，就不要在 runtime 再执行一次。

## 10. Dominance 与 Proof

Control flow 本身可以提供 proof。

例如：

```go
if len(b) < 8 {
    return
}

use(b[7])
```

到达 `b[7]` 的所有路径都经过：

```text
len(b) >= 8
```

这个 guard dominates 后续 access。

因此 compiler 可能删除独立 bounds check。

## 11. Architecture-Specific Lowering

SSA operation 仍然是 abstract operation。

最终必须 lower 到：

- amd64；
- arm64；
- riscv64；
- 其他 architecture。

同一个 abstract add / atomic / copy，在不同 CPU 上可能变成不同 instruction sequence。

所以 source-level 性能结论即使 Go semantics 相同，也可能 architecture-sensitive。

## 12. Register Allocation

CPU register 数量有限。

Compiler 需要决定：

- 哪些 value 放 register；
- 哪些 value spill 到 memory。

Large inlined function 或同时 live value 太多，会增加 register pressure。

这也是为什么：

```text
more inlining
```

不等于：

```text
always faster
```

Inlining 需要平衡：

- call removal；
- optimization opportunity；
- code size；
- register pressure；
- instruction-cache impact。

## 13. Machine Code 是最终实现证据

当高层 diagnostic 不够时，machine code 可以回答：

- CALL 是否还在？
- interface dispatch 是否仍 indirect？
- bounds panic path 是否仍存在？
- atomic lower 成什么 instruction？
- branch 是否被 fold？

常用工具：

```bash
go build -gcflags='-S' ./...
```

以及：

```bash
go tool objdump -s 'package\.Function' ./binary
```

Assembly 很强，但通常应该是诊断链后端，而不是第一步。

## 14. Compiler Optimization 是一张图，不是孤立技巧

不要把：

```text
inlining
escape analysis
BCE
devirtualization
```

看成互不相关的技巧。

更真实的关系是：

```text
concrete type discovered
        ↓
devirtualization
        ↓
direct call
        ↓
inlining
        ↓
caller facts exposed
        ↓
escape improves
BCE improves
constant propagation improves
        ↓
dead code disappears
```

这是 compiler performance engineering 最重要的整体认识之一。

## 15. Readable Code 经常也是 Optimizable Code

Compiler-friendly hot code 往往也符合良好工程风格：

- clear control flow；
- explicit guard；
- simple helper；
- concrete invariant；
- limited hidden mutation。

这不意味着整个项目都要“为了 compiler 写代码”。

只有真正 hot 的路径才值得考虑 optimizer visibility。

## 16. Version Sensitivity

下面这些都不是 Go language 永久保证：

- 某函数一定 inline；
- 某 allocation 一定 stack；
- 某 bounds check 一定删除；
- generic 一定采用某种 lowering；
- interface call 一定 devirtualize；
- operation 一定 lower 成某条 instruction。

Compiler heuristic 会变化。

所以 implementation-sensitive 优化必须在目标 Go version 上验证。

## 17. 工程视角

Compiler 不是“把语法翻译成机器码”的黑盒。

它是性能模型中的主动参与者。

正确的问题不是：

> 源码写了多少操作？

而是：

> Compiler 已经证明并删除一切能删除的工作之后，最终还有什么工作真正到达 CPU？
