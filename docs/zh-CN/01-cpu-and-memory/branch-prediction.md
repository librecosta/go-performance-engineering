# 分支预测

[English](../../01-cpu-and-memory/branch-prediction.md) | 简体中文

## 1. 为什么需要 Branch Prediction

现代 processor 使用深 pipeline 与 speculative execution。

遇到：

```go
if cond {
    pathA()
} else {
    pathB()
}
```

CPU 不希望一直等到 `cond` 完全求值后才继续 fetch instruction。

它会预测下一条路径。

## 2. Correct Prediction

如果预测正确，pipeline 可以继续推进，branch 成本可能很小。

类似：

```text
false false false false false
```

或：

```text
true true true true true
```

这种 pattern 很容易学习。

## 3. Misprediction

预测错误时，wrong path 的 speculative work 必须被丢弃，然后从正确路径恢复。

具体 penalty 随 architecture 变化，但稳定结论是：

> 不可预测 branch 往往比高度可预测 branch 昂贵得多。

## 4. Data Distribution 是 Performance Input

例如：

```go
for _, x := range values {
    if x >= threshold {
        sum += int(x)
    }
}
```

如果 data 已排序：

```text
false false ... true true
```

branch 可能很容易预测。

如果数据接近随机 50/50：

```text
T F T F F T ...
```

预测难度明显更高。

同一段代码可以因为输入分布而出现不同性能。

## 5. Benchmark 的影响

Synthetic benchmark 经常不小心制造非常可预测的 branch：

- 永远同一个 HTTP method；
- 永远 cache hit；
- 永远同一种 parser shape；
- switch 永远进入第一个 case。

Production traffic 可能完全不同。

不考虑分布的 benchmark 会产生误导。

## 6. Branchless Programming

有些 branch 可以改成 arithmetic、mask 或 lookup table。

但：

```text
branchless
≠
automatically faster
```

如果原 branch 很可预测，branchless 版本可能：

- 增加 instruction；
- 增加 dependency；
- 增加 memory access；
- 降低 readability。

必须测量。

## 7. Conditional Move 与 Compiler Decision

Compiler 可以在认为合适时把简单 condition 转换成 branchless machine instruction。

不要为了“强制 branchless”就先重写源码。

应该先检查 generated code 与 benchmark。

## 8. Hot / Cold Path

常见结构是：

```text
common fast path
rare slow path
```

例如：

- valid input vs error formatting；
- cache hit vs fallback；
- normal packet vs debug logging。

把 rare cold logic 隔离，有时可以减少 hot instruction footprint。

但具体 code layout / inline behavior 依赖 compiler，需要验证。

## 9. State Machine

protocol parser/state machine 中可能存在大量 branch。

性能取决于：

- state distribution；
- transition predictability；
- data representation；
- code layout。

把 state machine flatten 或 table-driven 不一定总是更快。

## 10. PMU Evidence

如果怀疑 branch behavior，但 source/profile 不够，可以使用 hardware counter：

```text
branches
branch-misses
```

这些 counter 应用于验证 hypothesis，而不是替代 hypothesis。

高 branch-miss rate 只有在知道 hotness、input distribution 与 candidate change 时才有工程意义。

## 11. 工程原则

不要问：

> 怎么把 branch 全部去掉？

应该问：

> branch misprediction 是否真的是当前 workload 的显著成本？能否通过 control/data representation 让 hot path 更可预测，而不引入过高复杂度？
