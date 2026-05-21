# Trace 机制说明

> 适用版本：modelALM / als  
> 相关文件：`lib/trace_grids.go`、`lib/trace_cache.go`、`formulae/diag_tracer.go`、`input/runsettings/diagnostics/diagnostics.yaml`、`config/products/RiskCalc.csv`

---

## 概述

Trace 是投影期间的**诊断落盘**机制，将运行时中间状态写出为 CSV 文件，用于调试和验证。  
核心设计原则：

- **零侵入**：计算公式（`CALC_*`）不含任何 trace 代码；trace 是独立的 `StatefulFormula`
- **nil-safe**：trace 关闭时 `Tracer` 为 `nil`，所有方法为 no-op，无性能损耗
- **配置驱动**：通过 `diagnostics.yaml` 开关，无需改代码即可开关和过滤
- **覆盖写**：每步 `Dump()` 覆盖同一文件，磁盘始终只有最后一步快照

---

## 统一架构

两种 trace 类型共享同一底层基础设施（`traceRegistry` / `traceConfigRegistry` / `TraceFunc`）：

| 类型 | 数据源 | 注册函数 | 示例 |
|---|---|---|---|
| **GridTrace** | `GetGrid(ctx, GridVarName)`，遍历 `(key, month, val)` | `RegisterGridTrace` | ZCB/SPD/RDF/ESD 曲线 |
| **CacheTrace** | `ctx.Cache[CacheKey]`，遍历 portfolio 内的记录切片 | `RegisterCacheTrace` | ExistingBond 逐条快照 |

区分方式：`TraceConfig.GridVarName` 非空 → GridTrace；空 → CacheTrace（by `CacheKey`）。

---

## TraceConfig — 统一配置结构

```go
type TraceConfig struct {
    Name        string   // snake_case base name，驱动所有 key 推导
    CacheKey    string   // ctx.Cache key（CacheTrace：portfolio 的存储键）
    Header      string   // CSV header；空 = 自动生成
    CapHint     int      // Tracer buffer 初始容量
    GridVarName string   // ArrayGrid varName；非空 → GridTrace
    DimNames    []string // GridTrace 维度名；空 = 运行时从 GridVarName 解析
}
```

`Name`（snake_case）驱动所有运行时 key 推导，规则不变：

| Name（代码） | tracerKey（ctx.Cache） | diagKey（yaml） |
|---|---|---|
| `zcb_curve` | `zcbCurveTracer` | `zcbCurveTrace` |
| `bond_record` | `bondRecordTracer` | `bondRecordTrace` |

---

## GridTrace

### 注册方式（`trace_grids.go`）

```go
var TRACE_ZCB_MONTHLY = formulae.RegisterGridTrace("TRACE_ZCB_MONTHLY", formulae.TraceConfig{
    Name:        "zcb_curve",
    GridVarName: "ZCB_CURVES",
    CapHint:     1200 * 40 * 200,
})
```

只需三个字段，其余（CSV header、DimNames、diagKey、文件路径）均自动推导。

### 过滤（白名单）

DimNames 一名两用：CSV 列名 + `diagnostics.yaml` 过滤字段名。

```yaml
zcbCurveTrace:
  enabled: 1
  economy: USD_HKPLG      # 只输出 economy == "USD_HKPLG" 的行

spdCurveTrace:
  enabled: 1
  economy: USD_HKPLG
  spreadBand: 1,2          # economy 且 spreadBand 同时满足才输出
```

### 数据桥门控

```go
// 只有 trace 启用时才写入 ArrayGrid，避免无谓分配
if formulae.IsGridTraceEnabled(ctx, TRACE_ESD_SNAPSHOT) {
    // 填充 ESD grid ...
}
```

### CSV 输出格式

```
T,Month,Year,economy,Val
0,1,0.0833,USD_HKPLG,0.9958123456
...
```

---

## CacheTrace

### 输出列由 struct tag 声明

在 `ExistingBond` 字段上加 `` `trace:"COL_NAME"` ``：

```go
type ExistingBond struct {
    Key         string  `trace:"KEY"`
    RedempYear  int     `tbl:"REDEMP_YEAR" trace:"REDEMP_YEAR"`
    SpreadBand  string  `tbl:"SPREAD_BAND" trace:"SPREAD_BAND"`
    // ...
    ABV float64 `trace:"ABV"`
    CF  float64 `trace:"CF"`
    MV  float64 `trace:"MV"`
}
```

- **列顺序** = 结构体字段声明顺序（带 tag 的字段）
- **固定首列** = `T`
- **新增列** = 只需加 tag，无需改其他代码

### 注册方式（`trace_cache.go`）

```go
var TRACE_BOND_RECORDS = formulae.RegisterCacheTrace[*ExistingBond, *BondPortfolio](
    "TRACE_BOND_RECORDS",
    formulae.TraceConfig{
        Name:     "bond_record",
        CacheKey: cacheKeyBonds,
        CapHint:  500 * 1200,
    },
)
```

`BondPortfolio` 需实现 `formulae.TraceSource[*ExistingBond]`（即 `TraceRecords() []*ExistingBond`）。

### CSV 输出格式

```
T,KEY,REDEMP_YEAR,SPREAD_BAND,...,ABV,CF,MV
0,BOND_001,2035,2,...,980000,0,980000
1,BOND_001,2035,2,...,975000,12000,975000
...
```

---

## diagnostics.yaml 配置

```yaml
globals:
  diagnostics:
    zcbCurveTrace:
      enabled: 1
      economy: USD_HKPLG

    spdCurveTrace:
      enabled: 1
      economy: USD_HKPLG
      spreadBand: 1,2

    rdfCurveTrace:
      enabled: 1
      economy: USD_HKPLG
      spreadBand: 1,2

    esdSnapshotTrace:
      enabled: 1

    bondRecordTrace:
      enabled: 1
```

**DiagKey 命名规则**：`lowerCamelCase(Name) + "Trace"`

---

## RiskCalc.csv 排序约定

Trace formula 必须排在对应 `CALC_*` **之后**（同一 Batch，`AscOrDesc=Asc`）：

```csv
default.CALC_ZCB_MONTHLY,   1,62,stateful,Asc,0
default.CALC_SPD_MONTHLY,   1,62,stateful,Asc,0
default.CALC_RDF_MONTHLY,   1,63,stateful,Asc,0
default.TRACE_ZCB_MONTHLY,  1,64,stateful,Asc,0
default.TRACE_SPD_MONTHLY,  1,64,stateful,Asc,0
default.TRACE_RDF_MONTHLY,  1,64,stateful,Asc,0
default.TRACE_ESD_SNAPSHOT, 1,64,stateful,Asc,0
default.CALC_BOND_PORTFOLIO,1,65,stateful,Asc,0
default.TRACE_BOND_RECORDS, 1,66,stateful,Asc,0
```

---

## 输出文件路径

```
<ResultPath>/.diag/<Name>_trace_<component>_<sim>_<plan>_<sp>.csv
```

---

## 添加新 Trace 的步骤

### 新增 GridTrace（Grid 曲线）

1. `trace_grids.go` 中加一行 `RegisterGridTrace`，填 `Name` + `GridVarName`
2. `diagnostics.yaml` 中加对应配置块（`lowerCamelCase(Name) + "Trace"`）
3. `RiskCalc.csv` 中在对应 `CALC_*` 之后加一行

### 新增 CacheTrace（记录列表）

1. 在目标结构体字段上加 `` `trace:"COL_NAME"` ``
2. portfolio 类型实现 `TraceRecords() []Item`
3. `trace_cache.go`（或新文件）中调用 `RegisterCacheTrace`
4. `diagnostics.yaml` + `RiskCalc.csv` 同上

### 对现有 CacheTrace 增加输出列

**只需在结构体字段上加 `trace` tag**，不需要改其他任何文件。

---

## 相关文件索引

| 文件 | 职责 |
|---|---|
| `formulae/diag_tracer.go` | 统一基础设施：Tracer、TraceConfig、RegisterGridTrace、RegisterCacheTrace |
| `lib/trace_grids.go` | GridTrace 声明（ZCB/SPD/RDF/ESD） |
| `lib/trace_cache.go` | CacheTrace 声明（BondPortfolio） |
| `lib/formula_bonds.go` | ExistingBond `trace` tag、TraceRecords() 方法 |
| `input/runsettings/diagnostics/diagnostics.yaml` | 开关与过滤配置 |
| `config/products/RiskCalc.csv` | Trace formula 的执行顺序 |

