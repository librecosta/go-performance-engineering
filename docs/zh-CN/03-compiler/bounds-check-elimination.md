# Bounds Check Elimination

[English](../../03-compiler/bounds-check-elimination.md) | 简体中文

## 1. 为什么存在 Bounds Check

Go 必须保证 indexed access 安全。

对于：

```go
v := s[i]
```

如果：

```text
i < 0
or
i >= len(s)
```

程序必须 panic。

如果 compiler 能证明 index 一定合法，就不需要再发出 redundant runtime check。

删除这些 check 就是 Bounds Check Elimination（BCE）。

## 2. BCE 是 Proof Problem

Compiler 必须证明：

```text
0 <= i < len(s)
```

在所有到达 access 的 path 上都成立。

Proof 不完整，就必须保留 check。

所以 BCE 强烈依赖 control flow 与 dominance。

## 3. Range Loop

例如：

```go
for i := range s {
    sum += s[i]
}
```

`range` semantics 本身已经保证 `i` 是合法 index。

Compiler 因而有很强的 proof information。

## 4. Explicit Length Guard

例如：

```go
func decode(b []byte) uint32 {
    if len(b) < 4 {
        return 0
    }

    return uint32(b[0]) |
        uint32(b[1])<<8 |
        uint32(b[2])<<16 |
        uint32(b[3])<<24
}
```

成功 path 上：

```text
len(b) >= 4
```

所以所有固定 index read 都可证明安全。

这是既 readable 又 optimizer-friendly 的 pattern。

## 5. Dominating Maximum-Index Check

Fixed-width parser 常见：

```go
_ = b[7]

x0 := b[0]
x1 := b[2]
x2 := b[7]
```

第一行确认：

```text
len(b) >= 8
```

如果它 dominates 后续 access，repeated check 可以被删除。

这是 non-obvious optimization，应该写 comment 保存理由。

## 6. `_ = b[N]` 没有绕过安全

如果 `b[N]` 本身越界，它依然按照普通 Go semantics panic。

这个 pattern 只是集中/复用 proof。

它与 unsafe pointer arithmetic 完全不同。

## 7. Loop Shape

BCE 会受 loop structure 影响。

简单：

```go
for i := 0; i < len(s); i++ {
    use(s[i])
}
```

index relation 非常直接。

复杂的 index mutation/arithmetic 更难证明。

## 8. Slice Operation

` s[a:b] ` 也需要 bounds validation。

Compiler 同样可以删除 redundant slice-bound checks。

## 9. Inlining 可以 Enable BCE

Caller 中：

```go
if len(b) < 8 {
    return
}
```

Helper 中：

```go
func read7(b []byte) byte {
    return b[7]
}
```

inline 后 caller 的 length proof 可以应用到 helper access。

因此：

```text
inlining
→ BCE
```

## 10. Diagnosing BCE

常用：

```bash
go build -gcflags='-d=ssa/check_bce/debug=1' ./...
```

非常重要：

```text
Found IsInBounds
```

表示这里仍然存在 bounds-check operation。

它不是“BCE 成功”的提示。

## 11. Assembly Evidence

如果 diagnostic 还不够，可以检查 assembly：

```bash
go build -gcflags='-S' ./...
```

或：

```bash
go tool objdump ...
```

看 index-check / panic path 是否仍存在。

## 12. BCE 与 Parser

BCE 对这些 workload 尤其相关：

- binary protocol decoder；
- image codec；
- serialization；
- packet parser；
- compression。

它们可能每次 operation 做大量固定 offset read。

## 13. 普通 Application Code 不需要追 BCE Trick

普通业务逻辑中，readable length guard 往往已经足够。

不要为了 cold path 添加 obscure proof trick。

Maintenance cost 可能远高于 runtime saving。

## 14. Unsafe 不是第一解决方案

如果 hot path 还有 bounds check，优先升级：

```text
clear safe control flow
   ↓
compiler diagnostics
   ↓
BCE-friendly source structure
   ↓
verify again
```

只有 specialized low-level code 在 safe form 无法达标时才考虑 unsafe。

## 15. Version Sensitivity

Compiler BCE 能力会持续改善。

某个 workaround 在新 Go version 中可能不再必要。

Toolchain upgrade 后应重新验证，能删就删。

## 16. 与 Proof Repository 的关系

一个 BCE proof 可以包含：

- baseline source；
- proof-shaped source；
- compiler diagnostics；
- assembly；
- benchmark。

目标是：

> 证明 compiler 能删除 redundant check。

不是：

> 宣称存在 universal speedup percentage。

## 17. 常见误解

### "每次 index access 都有 runtime check"

错误。

### "Compiler 永远会删 loop bounds check"

错误。

### "`Found IsInBounds` 表示已经消除"

错误。

### "Fast parser 必须 unsafe"

错误。

## 18. 工程视角

BCE 展示了 compiler optimization 的核心哲学：

> 当 compiler 已经能从程序结构证明 safety 时，redundant runtime safety check 就可以消失。

优化目标不是减少安全保证。

而是减少重复的 runtime work。
