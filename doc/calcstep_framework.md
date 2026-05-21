# CalcStep 框架设计文档

> 适用范围：`als/app/modelALM/lib`（业务层）+ `als/formulae`（引擎层）

---

## 1. 概述

CalcStep 框架允许不同日历年使用不同的计算频率。例如 step=1 表示每月计算，step=12 表示每年计算（年度估值）。框架的核心目标是：

- **引擎层**：统一管理时步推进、Grid 生命周期、IsCalcStep 门控
- **业务层**：只写纯业务逻辑，不感知框架细节
- **输入层**：通过 YAML 声明 CalcStep 断点表，对代码零侵入

---

## 2. 三层职责划分

### 2a. 引擎层（`formulae` 包）负责

| 职责 | 实现位置 | 说明 |
|------|---------|------|
| **Tick(t)**：推进 Cache 时步 | `Cache.Tick(t)` / `RegisterStatefulFormula` wrapper | 每步由第一个 StatefulFormula 的 wrapper 触发，幂等 |
| **Stream Grid 清空**：每步清空 Data | `Cache.Tick(t)` | `RetainPrev=false` 的 Grid，Tick 分配新空 map |
| **Snapshot Grid 跳过**：Tick 不清空 | `Cache.Tick(t)` | `RetainPrev=true` 的 Grid，Tick 完全跳过 |
| **IsCalcStep 门控**：非 calc step 跳过 fn | `RegisterStatefulFormula` wrapper | 仅当声明了 `gridNames` 时生效 |
| **Snapshot() 调用**：归档上步值 | `RegisterStatefulFormula` wrapper | 仅在 calc step 时，对每个声明的 `gridName` 调用 |
| **GetOrInitCache**：懒初始化 | `formulae.GetOrInitCache` | 首次调用时执行初始化，之后幂等 |
| **CalcStep 计算**：t → year → step | `Setting.GetCalcStepAtT(t)` | 从 Global.CalcSteps 断点表查找 |
| **IsCalcStep 幂等**：同 t 多次调用安全 | `ProjContext.IsCalcStep(t)` | `lastCalcT == t` 直接返回 true |

### 2b. 业务层（`lib` 包）负责

| 职责 | 典型例子 | 说明 |
|------|---------|------|
| **每步公式**：无条件每步执行 | `CALC_ESD_DATA`、`CALC_ZCB_MONTHLY` | 不声明 `gridNames`，fn 每步运行 |
| **Calc step 公式**：声明 gridNames | `CALC_BOND_PORTFOLIO` | 声明 `"BOND_VAL_PL"`，引擎自动门控 |
| **数据写入**：Fill 函数只管写 | `ZCBFill`、`SPDFill`、`RDFFill` | 不做幂等检查，每调必算 |
| **ESD 步进**：`store.stepTo(t)` | `CALC_ESD_DATA` fn | 推进 ESD 快照到当前 t |
| **债券组合滚动**：`evaluateAt(t)` | `BondPortfolio.evaluateAt` | 纯业务：推进 CF/ABV/MV，写 grid |

**业务层不需要调用**：
- `ctx.Cache.Tick(t)` ← 引擎 wrapper 已自动调用
- `grid.Snapshot()` ← 引擎 wrapper 已自动调用（通过 gridNames 声明）
- `ctx.IsCalcStep(t)` ← 声明了 gridNames 时，引擎已门控，fn 无需判断

### 2c. 输入层（YAML 配置）控制

| 配置项 | 文件位置 | 说明 |
|--------|---------|------|
| **CalcSteps 断点表** | `parameters/default.yaml` → `globals.calcSteps` | 断点 hold-last：只需写变化年份 |
| **Grid 模式声明** | `grids/default.yaml` → `retainPrev` | `true` = snapshot，`false`（默认）= stream |

---

## 3. StatefulFormula 注册模式

### 3a. 每步执行（无 gridNames）

```go
// 每步都运行；fn 负责全部逻辑
var CALC_ZCB_MONTHLY = formulae.RegisterStatefulFormula(
    formulae.Registry, "default", "CALC_ZCB_MONTHLY",
    func(ctx *formulae.ProjContext, t int) {
        for _, econ := range economies {
            ZCBFill(ctx, econ) // 无幂等检查，直接写入
        }
    }) // 不传 gridNames
```

引擎 wrapper 行为：
```
Tick(t) → fn(ctx, t)   // 每步
```

### 3b. Calc step 执行（有 gridNames）

```go
// 仅在 calc step 执行；fn 只写业务数据
var CALC_BOND_PORTFOLIO = formulae.RegisterStatefulFormula(
    formulae.Registry, "default", "CALC_BOND_PORTFOLIO",
    func(ctx *formulae.ProjContext, t int) {
        getBondPortfolio(ctx).evaluateAt(ctx, t) // 纯业务
    },
    "BOND_VAL_PL") // 声明 snapshot grid
```

引擎 wrapper 行为：
```
Tick(t)
  ├─ IsCalcStep(t) == false → return（跳过整步）
  └─ IsCalcStep(t) == true
       → Snapshot("BOND_VAL_PL")  // PrevData = 上calc step值，Data = 空
       → fn(ctx, t)               // 写入新值
```

---

## 4. Grid 生命周期

### Stream Grid（`retainPrev: false`，默认）

适用：每步全量刷新的数据（ZCB/SPD/RDF 月度曲线、CF 流量）。

```
t=N:  Tick(N): Data = {}（清空）
      Fill fn:  Data[key] = 新值
      读取：     Data[key]         ← 当步新值
      PrevData： 始终 nil
```

### Snapshot Grid（`retainPrev: true`）

适用：仅在 calc step 更新、非 calc step 保持上次值的数据（BOND_VAL_PL）。

```
t=0（calc step）:
  Tick(0):      跳过此 Grid
  Snapshot():   PrevData = {}（空），Data = {}（清空）
  fn:           Data[pool] = MV_0

t=1..11（非 calc step，step=12）:
  Tick:         跳过
  IsCalcStep=false → 整个 fn 跳过
  Data[pool]：  仍为 MV_0（hold-last）

t=12（calc step）:
  Tick(12):     跳过此 Grid
  Snapshot():   PrevData = {pool: MV_0}，Data = {}（清空）
  fn:           Data[pool] = MV_12
  读 PrevData： MV_0（上一 calc step 的值）
```

---

## 5. CalcSteps YAML 配置

断点 hold-last 规则：只需写发生变化的年份，之后的年份自动继承最后一个值。

```yaml
globals:
  calcSteps:
    2025: 1    # 2025 年：每月计算
    2026: 1    # 2026 年：每月计算
    2027: 12   # 2027 年起：每年计算（annual）
    # 2028 年及之后：继承 12（hold-last，无需重复写）
```

`GetCalcStep(year)` 逻辑：找 `CalcSteps` 中 `≤ year` 的最大 key，返回对应 step；若无匹配则返回 1。

---

## 6. Fill 函数设计规范

数据 Fill 函数（`ZCBFill`、`SPDFill`、`RDFFill`）**不应包含幂等检查**：

```go
// ✅ 正确：每调必算，由引擎 Tick 生命周期保证正确性
func ZCBFill(ctx *formulae.ProjContext, economy string) {
    grid := ctx.Grid("ZCB_CURVES")
    key  := grid.Key(economy)
    // ... 直接计算并写入 grid.Data[key]
}

// ❌ 错误：依赖 Tick 清空的幂等检查，retainPrev=true 时会读到脏数据
func ZCBFill(...) {
    if _, ok := grid.Data[key]; ok {
        return // 当 retainPrev=true 时，Tick 不清空，旧值永远被读到
    }
}
```

---

## 7. 数据流总览

```
输入 YAML
  calcSteps → Setting.Global.CalcSteps
  retainPrev → Setting.Grids[varName].RetainPrev

每步执行流（Asc 顺序）
  ┌─ CALC_ESD_DATA ──── 无 gridNames ──→ Tick(t) → store.stepTo(t)
  ├─ CALC_ZCB_MONTHLY ─ 无 gridNames ──→ Tick(t,幂等) → ZCBFill × economies
  ├─ CALC_SPD_MONTHLY ─ 无 gridNames ──→ Tick(t,幂等) → SPDFill × (econ×band)
  ├─ CALC_RDF_MONTHLY ─ 无 gridNames ──→ Tick(t,幂等) → RDFFill × (econ×band)
  └─ CALC_BOND_PORTFOLIO ─ gridNames=["BOND_VAL_PL"]
       IsCalcStep?
         No  → skip（hold-last）
         Yes → Snapshot(BOND_VAL_PL) → evaluateAt(t)
                 └─ stepTo(t) × bonds → 读 RDF/ZCB/SPD → 写 BOND_VAL_PL.Data
```

