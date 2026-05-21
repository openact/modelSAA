# BOND_* 公式 Cookbook

> 面向精算师 / 模型开发者：在 `formula_bonds.go` 里新增一条公司级（标量）债券公式，
> 通常**只需 3 段代码**。本文用 3 个最常见的场景手把手演示。

---

## 0. 背景速览

债券模块的运行约定（详见 `formula_bonds.go` 文件头注释）：

1. **驱动者**：`CALC_BOND_PORTFOLIO`（StatefulFormula）每个 `t` 调用一次，
   把组合评估结果写入 `bp.Metrics`（`BondMetrics{CF, ABV, MV}`）。
2. **读取者**：所有 `BOND_*` 公式都是 `RegisterScalarNum`，
   只做 `bondStep(ctx).X` 的 O(1) 读，**不重算**。
3. **Assembly 顺序**：同一 Batch 中 `CALC_BOND_PORTFOLIO` 必须排在所有 `BOND_*` 之前，
   `AscOrDesc=Asc`。

> 因此，"加一条公式"几乎从不修改 `evaluateAt` / `stepTo`——
> 90% 的需求只是给 `BondMetrics` 加一个字段，再注册一条 `BOND_*` 读它即可。

---

## 场景 A：复合现有指标（最简单，1 段代码）

需求示例：`BOND_ABV_PER_MV = ABV / MV`（账面/市值比）。

```go
var BOND_ABV_PER_MV = formulae.RegisterScalarNum(formulae.Registry, "default", "BOND_ABV_PER_MV",
    func(ctx *formulae.ProjContext, i int, dims ...int) float64 {
        st := bondStep(ctx)
        if st.MV == 0 {
            return 0
        }
        return st.ABV / st.MV
    })
```

把这一段贴到文件 §4 末尾即可。**不需要改 Assembly**——但记得把
`BOND_ABV_PER_MV` 加到产品表里需要展示的列。

---

## 场景 B：新增一个组合级聚合量（3 段代码）

需求示例：`BOND_COUPON_CF`——只统计派息现金流（不含到期本金还本）。

### 1) 在 `BondMetrics` 加字段

```go
type BondMetrics struct {
    CF       float64
    CouponCF float64 // ← 新增
    ABV      float64
    MV       float64
}
```

### 2) 在 `ExistingBond` 拆分单券现金流，让 `addBond` 累加

> 思路：把 `cfs` 拆成 `couponCfs` 和 `redempCfs`（或在 `BuildBondCF` 旁加一份），
> `stepTo` 内同时累加两条窗口；`addBond` 多加一行：

```go
func (s *BondMetrics) addBond(b *ExistingBond) {
    s.CF       += b.CF       * b.Scalar
    s.CouponCF += b.CouponCF * b.Scalar // ← 新增
    s.ABV      += b.ABV      * b.Scalar
    s.MV       += b.MV       * b.Scalar
}
```

### 3) 注册公式

```go
var BOND_COUPON_CF = formulae.RegisterScalarNum(formulae.Registry, "default", "BOND_COUPON_CF",
    func(ctx *formulae.ProjContext, i int, dims ...int) float64 {
        return bondStep(ctx).CouponCF
    })
```

完成。Assembly 不变（`CALC_BOND_PORTFOLIO` 已经在最前）。

---

## 场景 C：按经济体（CNY / USD / HKD）分组聚合

需求示例：`BOND_MV_CNY` / `BOND_MV_USD` …

### 1) `BondMetrics` 改成 map（或固定字段）

```go
type BondMetrics struct {
    CF, ABV, MV float64
    MvByEcon    map[string]float64 // ← 新增
}
```

### 2) `addBond` 同时按 economy 分桶

```go
func (s *BondMetrics) addBond(b *ExistingBond) {
    s.CF  += b.CF  * b.Scalar
    s.ABV += b.ABV * b.Scalar
    s.MV  += b.MV  * b.Scalar
    if s.MvByEcon == nil {
        s.MvByEcon = make(map[string]float64, 4)
    }
    s.MvByEcon[b.Econ] += b.MV * b.Scalar
}
```

> 注意 `evaluateAt` 已经在每步 `bp.Metrics = BondMetrics{}` 重置，map 会自然清空。

### 3) 每个经济体注册一条公式

```go
var BOND_MV_CNY = formulae.RegisterScalarNum(formulae.Registry, "default", "BOND_MV_CNY",
    func(ctx *formulae.ProjContext, i int, dims ...int) float64 {
        return bondStep(ctx).MvByEcon["CNY"]
    })

var BOND_MV_USD = formulae.RegisterScalarNum(formulae.Registry, "default", "BOND_MV_USD",
    func(ctx *formulae.ProjContext, i int, dims ...int) float64 {
        return bondStep(ctx).MvByEcon["USD"]
    })
```

---

## Checklist（提交前自检）

- [ ] 新公式名以 `BOND_` 开头，全大写蛇形，与产品表/Assembly 列名一致。
- [ ] 公式 lambda 内**不做循环、不读 `cfs`**——只 `bondStep(ctx).X` 读快照。
- [ ] 若改了 `BondMetrics` 字段，对应分子在 `addBond` 里**乘了 `b.Scalar`**。
- [ ] Assembly 中 `CALC_BOND_PORTFOLIO` 仍排在所有 `BOND_*` 之前，`AscOrDesc=Asc`。
- [ ] 产品表（输出列）里加了新公式名；如不需要输出，则只需源码里的 `var _ = ...` 即可保活。

---

## FAQ

**Q1：我能不能在 `BOND_*` 里直接 for-loop 算？**
不要。会被同 Batch 内多条公式各算一遍。重活只在 `evaluateAt` 里做一次，`BOND_*` 只读快照。

**Q2：t=0 怎么处理？**
`evaluateAt` 是幂等的——`init` 已经为你写好 t=0 快照，引擎调 `CALC_BOND_PORTFOLIO(0)`
也会得到完全一致的结果，不会"翻倍"。

**Q3：未来要支持新购债券（不是评估日就在的）怎么办？**
`ExistingBond` 已经预留 `EntryT` 字段。买入逻辑里 `append` 一只 `EntryT=tBuy` 的债券即可，
`stepTo` 自动跳过 `t < EntryT` 的步。所有现存 `BOND_*` 公式不需要改。

