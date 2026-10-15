# Evidence Model

[English](../../06-methodology/evidence-model.md) | 简体中文

## 1. 为什么需要 Evidence Model

性能结论的证据强度差异很大。

例如：

```text
"这段代码看起来 allocation 很多。"
```

与：

```text
"Production CPU profile 表明该路径占 23% CPU；
compiler diagnostics 显示每个 request 有一个 heap escape；
representative A/B 删除 allocation 后 CPU/request 下降 8%。"
```

显然不应该拥有相同决策权重。

Evidence model 的目的，就是让：

> 证据强度与优化风险、复杂度、适用范围成比例。

## 2. Claim 必须有 Scope

差：

> This is faster.

更好：

> 在 Go 1.27/amd64、4 KiB payload 下，B 将 allocation 从 3/op 降到 1/op，并使该 microbenchmark median 下降约 12%。

如果要支持 production decision，还应该继续：

> Component workload 中 CPU/request 降低 4%，P99 与 RSS 没有 regression。

Scope 防止 local fact 被传播成 universal myth。

## 3. Evidence Ladder

本项目使用：

```text
L0 — Source Inspection
L1 — Diagnostics / Profiles
L2 — Reproducible Microbenchmark
L3 — Component / Service Benchmark
L4 — Production / Canary Evidence
L5 — Hardware PMU / Low-Level Evidence
```

这些 level 不是简单“数字越大越好”。

不同层回答不同问题，成熟调查经常组合使用。

## 4. L0 — Source Inspection

L0 从源码提出 hypothesis，例如：

- repeated allocation；
- one global mutex；
- small view retaining huge buffer；
- interface call in hot-looking loop；
- pointer-heavy `[]*T`。

L0 可以回答：

> 什么东西可能昂贵？

不能回答：

> 什么东西真的昂贵？

## 5. L0 的适用范围

适合：

- initial hypothesis；
- review known hot code；
- 决定下一步工具；
- 找可能的 correctness/performance smell。

单纯 performance-only 的复杂优化，L0 不能作为 merge 依据。

## 6. L1 — Diagnostic / Profile

L1 直接观察 compiler/runtime/system behavior，例如：

- CPU pprof；
- heap/allocs；
- mutex/block；
- trace；
- escape diagnostics；
- BCE diagnostics；
- SSA/assembly；
- runtime metrics。

它回答：

> 怀疑的机制是否真的存在？

## 7. L1 Example

### Escape

```text
compiler: moved to heap
```

确认 escape decision。

### BCE

Residual diagnostic 表示 bounds check 仍存在。

### CPU

pprof 确认 function 是显著 CPU consumer。

### Contention

mutex profile 确认 critical section 真的让其他 goroutine wait。

## 8. L1 的边界

L1 通常能证明 mechanism，不一定能证明 benefit。

例如：

```text
one bounds check remains
```

但它可能每小时执行一次。

所以通常还需要 L2/L3。

## 9. L2 — Reproducible Microbenchmark

L2 在隔离环境下比较 candidate。

适合：

- mechanism validation；
- local cost；
- allocation；
- synchronization primitive scaling。

一个好的 L2 应包含：

- code；
- workload；
- command；
- environment；
- repeated result。

## 10. L2 的强项

它擅长 causal isolation。

例如同一 parser，只改变 BCE-friendly source shape，再用 compiler diagnostic + benchmark，就能很强地支持 local mechanism claim。

## 11. L2 的弱点

Microbenchmark 不能证明 system importance。

Local +30% 可能只带来 service +0.02%。

因此 L2 不能自动授权 high-complexity change。

## 12. L3 — Component / Service Benchmark

L3 把修改放回更 realistic subsystem/workload。

例如：

- HTTP handler；
- storage engine；
- parser + allocation + downstream processing；
- concurrency load test。

它回答：

> Local mechanism 在真实上下文中是否仍有价值？

## 13. L3 的 Representative Input

应尽量保留：

- payload distribution；
- key distribution；
- concurrency；
- cache state；
- object lifetime；
- relevant dependencies。

并非所有 external dependency 都必须包含，但 cost structure 应尽可能真实。

## 14. L4 — Production / Canary

L4 在真实 workload 下看：

- CPU/request；
- host CPU；
- RSS；
- throughput；
- P50/P99；
- GC CPU；
- error rate；
- cost。

这是 user/system value 最强的一类证据。

## 15. Production Evidence 也有 Noise

Production 受：

- traffic；
- deployment timing；
- dependency variance；
- region mix；
- background job；

影响。

应尽量使用：

- canary；
- matched cohort；
- normalized metrics；
- controlled rollout。

一张 dashboard screenshot 不足以建立因果。

## 16. L5 — Hardware PMU

L5 使用：

- cycles；
- instructions；
- IPC；
- cache miss；
- branch miss；
- bandwidth；
- NUMA event。

适合 hardware-specific claim：

- false sharing；
- locality；
- branch predictor；
- bandwidth saturation。

## 17. L5 不是所有优化都需要

如果：

```text
profile
+
benchmark
+
system validation
```

已经回答问题，就没有必要为了“证据等级高”再强行上 PMU。

Evidence cost 也应该与 uncertainty 成比例。

## 18. Evidence by Risk

简单、安全、可逆的优化，需要的证据较少。

例如 measured hotspot 上的 idiomatic preallocation。

高风险技术：

- unsafe zero-copy；
- custom lock-free；
- assembly；
- private runtime；

必须提高 evidence threshold。

## 19. Risk-Adjusted Evidence

可以概念化为：

```text
required evidence
↑
as
correctness risk
maintenance cost
portability risk
scope
↑
```

这是本项目的核心治理规则之一。

## 20. Mechanism Evidence vs Outcome Evidence

应该区分：

### Mechanism

> 预期低层行为真的发生了吗？

例如：

- allocation disappeared；
- bounds check removed；
- cache miss decreased；
- lock wait decreased。

### Outcome

> 系统真的改善了吗？

例如：

- CPU/request ↓；
- P99 ↓；
- throughput ↑；
- RSS within guardrail。

强 optimization 往往两者都有。

## 21. Negative Evidence

证据也可以推翻 optimization hypothesis。

例如：

```text
source:
global atomic looks suspicious

scaling benchmark:
no meaningful contention

profile:
atomic <0.2% CPU
```

结论：

> Do not optimize this path.

这仍然是成功的 performance investigation。

## 22. Absence of Evidence

没找到 hotspot 不等于没有问题。

也可能是工具选错。

例如：

```text
CPU profile normal
P99 terrible
```

下一步应该看 trace/block，而不是宣布“没有性能问题”。

## 23. Representative Evidence

任何 benchmark 只对它代表的 workload 有效。

结果至少记录：

- payload；
- concurrency；
- distribution；
- Go version；
- architecture。

否则 scope 模糊。

## 24. Version Evidence

Compiler/runtime claim 应记录 Go version：

- escape；
- BCE；
- inline；
- allocator；
- GC。

这些都可能随 release 变化。

## 25. Hardware Evidence

涉及：

- cache line；
- branch predictor；
- atomic scaling；
- instruction lowering；

应记录 CPU/architecture。

一个 x86 result 不能直接升级成“Go 一般规律”。

## 26. Statistical Evidence

Repeated runs 能降低 noise。

A/B 要有足够 sample 描述 variance。

Statistical significance 很有用，但不是最终工程 gate。

## 27. Engineering Significance

一个粗略 value model：

```text
measured gain
×
hotness
×
deployment scale
```

应该足以覆盖：

```text
complexity
risk
maintenance
```

## 28. Confidence Language

语言要匹配证据。

### 只有 L0

> 这个 pattern 可能造成 heap allocation。

### L1

> 在当前 Go 1.27 build 中，compiler 将该 value 移到 heap。

### L2

> 在这个 benchmark 下，修改减少 1 alloc/op，并降低 median runtime。

### L4

> Comparable traffic canary 中 CPU/request 降低，P99 无 regression。

精准措辞能防止 overclaim。

## 29. Evidence Provenance

重要结论应保留：

- source commit；
- benchmark code；
- Go version；
- command；
- hardware；
- raw results；
- profile context。

未来维护者才能重建 reasoning。

## 30. Proof Structure

本仓库 proof 推荐：

```text
Claim
Cost Model
Baseline
Experiment
Verification
Benchmark
Environment
Caveats
Recommendation
```

## 31. Claim

Claim 应窄而具体。

例如：

> Dominating length check 可以让 compiler 删除 fixed-index reads 的 redundant bounds checks。

不要直接写：

> 这样 parser 会快很多。

## 32. Cost Model

解释机制为什么可能影响性能。

例如：

```text
redundant bounds checks
→ compare/branch/panic-edge machinery
```

## 33. Baseline

Baseline 应该合理，而不是故意糟糕。

通常是 current/idiomatic implementation。

## 34. Experiment

Mechanism proof 尽量只改变需要测试的那一个因素，降低 confounder。

## 35. Verification

优先用直接机制证据：

- compiler output；
- assembly；
- profile；
- PMU。

Timing alone 通常是 indirect evidence。

## 36. Benchmark

Measure local cost，并 repeated run。

涉及 allocation 时包含 `B/op` / `allocs/op`。

## 37. Environment

至少：

```text
Go version
GOOS
GOARCH
CPU
GOMAXPROCS
commands
```

## 38. Caveats

主动写明：

- compiler-version sensitive；
- low-contention only；
- amd64 only；
- small payload；
- retention not measured。

## 39. Recommendation

Proof 证明 mechanism，不自动等于 recommend。

可以分类：

```text
safe/default
conditional
advanced
diagnostic only
historical
avoid/private runtime
```

## 40. Evidence Matrix

| Claim | 最低有用证据 | 更强证据 |
|---|---|---|
| Heap escape exists | compiler `-m` | alloc benchmark |
| Bounds check remains | BCE diagnostic | assembly |
| Mutex contended | mutex profile | scaling benchmark |
| False sharing | scaling anomaly | PMU/cache evidence |
| Zero-copy helps | alloc/copy benchmark | system CPU + retention |
| PGO helps | A/B build | canary |

## 41. Causality

弱：

```text
changed code
benchmark faster
```

更强：

```text
changed layout
↓
cache misses decrease
↓
cycles/op decrease
↓
throughput increases
```

不要求每次都证明所有 link，但 mechanism clarity 会提高 confidence。

## 42. Confounders

常见：

- Go version；
- CPU frequency；
- workload distribution；
- data set；
- GC config；
- PGO profile；
- dependency version。

A/B 要尽量控制。

## 43. Correlation Trap

例如 deploy pool 后 GC CPU 下降，但 traffic 同时下降 20%。

没有 normalization/control，不能简单认为 pool 是原因。

## 44. Amdahl's Law 作为 Evidence Filter

如果 target path 只占总 CPU 1%，即使 local 2×，total CPU 理论上限也只有约 0.5%。

复杂优化可以在实现前就因此停止。

## 45. Evidence Budget

Evidence 也需要工程时间。

小、安全、可逆 change 可能：

```text
profile + benchmark
```

就足够。

Private runtime hack 则可能需要：

```text
profile
benchmark
system test
correctness stress
version CI
fallback
```

## 46. Reproducibility

不能 reproduce 的 result 会逐渐变成 folklore。

目标不是跨所有硬件完全一致，而是：

> 其他工程师能运行同一 experiment，并理解为什么结果可能不同。

## 47. Evidence Decay

证据会老化：

- compiler improves；
- runtime redesign；
- hardware changes；
- workload drift。

Proof 应被理解为 reproducible evidence，而不是 permanent law。

## 48. Revalidation Trigger

出现：

- major Go update；
- architecture change；
- workload change；
- optimization implementation change；

时应重新验证。

## 49. Evidence 与 Maintainability

Documentation 是 evidence preservation 的一部分。

Non-obvious source comment 可以引用已有 benchmark/proof，让 future maintainer 明白为什么不能随意“简化”。

## 50. Evidence 与 Review

Reviewer 应能回答：

- 问题是什么？
- mechanism 是什么？
- evidence 在哪里？
- trade-off 是什么？
- regression 怎么防？

回答不了，说明 optimization 还没有 review-ready。

## 51. 官方资料

- Diagnostics: https://go.dev/doc/diagnostics
- `testing`: https://pkg.go.dev/testing
- `runtime/pprof`: https://pkg.go.dev/runtime/pprof
- `benchstat`: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat

## 52. 工程视角

Evidence model 的目的不是 bureaucracy。

它的目的，是让 optimization confidence 与 optimization risk 匹配。

最终每个重要 recommendation 都应该能回答：

```text
What do we know?
How do we know it?
Under what conditions is it true?
```
