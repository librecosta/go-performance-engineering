# ABA 问题

[English](../../02-concurrency/aba.md) | 简体中文

## 1. 什么是 ABA

ABA 表示一个 value 经历：

```text
A
↓
B
↓
A
```

CAS 只看到 current value 又是 `A`。

它不知道中间已经发生过变化。

## 2. Stack Example

假设：

```text
head
 ↓
A → B → C
```

G1 读取：

```text
old = A
next = B
```

在 G1 CAS 前，G2：

```text
pop A
pop B
push A
```

现在：

```text
head
 ↓
A → C
```

G1 恢复并判断：

```text
head == A
```

CAS 可能成功，但 structure 已经不是 G1 最初观察到的 state。

## 3. ABA 关于 Identity 与 History

ABA 经常被说成 allocator reuse：

```text
object freed
same address reused
```

但 ABA 不需要 allocator。

同一个 object 也可以：

```text
removed
reinserted
```

产生 logical ABA。

所以：

> GC 不会自动消除 ABA。

## 4. 为什么 CAS 看不到 History

CAS 只比较 current machine value 与 expected value。

它不编码：

- version；
- mutation history；
- generation。

如果 machine representation 一样，CAS 就认为“没有变化”。

## 5. Tagged / Versioned State

常见防御方式是存：

```text
(value, version)
```

而不是只存：

```text
value
```

例如原本：

```text
(A, 17)
```

删除再插回后：

```text
(A, 18)
```

旧 CAS 期待 `(A,17)` 就会失败。

## 6. Tagged Pointer

Low-level algorithm 可能把：

- pointer/address；
- version/tag；

压入一个 atomic word。

这样 CAS 对 reuse 更敏感。

但具体技术依赖：

- address width；
- alignment；
- architecture；
- runtime assumption。

属于高度 implementation-sensitive 代码。

## 7. Go Runtime 作为 Study Case

Go runtime 内部存在 tagged/versioned lock-free state。

这些代码很适合学习 algorithm design。

但不能直接复制到 application，因为 runtime 可能依赖：

- private invariant；
- non-GC-visible representation；
- architecture-specific assumption。

## 8. GC 与 ABA

Go GC 帮助 memory lifetime：

只要 goroutine 仍持有 normal Go pointer，对象就保持 reachable。

这降低 use-after-free 风险。

但如果相同 logical identity 又回到 structure，ABA 仍然存在。

## 9. Pooling

Pool/freelist 增加 object reuse。

因此 lock-free structure 引入 `sync.Pool` 时必须重新检查 ABA assumption。

## 10. Testing ABA

ABA bug 很难通过普通 unit test 命中。

更有用的策略：

- orchestrated interleaving；
- stress test；
- model-based test；
- invariant check。

Race detector 不能证明 ABA safety。

## 11. 工程原则

只要 CAS correctness 建立在：

> “value 没变，所以 state 没变。”

就应该继续问：

> value 能否离开 A，然后再回到 A？

如果可以，就必须显式考虑 ABA。
