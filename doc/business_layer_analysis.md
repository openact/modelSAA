# 业务层简洁性分析报告

> 分析范围：`als/app/modelALM/lib` 业务公式文件  
> 目标：找出可减少业务层负担的改进空间

---

## 1. 已完成的改进（本次 session）

### 1a. 引擎自动 Tick（`factory.go`）
**改前**：每个 StatefulFormula fn 手动调用 `ctx.Cache.Tick(t)`  
**改后**：`RegisterStatefulFormula` wrapper 自动调用，业务层零感知

### 1b. 引擎自动 IsCalcStep + Snapshot（`factory.go`，gridNames）
**改前**：fn 内手动 `if !ctx.IsCalcStep(t) { return }` + `grid.Snapshot()`  
**改后**：声明 `gridNames` 后，引擎负责门控和生命周期。fn 只写业务数据

```go
// 改后：fn 极简，无框架代码
var CALC_BOND_PORTFOLIO = formulae.RegisterStatefulFormula(...,
    func(ctx, t) {
        getBondPortfolio(ctx).evaluateAt(ctx, t)
    }, "BOND_VAL_PL")
```

### 1c. 移除 Fill 函数幂等检查（`ZCBFill`、`SPDFill`、`RDFFill`）
**问题**：`if _, ok := grid.Data[key]; ok { return }` 依赖 Tick 清空 Data。
当 grid 被设为 `retainPrev: true` 时，Tick 跳过，检查永远命中，导致使用 t=0 旧数据。  
**改后**：无幂等检查，每调必算，由 Tick 生命周期保证正确性

### 1d. 枚举迭代 helper（`esd_helpers.go`）
**改前**：ZCB/SPD/RDF 三处各有 ~8 行枚举读取 + 空值校验 + 两重 for 循环  
**改后**：统一由 `forEachEcon` / `forEachEconBand` 处理，fn 极简

```go
// CALC_SPD_MONTHLY：从 8 行缩减为 1 行有效逻辑
func(ctx, t) {
    forEachEconBand(ctx, "SPDMonthly", func(econ, band string) { SPDFill(ctx, econ, band) })
}
```

---

## 2. 当前业务层的最终负担

经过以上改进，业务公式 fn 的责任已降至最低：

| 公式 | fn 内容 | 说明 |
|------|---------|------|
| `CALC_ESD_DATA` | `store.stepTo(t)` + `updateESDGrid(ctx, t)` | 领域必须；ESD 步进是核心业务逻辑 |
| `CALC_ZCB_MONTHLY` | `forEachEcon(...)` | 1 行；enum 迭代不可消除（依赖 ECONOMY enum） |
| `CALC_SPD_MONTHLY` | `forEachEconBand(...)` | 1 行 |
| `CALC_RDF_MONTHLY` | `forEachEconBand(...)` + zero-band | 2 行；zero-band 特例是领域逻辑 |
| `CALC_BOND_PORTFOLIO` | `evaluateAt(ctx, t)` | 1 行；纯业务 |

---

## 3. 潜在进一步优化（架构层面，暂未实现）

### 3a. `RegisterESDGridFormula`（引擎层新接口）

ZCB/SPD 公式的模式完全一致：  
**"遍历枚举组合 → 对每个 key 调用 Fill(ctx, keys...) → 写 grid"**

如果引擎提供这个接口，业务层只需声明填充逻辑，无需任何迭代代码：

```go
// 假设新接口（未实现）
var CALC_ZCB_MONTHLY = formulae.RegisterESDGridFormula(
    formulae.Registry, "default", "CALC_ZCB_MONTHLY",
    "ZCB_CURVES",
    func(ctx *formulae.ProjContext, keys ...string) {
        ZCBFill(ctx, keys[0]) // economy
    },
    "ECONOMY")

var CALC_SPD_MONTHLY = formulae.RegisterESDGridFormula(
    formulae.Registry, "default", "CALC_SPD_MONTHLY",
    "SPD_CURVES",
    func(ctx *formulae.ProjContext, keys ...string) {
        SPDFill(ctx, keys[0], keys[1]) // economy, band
    },
    "ECONOMY", "SPREAD_BAND")
```

**实现代价**：中等（需在 `formulae/factory.go` 新增注册函数）  
**收益**：消除 lib 中全部枚举迭代代码，注册声明即文档  
**限制**：RDF 的 zero-band 特例（不在 enum 内）无法用此接口统一，仍需手动处理

### 3b. Fill 函数签名统一

目前 Fill 函数签名不一致：
```go
ZCBFill(ctx, economy string)
SPDFill(ctx, economy, spreadBand string)
RDFFill(ctx, economy, spreadBand string)
```

可统一为：
```go
type GridFillFn func(*formulae.ProjContext, ...string)
```

配合 `RegisterESDGridFormula` 可实现完全对称的 API，但 Go 的可变参数会损失类型安全。  
**结论**：不建议为了统一而牺牲类型安全

### 3c. 批次依赖声明（当前通过 YAML 顺序维护）

RDF 依赖 ZCB 和 SPD 已先填充（同 Batch Asc 顺序），当前靠 StructureALM.csv 中的顺序保证。  
若引擎支持声明式依赖 `dependsOn: ["CALC_ZCB_MONTHLY", "CALC_SPD_MONTHLY"]`，则可在注册时静态校验。  
**实现代价**：较高（引擎 sequencing 层修改）；当前风险可接受

### 3d. ESD Grid 桥接（`updateESDGrid`）自动化

`updateESDGrid` 将 ESDStore 的 snap 写入 ESD grid 以供 trace 消费。  
此模式（cache 对象 → grid 桥接，仅 trace 启用时执行）可抽象为引擎级 `RegisterCacheToGridBridge`，但
当前仅有一处使用，泛化收益有限。

---

## 4. 总结

| 维度 | 当前状态 | 最优状态（若实现 3a） |
|------|---------|---------------------|
| 框架代码在 fn 中 | 零 | 零 |
| 枚举迭代代码 | lib helper（1行/公式） | 引擎负责（0行/公式） |
| Fill 函数 | 纯业务，无样板 | 不变 |
| Snapshot/IsCalcStep | 引擎自动 | 不变 |
| 依赖声明 | YAML 顺序 | 可声明化 |

**当前设计已达到较高的简洁性**。剩余改进点（3a）属于引擎层扩展，投入适中，但非紧迫需求——
现有的 `forEachEcon/Band` helper 已将重复代码收敛到单一位置，维护成本可控。

