# formula_bonds 未来改进路线图

本文记录 `lib/formula_bonds.go` 当前实现的三块短板及补齐方案，按"成本低、收益大"优先级排列。
当前实现已经满足生产精度与性能要求，以下条目按需引入，不做过早优化。

---

## 1. 易用性：精算师 cookbook（成本最低）

**痛点**：精算师只想新增一条 `BOND_*` 标量指标（例如 `BOND_DURATION`），但要先理解
`StatefulFormula` / `ScalarNum` / Cache / Batch 排序等机制，门槛偏高。

**方案**：在 `als/doc/` 下新增 `cookbook-bond-formula.md`，3 段代码即可覆盖 90% 场景。

### 模板

#### 步骤 1：在 `BondMetrics` / `ExistingBond` 上加字段（如需新维度）
```go
type BondMetrics struct {
    CF, ABV, MV float64
    Duration    float64 // ← 新增
}
```

#### 步骤 2：在 `addBond` 里聚合（口径：单券 × Scalar 累加）
```go
func (s *BondMetrics) addBond(b *ExistingBond) {
    // ...existing...
    s.Duration += b.Duration * b.Scalar
}
```

#### 步骤 3：注册公式（一行注册 + 一行读取）
```go
var BOND_DURATION = formulae.RegisterScalarNum(formulae.Registry, "default", "BOND_DURATION",
    func(ctx *formulae.ProjContext, i int, dims ...int) float64 {
        return bondStep(ctx).Duration
    })
```

### 隐含约定（cookbook 必须明示）
- 在 Assembly 表里把 `BOND_DURATION` 排在 `CALC_BOND_PORTFOLIO` 之后、同一 Batch、`AscOrDesc=Asc`。
- 单券 `Duration` 在 `stepTo` 里和 ABV/MV 一起算，避免热路径多次扫切片。

---

## 2. 可观测性：debug 模式落盘单券明细

**痛点**：当前只保留组合级 `bp.Metrics` 当期快照，发现总量异常时无法定位是哪只债券、哪个时点。
modelx 的天然优势是"每个 cell 都可点开"，本库需要补上对等的 debug 体验。

**方案**：在 `BondPortfolio` 上加可选 sink，由环境变量或 setting 开关控制，默认关闭，零开销。

### 数据结构
```go
type BondDebugRow struct {
    T      int
    Key    string  // 债券 ID
    CF     float64
    ABV    float64
    MV     float64
    Scalar float64
}

type BondPortfolio struct {
    bonds   []*ExistingBond
    Metrics BondMetrics
    debug   []BondDebugRow // nil 表示关闭
}
```

### 开关
- `ctx.Setting.GetParameter("bondDebug") == 1` 时，`initBondPortfolio` 给 `bp.debug` 分配容量
  `len(bonds) * projMonths`。
- `ExistingBond` 需要外部知道 `Key`，因此在 `loadExistingBond` 里把 `key` 存到 `b.Key string`。

### 落盘点
```go
func (bp *BondPortfolio) evaluateAt(ctx *formulae.ProjContext, t int) {
    bp.Metrics = BondMetrics{}
    for _, b := range bp.bonds {
        b.stepTo(ctx, t)
        bp.Metrics.addBond(b)
        if bp.debug != nil {
            bp.debug = append(bp.debug, BondDebugRow{t, b.Key, b.CF, b.ABV, b.MV, b.Scalar})
        }
    }
}
```

### 输出
- run 结束时由 orchestrator 调一次 `bp.dumpDebug(path)` 写 CSV/Parquet。
- 列：`T, Key, CF, ABV, MV, Scalar`，可在 Excel/Python 里透视到 modelx 的"单 cell"粒度。

### 成本评估
- 关闭时：1 次 nil 判断，无分支预测损失。
- 打开时：`append` 一行 ~48 B，10000 券 × 360 月 ≈ 165 MB，可接受；如需更省内存改成
  `bufio.Writer` 流式写盘。

---

## 3. 历史保留：可选 ring buffer

**痛点**：当前 `bp.Metrics` 只保留当期快照，下游公式（如年化 unrealised G/L 累计、
回看 12 月 CF 平均）没法直接用，需要去 `ResultsStore` 反查（一次额外 O(t) 扫描）。

**方案**：给 `BondPortfolio` 加可选环形缓冲区，用户按需开启。

### 数据结构
```go
type BondPortfolio struct {
    bonds   []*ExistingBond
    Metrics BondMetrics

    // 可选历史，nil = 关闭。容量在 init 时按 `bondHistorySize` 参数确定。
    history    []BondMetrics // ring buffer，长度 = cap
    historyEnd int           // 写指针（下一个写入位置）
}

// At returns the metrics k steps before current (k=0 → current, k=1 → 上一步).
// 越界返回 (BondMetrics{}, false)。
func (bp *BondPortfolio) At(k int) (BondMetrics, bool) {
    if bp.history == nil || k >= len(bp.history) {
        return BondMetrics{}, false
    }
    idx := (bp.historyEnd - 1 - k + len(bp.history)) % len(bp.history)
    return bp.history[idx], true
}
```

### 写入点
```go
func (bp *BondPortfolio) evaluateAt(ctx *formulae.ProjContext, t int) {
    // ...existing 累加 Metrics ...
    if bp.history != nil {
        bp.history[bp.historyEnd] = bp.Metrics
        bp.historyEnd = (bp.historyEnd + 1) % len(bp.history)
    }
}
```

### 使用样例
```go
var BOND_CF_AVG_12M = formulae.RegisterScalarNum(formulae.Registry, "default", "BOND_CF_AVG_12M",
    func(ctx *formulae.ProjContext, i int, dims ...int) float64 {
        bp := getBondPortfolio(ctx)
        sum, n := 0.0, 0
        for k := 0; k < 12; k++ {
            if m, ok := bp.At(k); ok {
                sum += m.CF
                n++
            }
        }
        if n == 0 { return 0 }
        return sum / float64(n)
    })
```

### 成本评估
- 关闭：1 次 nil 判断，0 内存。
- 打开 `cap=12`：每只债券组合每月多 1 次结构体复制 (~24 B)，可忽略。

---

## 实施次序建议

| 阶段 | 项 | 工作量 | 收益 |
| --- | --- | --- | --- |
| 1 | cookbook | 0.5 天 | 立刻降低协作门槛 |
| 2 | debug 落盘 | 1 天 | 排错效率追上 modelx |
| 3 | history ring | 0.5 天 | 解锁回看类公式 |

三项都向后兼容，互不依赖，可独立推进。

