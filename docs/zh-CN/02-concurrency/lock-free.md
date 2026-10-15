# Lock-Free 算法

[English](../../02-concurrency/lock-free.md) | 简体中文

## 1. Lock-Free 是 Progress Property

“Lock-free”经常被误解成：

> 没有 lock，所以一定更快。

这是错误的。

Lock-free 描述 concurrent algorithm 的 progress guarantee。

它不保证低 latency，也不保证低 CPU。

## 2. Progress Classes

可以粗略理解为：

### Blocking

某个 stalled participant 可能阻止其他人继续。

### Obstruction-Free

一个 operation 如果最终获得无干扰执行时间，就能完成。

### Lock-Free

系统整体保证总有某个 operation 持续完成。

### Wait-Free

每个 operation 都能在 bounded steps 内完成。

这些是 correctness/progress property，不是 benchmark 排名。

## 3. CAS Loop

常见 lock-free pattern：

```go
for {
    old := state.Load()
    next := update(old)

    if state.CompareAndSwap(old, next) {
        return
    }
}
```

成功 CAS 往往就是 linearization point。

在那一刻之前，operation 还没有逻辑 commit。

## 4. Retry Cost

CAS fail 后：

- previous computation 被丢弃；
- state 重新 load；
- update 可能重新计算；
- 再次 RMW 同一 shared line。

在高 contention 下，retry 可以成为主成本。

## 5. Retry Amplification

64 个 worker 更新同一 state。

一个成功，大量失败。

如果 failure 立即 retry，系统可能把多数 CPU 花在无效 attempt 上。

这时一个会 park loser 的 Mutex 反而更高效。

## 6. Backoff

Backoff 的目的不是“sleep for speed”，而是降低 contenders 同时攻击同一 shared location 的频率。

可以包括：

- CPU pause/yield；
- increasing delay；
- adaptive retry。

没有 universally correct constant。

必须结合 hardware 与 workload。

## 7. Linearization Point

一个 concurrent operation 应该看起来像在某个单独 logical instant 生效。

例如 stack push：

```text
prepare node
↓
successful CAS(head)
↓
operation visible
```

如果无法清晰指出 linearization point，算法就很难验证。

## 8. Helping

某些 lock-free algorithm 允许一个 goroutine 帮另一个 goroutine 完成部分操作。

Queue algorithm 中，一个 thread 可能帮忙 advance tail。

这与普通 critical-section reasoning 很不同。

## 9. Scheduler Interaction

Goroutine 可以在 lock-free operation 中途被 preempt。

它恢复时，其他 goroutine 可能已经修改 structure 很多次。

算法仍然必须正确。

Lock-free theory 也不等于 Go scheduler 下拥有良好 P99。

## 10. GC 有帮助，但不解决一切

Go GC 减少了 C/C++ lock-free structure 常见的 manual memory reclamation 负担。

如果 goroutine 持有 normal Go pointer，对象会参与 GC reachability。

但 GC 不会解决：

- ABA；
- linearizability；
- retry；
- starvation；
- broken invariant。

## 11. Pooling 与 Reuse

Object reuse 会影响 lock-free correctness。

被删除的 node 可能稍后以同一 identity/address 再次出现。

这会增强 ABA 风险。

因此给 lock-free structure 加 `sync.Pool` 不能只看 allocation 减少。

## 12. Race-Free 不等于 Correct

Atomic 可以消除 data race，但算法仍可能逻辑错误：

- lost node；
- ABA；
- ordering bug；
- non-linearizable history。

Race detector 必要，但远远不充分。

## 13. Benchmark Lock-Free

至少覆盖：

- low contention；
- moderate contention；
- high contention；
- multicore scaling；
- retry count；
- CPU；
- latency；
- allocation。

如果 throughput +5%，CPU 却 5×，它未必是系统优化。

## 14. 什么时候适合 Lock-Free

更合理的场景是 specialized low-level structure，并且：

- operation 很小；
- blocking 真的有问题；
- contention pattern 已知；
- correctness 可以严格测试；
- expected gain 足够大。

普通 application state 通常不需要 custom lock-free。

## 15. 工程原则

不要因为 lock-free “高级”就使用它。

只有 progress model 真正解决一个被测量的问题，并且简单 ownership/locking 方案无法满足时，才值得进入这一层。
