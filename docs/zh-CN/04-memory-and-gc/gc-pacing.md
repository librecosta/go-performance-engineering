# GC Pacing

[English](../../04-memory-and-gc/gc-pacing.md) | 简体中文

## 1. 为什么需要 Pacer

Tracing GC 不能等 memory 快耗尽才开始。

它必须持续决定：

- 什么时候 start；
- 用多少 CPU；
- 如何跟上 allocation；
- 允许 heap 增长多少。

这就是 GC pacing。

## 2. Heap-Growth Trade-Off

核心交换：

```text
more heap headroom
→ fewer GC cycles
→ more memory

less heap headroom
→ more GC cycles
→ less memory
```

Go 主要通过：

- GOGC；
- GOMEMLIMIT / `debug.SetMemoryLimit`；

表达这个 trade-off。

## 3. GOGC

一个有用的现代近似模型：

```text
Target heap =
Live heap +
(Live heap + GC roots) × GOGC / 100
```

真实 runtime pacer 更复杂，但这个公式足以解释大方向。

## 4. Example

假设：

```text
live heap = 2 GiB
roots ≈ 0.2 GiB
GOGC = 100
```

额外 headroom 约：

```text
2.2 GiB
```

target 大约：

```text
4.2 GiB
```

## 5. Higher GOGC

例如：

```text
GOGC 100 → 200
```

允许更多 heap growth。

可能：

- lower GC frequency；
- lower GC CPU；
- higher heap/RSS。

这是明确的 memory-for-CPU trade。

## 6. Lower GOGC

Lower GOGC：

- reduce heap footprint；
- increase GC frequency；
- increase GC CPU；
- increase assist pressure。

Allocation-heavy workload 中过低 GOGC 可能明显伤害 throughput/P99。

## 7. Allocation Rate 与 Cycle Frequency

如果有 1 GiB headroom，allocation 500 MiB/s，粗略两秒左右就会耗尽。

如果变成 2 GiB/s，cycle 会更频繁。

因此 live heap 不变，GC frequency 仍可大幅变化。

## 8. Concurrent Marking

Pacer 把 mark work 分布在 application execution 期间，通过 background worker 持续追踪。

目标是在 heap growth 超过 target 前完成 collection。

## 9. GC Assist Ratio

Application allocation 如果快于 background marking，goroutine 会积累 GC debt，并需要承担 mark assist。

```text
higher allocation
→ more assist
→ more mutator time spent marking
```

## 10. Assist 与 Latency

Heavy allocating request 可能自己做 GC work，从而出现 latency spike。

所以 low STW 不排除 GC-related P99。

## 11. GOMEMLIMIT

GOMEMLIMIT 给 Go runtime-managed memory 一个 absolute soft budget。

当普通 GOGC target 会逼近/超过 limit 时，memory limit 会改变 pacing，使 collector 更积极。

## 12. GOGC 与 GOMEMLIMIT

两者解决不同问题：

### GOGC

正常情况下控制 proportional growth。

### GOMEMLIMIT

提供 operational upper budget。

健康配置经常同时使用：

```text
GOGC
→ normal CPU/memory trade

GOMEMLIMIT
→ soft upper bound
```

## 13. `GOGC=off` + Limit

可以关闭普通 GOGC-driven cycle，同时保留 memory limit。

这让 memory-limit pressure 成为主要 trigger。

当固定 memory budget 很充足时，可能减少不必要 GC。

但行为变化较大，必须真实 load test。

## 14. Memory-Limit Thrashing

如果 true live set 非常接近 limit：

```text
GC runs
↓
little reclaimed
↓
allocation resumes
↓
limit pressure
↓
GC again
```

就会 thrash。

解决方案：

- reduce live set；
- increase limit；
- change representation。

不是“更频繁 GC”。

## 15. Soft Limit

Go memory limit 是 soft，而不是绝对 hard process kill boundary。

当 target 本身不可满足时，runtime 会优先让程序继续 progress，而不是把 100% CPU 全部耗在无意义 collection。

## 16. Root Cost

Modern GOGC model 也考虑 root scanning。

Large goroutine stack/global root set 会影响 pacing。

所以 live heap bytes 不是唯一 GC work predictor。

## 17. Bursty Allocation

突然 batch allocation 几 GiB 可能让 pacer 面临强烈 transient pressure，并触发 assist/aggressive collection。

Average metrics 会隐藏 burst。

## 18. CPU Availability

GC worker 与 application goroutine 争用 CPU。

CPU-bound service 中 GC CPU 直接挤占 request throughput。

Idle service 中同样 GC work 可能不明显。

## 19. What to Measure

一起观察：

```text
GC cycle frequency
GC CPU
mark assist CPU
heap live
heap goal
RSS
allocation bytes/sec
P99 latency
```

只根据 GC count 调 GOGC 不够。

## 20. 官方资料

- GC guide: https://go.dev/doc/gc-guide
- Runtime pacer: https://go.dev/src/runtime/mgcpacer.go
- `runtime/debug`: https://pkg.go.dev/runtime/debug

## 21. 工程视角

GC pacing 是 feedback-control problem。

Application 提供：

```text
allocation rate
live set
roots
CPU availability
```

Runtime 根据这些输入调整：

```text
heap target
background marking
assist work
```

只有先理解输入，tuning 才有意义。
