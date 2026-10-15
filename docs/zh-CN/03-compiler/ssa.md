# Static Single Assignment

[English](../../03-compiler/ssa.md) | 简体中文

## 1. SSA 是什么

SSA = Static Single Assignment。

在 SSA form 中，每个 logical value 只有一个 assignment/definition。

这不是要求 Go 源码这样写，而是 compiler internal representation。

例如：

```go
x := 1

if cond {
    x = 2
}

return x + 1
```

概念上变成：

```text
x0 = 1

if cond:
    x1 = 2

x2 = φ(x0, x1)

return x2 + 1
```

`φ` 表示从不同 control-flow path 合并 value。

## 2. SSA 为什么有利于优化

普通源码变量可以多次赋值。

Compiler 需要回答：

- 哪次 assignment 到达当前 use？
- 某 branch 是否覆盖它？
- value 是否 constant？
- value 是否仍 live？

SSA 中 use 直接指向一个具体 definition，使这些关系显式化。

这有利于：

- constant propagation；
- DCE；
- redundancy removal；
- BCE；
- branch simplification。

## 3. Values 与 Blocks

SSA program 可以理解成：

```text
basic blocks
+
SSA values
+
control-flow edges
```

一个 basic block 内的 operation 顺序执行，直到 control 转向其他 block。

例如：

```text
entry
  ↓
check condition
 ↙        ↘
then      else
  ↘        ↙
    merge
```

Compiler 可以分析 block dominance 与 value availability。

## 4. Dominance

如果到达 block B 的每条路径都必须经过 A，那么 A dominates B。

这是很多 optimization 的基础。

例如：

```go
if len(b) < 8 {
    return
}

x := b[7]
```

successful length check dominates `b[7]`。

所以 compiler 知道：

```text
every path reaching b[7]
has len(b) >= 8
```

这就是 BCE 所需的 proof。

## 5. Phi Value

例如：

```go
x := 10

if cond {
    x = 20
}

use(x)
```

merge point 的 `x` 可能来自两个 definition。

SSA 会构造：

```text
x2 = φ(x0, x1)
```

然后继续分析 possible value。

## 6. Constant Propagation

Inlining 之后可能出现：

```go
const flags = 0

if flags&1 != 0 {
    expensive()
}
```

SSA 能证明：

```text
flags&1 == 0
```

于是 branch 恒 false。

Compiler 可以先删除 branch，再通过 DCE 删除 `expensive()`。

这说明 optimization pass 经常是链式发生。

## 7. Dead-Code Elimination

如果某 SSA value 没有 observable effect，也没有需要的 use，它可以消失。

因此错误 microbenchmark 可能最终 benchmark 了“什么都没有”。

Benchmark 必须确保被测 operation 对程序仍然具有语义相关性。

## 8. Branch Simplification

当 condition 可知时：

```text
if true
```

不需要 runtime branch。

Condition 可能因为：

- constant；
- prior comparison；
- type information；
- inlining；

而变得可知。

## 9. Bounds Check 也是 SSA Operation

Bounds check 会成为 compiler 可分析的 SSA operation。

如果 dominating condition 已经证明 index 安全，redundant check 可以消除。

因此 BCE 不只是局部 peephole optimization，它依赖 control flow。

## 10. Nil-Check Elimination

同样，如果 prior operation 已经证明 pointer 非 nil，后续 redundant nil check 也可能被删除。

源码存在 dereference 并不意味着独立 runtime nil check 一定留下。

## 11. SSA 与 Architecture Lowering

Machine-independent SSA operation 最终会 lower 成 architecture-specific operation。

Atomic、add、copy 等都可能在不同 architecture 上生成不同 instruction。

SSA 是 Go semantics 与 machine code 之间的重要桥梁。

## 12. 查看 SSA

可以用：

```bash
GOSSAFUNC=FunctionName go build ./...
```

生成 SSA HTML。

它适合分析：

- 为什么 bounds check 没删？
- 为什么 branch 没简化？
- value 为什么仍 live？
- abstract operation 如何 lower？

## 13. SSA 不是第一层 Profiling Tool

SSA 回答：

> Compiler 做了什么？

它不回答：

> 这个函数是否值得优化？

健康流程应该是：

```text
profile / benchmark
      ↓
identify hot function
      ↓
compiler diagnostics
      ↓
SSA if needed
```

对 cold code 随意打开 SSA 通常没有价值。

## 14. Source Structure 与 SSA Quality

能清楚暴露 fact 的源码通常产生更简单 SSA，例如：

- explicit length guard；
- simple control flow；
- concrete hot-path type；
- fewer unnecessary interface boundaries。

但 source clarity 仍然优先。

不要为 speculative optimizer behavior 扭曲普通 application code。

## 15. SSA 与 Benchmark DCE

如果 benchmark 调用 pure function 却完全不使用结果，inlining + SSA 可能发现 operation 没有 observable effect 并删除它。

现代 benchmark API 降低了一部分风险，但原则不变：

> 被测 operation 必须在语义上仍然存在。

## 16. 工程视角

SSA 暴露了 compiler optimization 的核心：

> 很多优化本质上是 data-flow proof。

如果 compiler 能证明：

- value 是 constant；
- branch 不可能发生；
- index 安全；
- result 无用；

runtime work 就可以消失。
