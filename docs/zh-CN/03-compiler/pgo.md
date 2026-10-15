# Profile-Guided Optimization

[English](../../03-compiler/pgo.md) | 简体中文

## 1. Static Analysis 的边界

Compiler 能看到 program structure，却不知道 production behavior。

它可能知道：

```text
interface 有 5 个 implementation
```

却不知道：

```text
99% real calls 使用 implementation A
```

也可能知道两条 path 都 reachable，却不知道哪条真正 hot。

Profile-Guided Optimization（PGO）把 runtime evidence 加入 compiler decision。

## 2. CPU Profile 提供什么

Representative CPU profile 可以近似告诉 compiler：

- hot function；
- hot call edge；
- common dynamic receiver。

这样 compiler 能把 optimization budget 放在真正重要的位置。

## 3. PGO 不是 JIT

Go PGO 仍然是 ahead-of-time compilation：

```text
production-like workload
      ↓
CPU profile
      ↓
go build with PGO
      ↓
optimized binary
```

Profile 作为 build input 参与编译。

## 4. Hot Inlining

Inlining 有 code-size trade-off。

没有 profile 时，compiler 用 generic heuristic。

PGO 可以告诉 compiler 某个 call site 足够 hot，值得更积极 inline。

随后还可能继续触发：

- escape improvement；
- constant propagation；
- BCE。

## 5. Profile-Guided Devirtualization

如果 interface call 有多个可能 receiver，profile 可能显示一个 dominant type。

Compiler 可以构造：

```text
hot receiver
→ direct path

other receivers
→ generic fallback
```

在保持 correctness 的同时优化 common path。

## 6. Representative Profile

PGO 的质量完全依赖 profile 质量。

Bad input 包括：

- 一个 rare incident；
- 与 production 无关的 synthetic benchmark；
- 只覆盖一个 endpoint；
- debugging workload。

Profile 必须代表 binary 未来主要服务的真实 workload。

## 7. Workload Drift

Production behavior 会变。

旧 profile 可能不再代表：

- endpoint distribution；
- data shape；
- receiver type；
- hot call path。

所以 PGO artifact 需要 lifecycle management。

它是 performance input，不是永久真理。

## 8. Multiple Workload

Service 可能有多个重要模式：

- daytime read traffic；
- write-heavy batch；
- region-specific behavior。

单个 profile 可能 overrepresent 某一模式。

可以根据需要：

- 收集 mixed representative profile；
- 使用 dominant workload profile；
- 合并 profile；
- 只有确有必要时才维护 workload-specific binary。

## 9. PGO 与 Architecture Design

PGO 的一个重要价值是可以保留 clean abstraction。

相比为了性能手工 special-case every interface/helper，compiler 可以根据真实 profile 做 specialization。

这减少了为了性能扭曲 source architecture 的压力。

## 10. Comparing PGO Builds

正确比较应该是：

```text
same source
same toolchain
same environment
PGO off
vs
PGO on
```

用 representative workload 测：

- throughput；
- latency；
- CPU；
- binary size；
- allocation。

## 11. Secondary Metric 也可能变化

More aggressive inlining 可能：

- increase binary size；
- change instruction-cache behavior；
- change escape result；
- change allocation pattern。

所以不能只看一个 ns/op。

## 12. Microbenchmark Profile 的风险

用 microbenchmark profile 做 PGO 可能让 compiler 过拟合非 production behavior。

更合理来源：

- production；
- production-like staging；
- representative integration workload。

## 13. Diagnostic Profile vs PGO Profile

Incident profile 回答：

> 异常时发生了什么？

PGO profile 回答：

> 平时什么最 hot？

这两个问题不同。

不要把 P99 incident profile 直接变成长期 default PGO。

## 14. Profile Freshness

出现这些变化时应重新评估：

- major feature；
- traffic mix；
- large dependency change；
- Go version；
- architecture。

## 15. Build Reproducibility

PGO 新增了 build input。

应该记录：

- source commit；
- Go version；
- profile origin；
- profile date/workload；
- build flags。

未知来源的 stale profile 会让 binary 更难解释。

## 16. PGO 不替代 Profiling

如果系统根因是：

- excessive allocation；
- global lock serialization；
- external IO；
- memory bandwidth；

PGO 不会自动重构 architecture。

它只是在现有 program 内改善 compiler decision。

## 17. Expected Benefit

PGO 效果完全 workload-dependent。

有些程序显著改善，有些很小。

原因可能是：

- hot code 本来就优化很好；
- bottleneck 在 external dependency；
- contention dominates；
- memory bandwidth dominates；
- profile 没有提供新的 useful information。

不要套用一个公开百分比作为 expectation。

## 18. Version Sensitivity

Go release 会继续扩展 PGO use case，例如更好的 devirtualization / inlining。

所以 toolchain upgrade 后应该重新 benchmark。

## 19. 常见误解

### "PGO 只适合超大项目"

错误。

### "PGO 可以替代手工性能分析"

错误。

### "任何 CPU profile 都适合作 PGO"

错误。

### "PGO 一定提升"

错误。

### "PGO 会改变 Go semantics"

不会，它只改变 optimization decision。

## 20. 工程视角

PGO 给 compiler 提供 static analysis 无法完全恢复的信息：

> 这个应用在真实 workload 中到底最常做什么。

它最重要的价值不是某个单独 pass，而是让 compiler effort 与 production reality 对齐。
