# modelALM_SAA — 模型文档

> 本文档整合了输入表、枚举、矩阵及变量伪代码的完整参考，供开发与审阅使用。

---

## 目录

1. [输入表（tables/）](#1-输入表tables)
2. [枚举（enums/）](#2-枚举enums)
3. [矩阵（matrices/）](#3-矩阵matrices)
4. [变量伪代码](#4-变量伪代码)

---

# 1. 输入表（tables/）

所有表存放于 `input/tables/`，格式为 `.fac`（内部 CSV 变体）。

`.fac` 格式约定：
- `!2` / `!3` — 行首标记：后面紧跟的整数表示键列数（key columns）
- `*` — 数据行标记
- 列值按 col 索引读取（`ReadTableNum(name, "Y", key...)` 中 `"Y"` 表示按时间步列读取）

---

## Assump.fac

**用途**：各资产类别的核心假设，逐行按 `ASSET_CLASS` 索引。
**键列**：`ASSET_CLASS`（1 个键）

| 列名 | 含义 |
|------|------|
| `CLASS` | 资产大类（Fixed_Income / Equity / Alternatives / Cash / Dummy 等） |
| `BOND_TYPE` | 债券子类型（IG / HY / Private_Credit，非债券为空） |
| `ASSET_MIX` | 默认资产配置比例（被 SAA_Sims 按情景覆盖） |
| `NAR_TOT` | NAR 总倍数（`ASSET_BAL × NAR_TOT = NAR_TOT`） |
| `NAR_FI` | 固定收益 NAR 倍数 |
| `NAR_EQ` | 权益 NAR 倍数 |
| `NAR_PROP` | 房地产 NAR 倍数 |
| `RET_RATE` | 年化预期回报率 |
| `SD_RATE` | 年化标准差（波动率） |
| `ASSET_HKD_MIX` | 港币敞口比例（用于 FX 风险资本计算） |
| `ASSET_USD_MIX` | 美元敞口比例 |
| `RC_FAC_EQ` | 权益资产风险资本因子 |
| `RC_FAC_PROP` | 房地产资产风险资本因子 |

**资产类别**：AC · IG_Tradable · HY · Securitized_CMBS · DM · EM · PE · Loan_Private_Credit · Property · Hedging_Fund · Convertible_Bond · QIS · Loan_Private_Credit_New · Cash · Deposit_ST_Govern_Bond · Seed_Capital · Other_Asset · Dummy_2~7

---

## Param.fac

**用途**：按产品 (`PROD_NAME`) 和时间步（列）存放随时间变化的参数，供 `RegisterParamFormula` 读取。
**键列**：`PROD_NAME`（1 个键），列 1..n 对应时间步 i

| 列名 | 含义 |
|------|------|
| `RC_FAC_HKD` | 港币 FX 风险资本因子（时间步级） |
| `RC_FAC_USD` | 美元 FX 风险资本因子 |
| `RC_FAC_CNY` | 人民币 FX 风险资本因子 |

> 当前包含 `FUND01` / `FUND02` 两条产品记录。

---

## SAA_Sims.fac

**用途**：按模拟情景编号给出战略资产配置（SAA）权重，用于 `ASSET_MIX`。
**键列**：`SIMULATION`（1 个键）

每列对应一个 `ASSET_CLASS`，值为该情景下的资产配置权重（合计应为 1）。当前包含情景 1 和 2。

> 被 `ASSET_MIX[i, class] = Table("SAA_Sims", "Y", sim, class)` 按模拟编号逐行读取。

---

## AssetTradBondAssump.fac

**用途**：传统债券（Tradable Bond）各期限段的久期假设，用于 `TRAD_BOND_DUR`。
**键列**：`BOND_TERM`（1 个键）

| 期限段 | 含义 |
|--------|------|
| `TERM_5` | ≤5 年期债券组合 |
| `TERM_10` | ≤10 年期 |
| `TERM_15` | ≤15 年期 |
| `TERM_20` | ≤20 年期 |
| `TERM_25` | ≤25 年期 |
| `TERM_30` | ≤30 年期 |
| `TERM_100` | >30 年期（超长期，与 TERM_30 同久期） |

列 `DURATION`：该期限段的有效久期（年）。

---

## AssetAltBondAssump.fac

**用途**：另类债券（Alternative Bond，即 Private Credit / Loan）各期限段的久期假设，用于 `ALT_BOND_DUR`。
**键列**：`BOND_TERM`（同上）

结构与 `AssetTradBondAssump.fac` 完全一致，当前仅 `TERM_5` 有非零久期（2.21），其余期限段设为 0（另类债券集中于短端）。

---

## AssetTradBondDistByRatingByTerm.fac

**用途**：传统债券按信用评级和期限段的持仓分布矩阵，用于 `TRAD_BOND_DIST`。
**键列**：`BOND_RATING`（行键），期限段作为列（`TERM_5` … `TERM_100`）

| 评级行 | 说明 |
|--------|------|
| `GOVT_BOND` | 政府债 |
| `RATING_1` | 最高信用等级（近似 AAA） |
| `RATING_2` ~ `RATING_7` | 逐级下降 |
| `NON_RATED` | 无评级 |

每格值 = 该评级期限组合占传统债券总 NAR 的比例；全矩阵之和 = 1（整体配置 100%）。

---

## AssetAltBondDistByRatingByTerm.fac

**用途**：另类债券按评级和期限的持仓分布矩阵，用于 `ALT_BOND_DIST`。
**键列**：`BOND_RATING`（行键）

结构与 `AssetTradBondDistByRatingByTerm.fac` 一致。当前另类债券全部归入 `NON_RATED` × `TERM_5`（值 = 1.0），反映私人信贷无外部评级、短期限特征。

---

## StressBondSpdByRatingByTerm.fac

**用途**：压力情景下各评级期限组合的利差扩大幅度（spread widening），用于 `STRESS_BOND_SPD_PARAMS` 和 `PCR_SPD` 计算。
**键列**：`BOND_RATING`（行键），期限段作为列

所有值为**负数**，表示债券价值在利差扩张时的变动幅度（绝对值越大 = 评级越低 / 期限越长 = 风险越高）。

- `GOVT_BOND`：全零（政府债无信用利差风险）
- `NON_RATED`：介于 RATING_4～5 之间的保守估计

---

## RF_Curves.fac

**用途**：无风险利率曲线，按货币（ECON）和压力情景（SCENARIO）提供期限结构，用于利率风险（`PCR_INT`）和 `RF_ECON_SCENARIO` 计算。
**键列**：`ECON` + `SCENARIO`（2 个键），列 1..46 对应久期（年）

| ECON | SCENARIO | 说明 |
|------|----------|------|
| `HKD` | `BASE` | 港币基准利率曲线 |
| `USD` | `BASE` | 美元基准利率曲线 |
| `CNY` | `BASE` | 人民币基准利率曲线 |
| `HKD` | `INT_UP` | 港币利率上行压力曲线（+200 bps 附近） |
| `USD` | `INT_UP` | 美元利率上行压力曲线 |
| `CNY` | `INT_UP` | 人民币利率上行压力曲线 |
| `HKD` | `INT_DN` | 港币利率下行压力曲线 |
| `USD` | `INT_DN` | 美元利率下行压力曲线 |
| `CNY` | `INT_DN` | 人民币利率下行压力曲线 |

列索引 = `BOND_DUR_INT[i]`（整数化久期），读取对应期限点的利率值，用于计算 `HKD_BOND_MV_CHG` / `USD_BOND_MV_CHG`。

> `mp.INT_BITING` 字段决定使用 `INT_UP` 还是 `INT_DN` 情景，对应 PCR 计算的 biting 方向（利率上行更不利 vs 下行更不利）。

---

# 2. 枚举（enums/）

所有枚举存放于 `input/enums/`，格式为 `.csv`。

**格式约定**：三列格式：`KEY`（整数序号）、`VALUE`（枚举值名称）、`DESCRIPTION`（描述，`na` 表示暂无）。枚举维度在公式中通过 `coord.DimValue("ENUM_NAME")` 读取，用于索引向量变量（`RegisterVectorNum` / `RegisterVectorTxt`）。

---

## ASSET_CLASS.csv

**用途**：资产类别枚举，模型最核心的向量维度，贯穿所有资产级别变量（`ASSET_MIX`、`ASSET_BAL`、`NAR_FI`、`NAR_EQ`、`NAR_PROP`、`RC_ASSET_EQ` 等）。

| KEY | VALUE | 说明 |
|-----|-------|------|
| 1 | `AC` | 固定收益综合（Annuity Contracts 或 General Fixed Income） |
| 2 | `IG_Tradable` | 投资级可交易债券 |
| 3 | `HY` | 高收益债券 |
| 4 | `Securitized_CMBS` | 证券化/商业抵押债券 |
| 5 | `DM` | 发达市场股票 |
| 6 | `EM` | 新兴市场股票 |
| 7 | `PE` | 私募股权 |
| 8 | `Loan_Private_Credit` | 私人信贷贷款（现有） |
| 9 | `Property` | 房地产 |
| 10 | `Hedging_Fund` | 对冲基金 |
| 11 | `Convertible_Bond` | 可转换债券 |
| 12 | `QIS` | 量化投资策略 |
| 13 | `Loan_Private_Credit_New` | 私人信贷贷款（新增） |
| 14–19 | `Dummy_2` ~ `Dummy_7` | 占位类别（相关系数与配置均为 0） |
| 20 | `Cash` | 现金 |
| 21 | `Deposit_ST_Govern_Bond` | 存款 / 短期政府债 |
| 22 | `Seed_Capital` | 种子资本 |
| 23 | `Other_Asset` | 其他资产 |

---

## ENUM_BOND_RATING.csv

**用途**：债券信用评级枚举，作为 `TRAD_BOND_DIST`、`ALT_BOND_DIST`、`STRESS_BOND_SPD_PARAMS`、`PIVOT_RC_SPD_BOND_*` 等向量变量的评级维度。

| KEY | VALUE | 说明 |
|-----|-------|------|
| 1 | `GOVT_BOND` | 政府债（零信用利差风险） |
| 2 | `RATING_1` | 最高信用等级（近似 AAA/AA） |
| 3–8 | `RATING_2` ~ `RATING_7` | 逐级下降（RATING_7 接近 CCC） |
| 9 | `NON_RATED` | 无外部评级（私人信贷等） |

---

## ENUM_BOND_TERM.csv

**用途**：债券期限段枚举，作为 `TRAD_BOND_DUR`、`ALT_BOND_DUR`、`TRAD_BOND_DIST`、`ALT_BOND_DIST` 等向量变量的期限维度。与 `AssetTradBondAssump.fac` / `AssetAltBondAssump.fac` 及 `*DistByRatingByTerm.fac` 的列结构一一对应。

| KEY | VALUE | 说明 |
|-----|-------|------|
| 1 | `TERM_5` | ≤5 年期 |
| 2 | `TERM_10` | ≤10 年期 |
| 3 | `TERM_15` | ≤15 年期 |
| 4 | `TERM_20` | ≤20 年期 |
| 5 | `TERM_25` | ≤25 年期 |
| 6 | `TERM_30` | ≤30 年期 |
| 7 | `TERM_100` | >30 年期（超长期） |

---

## ENUM_ECON.csv

**用途**：货币 / 经济体枚举，作为 `RF_ECON_SCENARIO` 向量变量的货币维度，与 `RF_Curves.fac` 的 `ECON` 键对应。

| KEY | VALUE | 说明 |
|-----|-------|------|
| 1 | `HKD` | 港币 |
| 2 | `USD` | 美元 |
| 3 | `CNY` | 人民币 |

---

## ENUM_MKT_RISK.csv

**用途**：市场风险类型枚举，作为 `PCR_MKT_RISK` 向量变量的风险类型维度，也与 `MKT_RISK_CORR_INTUP` / `MKT_RISK_CORR_INTDN` 矩阵的行列标签对应，用于 `PCR_TOTAL` 相关系数聚合。

| KEY | VALUE | DESCRIPTION |
|-----|-------|-------------|
| 1 | `INT` | Interest Rate（利率风险） |
| 2 | `SPD` | Credit Spread（信用利差风险） |
| 3 | `EQ` | Equity（股票风险） |
| 4 | `PROP` | Property（房地产风险） |
| 5 | `FX` | Currency（外汇风险） |

---

## ENUM_SCENARIOS_INT.csv

**用途**：利率情景枚举，作为 `RF_ECON_SCENARIO` 向量变量的情景维度，与 `RF_Curves.fac` 的 `SCENARIO` 键对应。情景选择由 `mp.INT_BITING` 字段驱动（`INT_UP` 或 `INT_DN`）。

| KEY | VALUE | 说明 |
|-----|-------|------|
| 1 | `BASE` | 基准利率情景 |
| 2 | `INT_UP` | 利率上行压力情景 |
| 3 | `INT_DN` | 利率下行压力情景 |

---

## ENUM_TEST_1.csv / ENUM_TEST_2.csv

**用途**：测试用枚举，仅包含 `AC` 和 `IG_Tradable` 两个值，供 `TEST_ARRAY` 和 `TEST_ARRAY_TXT` 变量在开发阶段验证多维向量功能，不参与正式计算。

---

# 3. 矩阵（matrices/）

所有矩阵存放于 `input/matrices/`，格式为 `.csv`。

**格式约定**：第一列为行标签（与列标签对应的维度值），首行为列标签，左上角首格为 `MATRIX_LIFE`（矩阵标识符）。通过 `formulae.ArrayAggregateByCorr(ctx, i, "MATRIX_NAME", vector)` 调用，计算公式为：

$$\text{result} = \sqrt{\mathbf{v}^\top \mathbf{C} \, \mathbf{v}}$$

其中 $\mathbf{v}$ 为输入向量（各分量的 PCR 或配置权重），$\mathbf{C}$ 为相关系数矩阵。

---

## ASSET_CORR_MATRIX.csv

**用途**：资产类别间的协方差矩阵，用于 `PORT_RISK`（组合风险）计算。
**维度**：23×23，行列标签均为 `ASSET_CLASS` 枚举值。

**调用公式**：`PORT_RISK = ArrayAggregateByCorr(ctx, i, "ASSET_CORR_MATRIX", ASSET_MIX)`

**关键数值特征**：

| 资产对 | 近似值 | 说明 |
|--------|--------|------|
| `HY` × `HY` | 0.0085 | 高收益债自身方差 |
| `DM` × `DM` | 0.0245 | 发达市场股票自身方差 |
| `EM` × `EM` | 0.0541 | 新兴市场股票自身方差 |
| `QIS` × `QIS` | 0.0869 | QIS 自身方差（最高） |
| `DM` × `EM` | 0.0276 | 发达/新兴市场强正相关 |
| `EM` × `QIS` | 0.0686 | 新兴市场与 QIS 高度相关 |
| `AC` / `IG_Tradable` | 0.0 | 全行全列为零（占位） |
| `Dummy_2` ~ `Dummy_7` | 0.0 | 全行全列为零（占位） |
| `Cash` / `Deposit_ST_Govern_Bond` / `Seed_Capital` / `Other_Asset` | 0.0 | 全行全列为零 |
| `Loan_Private_Credit` = `Loan_Private_Credit_New` | 完全相同 | 两行/两列数值一致 |

> 矩阵当前采用**协方差矩阵**而非标准化相关系数矩阵（对角元素非 1），各分量由 `ASSET_MIX` 权重向量驱动。

---

## MKT_RISK_CORR_INTUP.csv

**用途**：利率**上行**情景下的市场风险相关系数矩阵，用于 `PCR_TOTAL` 聚合。当 `mp.INT_BITING = "INT_UP"` 时使用。
**维度**：5×5，行列标签为 `ENUM_MKT_RISK`（INT / SPD / EQ / PROP / FX）。

|      | INT  | SPD  | EQ   | PROP | FX   |
|------|------|------|------|------|------|
| INT  | 1    | 0    | 0    | 0    | 0.25 |
| SPD  | 0    | 1    | 0.75 | 0.5  | 0.25 |
| EQ   | 0    | 0.75 | 1    | 0.5  | 0.25 |
| PROP | 0    | 0.5  | 0.5  | 1    | 0.25 |
| FX   | 0.25 | 0.25 | 0.25 | 0.25 | 1    |

> **设计逻辑**：利率上行时债券价值下跌（INT 风险主导），与股票、利差、房地产风险方向相反——INT 与 SPD/EQ/PROP 相关性设为 0，体现分散化效益。FX 与所有风险保持 0.25 的低度正相关。

---

## MKT_RISK_CORR_INTDN.csv

**用途**：利率**下行**情景下的市场风险相关系数矩阵，用于 `PCR_TOTAL` 聚合。当 `mp.INT_BITING = "INT_DN"` 时使用。
**维度**：5×5，行列标签同上。

|      | INT  | SPD  | EQ   | PROP | FX   |
|------|------|------|------|------|------|
| INT  | 1    | 0.5  | 0.5  | 0.25 | 0.25 |
| SPD  | 0.5  | 1    | 0.75 | 0.5  | 0.25 |
| EQ   | 0.5  | 0.75 | 1    | 0.5  | 0.25 |
| PROP | 0.25 | 0.5  | 0.5  | 1    | 0.25 |
| FX   | 0.25 | 0.25 | 0.25 | 0.25 | 1    |

> **设计逻辑**：利率下行时各风险来源趋于同向运动（"risk-off"环境），INT 与 SPD/EQ 相关性分别为 0.5，整体矩阵相关系数更高，PCR_TOTAL 聚合后惩罚更大，反映下行情景下的风险集中效应。

---

# 4. 变量伪代码

按 `config/products/RiskCalc.csv` 中的 `SecondaryPriority` 顺序排列（测试变量已剔除）。

**符号约定**：
- `Scalar :: Num` — 标量数值
- `Scalar :: Txt` — 标量文本
- `Array[d1, d2] :: Num` — 数值数组，下标维度为 d1, d2
- `Param` — 由 Param 表读入的标量参数（随时间步变化）
- `i` — 当前时间步索引
- `sim` — 当前模拟情景编号
- `mp.*` — 模型点字段
- `Table(name, col, key...)` — 从假设表按键读取

---

## 1. CALENDAR_MTH　`Scalar :: Num`

```
startMth = param("startMonth")
CALENDAR_MTH[i] = ((startMth - 1 + i) mod 12) + 1
```

---

## 2. CALENDAR_YR　`Scalar :: Num`

```
startYear  = param("startYear")
startMonth = param("startMonth")
totalMonths = startMonth - 1 + i
CALENDAR_YR[i] = startYear + floor(totalMonths / 12)
```

---

## 3. CALENDAR_DATE　`Scalar :: Num`

```
CALENDAR_DATE[i] = CALENDAR_YR[i] * 100 + CALENDAR_MTH[i]
```

---

## 4. RC_FAC_HKD　`Scalar :: Num`　`Param`

```
RC_FAC_HKD[i] = Table("Param", col=i, "RC_FAC_HKD")
```

---

## 5. RC_FAC_USD　`Scalar :: Num`　`Param`

```
RC_FAC_USD[i] = Table("Param", col=i, "RC_FAC_USD")
```

---

## 6. RC_FAC_CNY　`Scalar :: Num`　`Param`

```
RC_FAC_CNY[i] = Table("Param", col=i, "RC_FAC_CNY")
```

---

## 7. ASSET_MIX　`Array[ASSET_CLASS] :: Num`

```
for each class:
    ASSET_MIX[i, class] = Table("SAA_Sims", col="Y", sim, class)
```

> 按模拟情景 `sim` 从 SAA_Sims 表读取各资产类别的配置比例。

---

## 8. TRAD_BOND_DIST　`Array[ENUM_BOND_RATING, ENUM_BOND_TERM] :: Num`

```
for each (rating, term):
    TRAD_BOND_DIST[i, rating, term] =
        Table("AssetTradBondDistByRatingByTerm", col="Y", rating, term)
```

---

## 9. ALT_BOND_DIST　`Array[ENUM_BOND_RATING, ENUM_BOND_TERM] :: Num`

```
for each (rating, term):
    ALT_BOND_DIST[i, rating, term] =
        Table("AssetAltBondDistByRatingByTerm", col="Y", rating, term)
```

---

## 10. TRAD_BOND_DUR　`Array[ENUM_BOND_TERM] :: Num`

```
for each term:
    TRAD_BOND_DUR[i, term] = Table("AssetTradBondAssump", col="Y", term, "DURATION")
```

---

## 11. ALT_BOND_DUR　`Array[ENUM_BOND_TERM] :: Num`

```
for each term:
    ALT_BOND_DUR[i, term] = Table("AssetAltBondAssump", col="Y", term, "DURATION")
```

---

## 12. TRAD_BOND_MIX_BY_TERM　`Array[ENUM_BOND_TERM] :: Num`

```
for each term:
    TRAD_BOND_MIX_BY_TERM[i, term] = Σ TRAD_BOND_DIST[i, *, term]
                                       (sum over all ENUM_BOND_RATING)
```

---

## 13. ALT_BOND_MIX_BY_TERM　`Array[ENUM_BOND_TERM] :: Num`

```
for each term:
    ALT_BOND_MIX_BY_TERM[i, term] = Σ ALT_BOND_DIST[i, *, term]
                                      (sum over all ENUM_BOND_RATING)
```

---

## 14. STRESS_BOND_SPD_PARAMS　`Array[ENUM_BOND_RATING, ENUM_BOND_TERM] :: Num`

```
for each (rating, term):
    STRESS_BOND_SPD_PARAMS[i, rating, term] =
        Table("StressBondSpdByRatingByTerm", col="Y", rating, term)
```

---

## 15. ASSET_BAL　`Array[ASSET_CLASS] :: Num`

```
for each class:
    ASSET_BAL[i, class] = mp.TOT_ASSET * ASSET_MIX[i, class]
```

---

## 16. NAR_TOT　`Array[ASSET_CLASS] :: Num`

```
for each class:
    NAR_TOT[i, class] = ASSET_BAL[i, class] * Table("Assump", "Y", class, "NAR_TOT")
```

---

## 17. NAR_FI　`Array[ASSET_CLASS] :: Num`

```
for each class:
    NAR_FI[i, class] = ASSET_BAL[i, class] * Table("Assump", "Y", class, "NAR_FI")
```

---

## 18. NAR_EQ　`Array[ASSET_CLASS] :: Num`

```
for each class:
    NAR_EQ[i, class] = ASSET_BAL[i, class] * Table("Assump", "Y", class, "NAR_EQ")
```

---

## 19. NAR_PROP　`Array[ASSET_CLASS] :: Num`

```
for each class:
    NAR_PROP[i, class] = ASSET_BAL[i, class] * Table("Assump", "Y", class, "NAR_PROP")
```

---

## 20. BOND_TRAD　`Scalar :: Num`

```
BOND_TRAD[i] = Σ NAR_FI[i, class]
               for class in {AC, IG_Tradable, HY, Securitized_CMBS}
```

---

## 21. BOND_ALT　`Scalar :: Num`

```
BOND_ALT[i] = Σ NAR_FI[i, class]
              for class in {Loan_Private_Credit}
```

---

## 22. PIVOT_RC_SPD_BOND_TRAD　`Array[ENUM_BOND_RATING, ENUM_BOND_TERM] :: Num`

```
for each (rating, term):
    PIVOT_RC_SPD_BOND_TRAD[i, rating, term] =
        BOND_TRAD[i] * TRAD_BOND_DIST[i, rating, term] * STRESS_BOND_SPD_PARAMS[i, rating, term]
```

---

## 23. PIVOT_RC_SPD_BOND_ALT　`Array[ENUM_BOND_RATING, ENUM_BOND_TERM] :: Num`

```
for each (rating, term):
    PIVOT_RC_SPD_BOND_ALT[i, rating, term] =
        BOND_ALT[i] * ALT_BOND_DIST[i, rating, term] * STRESS_BOND_SPD_PARAMS[i, rating, term]
```

---

## 24. PIVOT_RC_SPD_BOND　`Array[ENUM_BOND_RATING, ENUM_BOND_TERM] :: Num`

```
for each (rating, term):
    PIVOT_RC_SPD_BOND[i, rating, term] =
        PIVOT_RC_SPD_BOND_TRAD[i, rating, term] + PIVOT_RC_SPD_BOND_ALT[i, rating, term]
```

---

## 25. TOT_NAR　`Array[ASSET_CLASS] :: Num`

```
for each class:
    TOT_NAR[i, class] = NAR_FI[i, class] + NAR_EQ[i, class] + NAR_PROP[i, class]
```

---

## 26. BOND_DUR　`Scalar :: Num`

```
if BOND_TRAD[i] + BOND_ALT[i] == 0:
    BOND_DUR[i] = 0
else:
    durBondTrad = Σ TRAD_BOND_MIX_BY_TERM[i, t] * TRAD_BOND_DUR[i, t]   (SumProduct over term)
    durBondAlt  = Σ ALT_BOND_MIX_BY_TERM[i, t]  * ALT_BOND_DUR[i, t]    (SumProduct over term)
    BOND_DUR[i] = (BOND_TRAD[i] * durBondTrad + BOND_ALT[i] * durBondAlt)
                  / (BOND_TRAD[i] + BOND_ALT[i])
```

---

## 27. BOND_DUR_INT　`Scalar :: Num`

```
BOND_DUR_INT[i] = round(BOND_DUR[i])      // banker's rounding (RoundToEven)
```

---

## 28. RF_ECON_SCENARIO　`Array[ENUM_ECON, ENUM_SCENARIOS_INT] :: Num`

```
for each (econ, scenario):
    term = BOND_DUR_INT[i]
    RF_ECON_SCENARIO[i, econ, scenario] = Table("RF_Curves", "Y", econ, scenario, Text(term))
```

---

## 29. RC_ASSET_EQ　`Array[ASSET_CLASS] :: Num`

```
for each class:
    RC_ASSET_EQ[i, class] = NAR_EQ[i, class] * Table("Assump", "Y", class, "RC_FAC_EQ")
```

---

## 30. RC_ASSET_PROP　`Array[ASSET_CLASS] :: Num`

```
for each class:
    RC_ASSET_PROP[i, class] = NAR_PROP[i, class] * Table("Assump", "Y", class, "RC_FAC_PROP")
```

---

## 31. ASSET_RETURN_RATE　`Array[ASSET_CLASS] :: Num`

```
for each class:
    ASSET_RETURN_RATE[i, class] = Table("Assump", "Y", class, "RET_RATE")
```

---

## 32. ASSET_SD_RATE　`Array[ASSET_CLASS] :: Num`

```
for each class:
    ASSET_SD_RATE[i, class] = Table("Assump", "Y", class, "SD_RATE")
```

---

## 33. ASSET_HKD_MIX　`Array[ASSET_CLASS] :: Num`

```
for each class:
    ASSET_HKD_MIX[i, class] = Table("Assump", "Y", class, "ASSET_HKD_MIX")
```

---

## 34. ASSET_USD_MIX　`Array[ASSET_CLASS] :: Num`

```
for each class:
    ASSET_USD_MIX[i, class] = Table("Assump", "Y", class, "ASSET_USD_MIX")
```

---

## 35. NAR_TOT_HKD　`Array[ASSET_CLASS] :: Num`

```
for each class:
    NAR_TOT_HKD[i, class] = NAR_TOT[i, class] * ASSET_HKD_MIX[i, class]
```

---

## 36. NAR_TOT_USD　`Array[ASSET_CLASS] :: Num`

```
for each class:
    NAR_TOT_USD[i, class] = NAR_TOT[i, class] * ASSET_USD_MIX[i, class]
```

---

## 37. NAR_FI_HKD　`Array[ASSET_CLASS] :: Num`

```
for each class:
    NAR_FI_HKD[i, class] = NAR_FI[i, class] * ASSET_HKD_MIX[i, class]
```

---

## 38. NAR_FI_USD　`Array[ASSET_CLASS] :: Num`

```
for each class:
    NAR_FI_USD[i, class] = NAR_FI[i, class] * ASSET_USD_MIX[i, class]
```

---

## 39. SUM_NAR_FI_HKD　`Scalar :: Num`

```
SUM_NAR_FI_HKD[i] = Σ NAR_FI_HKD[i, class]   (sum over ASSET_CLASS)
```

---

## 40. SUM_NAR_FI_USD　`Scalar :: Num`

```
SUM_NAR_FI_USD[i] = Σ NAR_FI_USD[i, class]   (sum over ASSET_CLASS)
```

---

## 41. HKD_BOND_MV_CHG　`Scalar :: Num`

```
intBitingScen = mp.INT_BITING
term          = BOND_DUR_INT[i]

HKD_BOND_MV_CHG[i] =
    (Table("RF_Curves", "Y", "HKD", intBitingScen, Text(term))
   - Table("RF_Curves", "Y", "HKD", "BASE",        Text(term)))
    * BOND_DUR[i]
    * SUM_NAR_FI_HKD[i]
```

---

## 42. USD_BOND_MV_CHG　`Scalar :: Num`

```
intBitingScen = mp.INT_BITING
term          = BOND_DUR_INT[i]

USD_BOND_MV_CHG[i] =
    (Table("RF_Curves", "Y", "USD", intBitingScen, Text(term))
   - Table("RF_Curves", "Y", "USD", "BASE",        Text(term)))
    * (-BOND_DUR[i])
    * SUM_NAR_FI_USD[i]
```

---

## 43. PORT_BOND_MV　`Scalar :: Num`

```
PORT_BOND_MV[i] = SUM_NAR_FI_HKD[i] + SUM_NAR_FI_USD[i]
```

---

## 44. INT_RISK_SCAL　`Scalar :: Num`

```
if PORT_BOND_MV[i] == 0 or BOND_DUR[i] == 0:
    INT_RISK_SCAL[i] = 0
else:
    INT_RISK_SCAL[i] = (mp.TOT_LIAB * mp.LIAB_DUR_PCR)
                       / (PORT_BOND_MV[i] * BOND_DUR[i])  - 1
```

---

## 45. PORT_BOND_MV_CHG　`Scalar :: Num`

```
PORT_BOND_MV_CHG[i] = HKD_BOND_MV_CHG[i] + USD_BOND_MV_CHG[i]
```

---

## 46. PORT_NAR_HKD　`Scalar :: Num`

```
PORT_NAR_HKD[i] = Σ NAR_TOT_HKD[i, class]   (sum over ASSET_CLASS)
```

---

## 47. PORT_NAR_USD　`Scalar :: Num`

```
PORT_NAR_USD[i] = Σ NAR_TOT_USD[i, class]   (sum over ASSET_CLASS)
```

---

## 48. PORT_RETURN_RATE　`Scalar :: Num`

```
PORT_RETURN_RATE[i] = Σ ASSET_MIX[i, class] * ASSET_RETURN_RATE[i, class]
                       (SumProduct over ASSET_CLASS)
```

---

## 49. PORT_RISK　`Scalar :: Num`

```
PORT_RISK[i] = sqrt( ASSET_MIX[i]ᵀ × CorrMatrix("ASSET_CORR_MATRIX") × ASSET_MIX[i] )
               // ArrayAggregateByCorr：加权相关矩阵聚合
```

---

## 50. PCR_FX　`Scalar :: Num`

```
PCR_FX[i] = PORT_NAR_HKD[i] * RC_FAC_HKD[i]
           + PORT_NAR_USD[i] * RC_FAC_USD[i]
```

---

## 51. PCR_EQ　`Scalar :: Num`

```
PCR_EQ[i] = Σ RC_ASSET_EQ[i, class]   (sum over ASSET_CLASS)
```

---

## 52. PCR_PROP　`Scalar :: Num`

```
PCR_PROP[i] = Σ RC_ASSET_PROP[i, class]   (sum over ASSET_CLASS)
```

---

## 53. PCR_SPD　`Scalar :: Num`

```
PCR_SPD[i] = -Σ PIVOT_RC_SPD_BOND[i, rating, term]
               (sum over ENUM_BOND_RATING × ENUM_BOND_TERM)
```

---

## 54. PCR_INT　`Scalar :: Num`

```
PCR_INT[i] = PORT_BOND_MV_CHG[i] * INT_RISK_SCAL[i]
```

---

## 55. PCR_MKT_RISK　`Array[ENUM_MKT_RISK] :: Num`

```
for each mktRisk:
    PCR_MKT_RISK[i, mktRisk] =
        "INT"  → PCR_INT[i]
        "SPD"  → PCR_SPD[i]
        "EQ"   → PCR_EQ[i]
        "PROP" → PCR_PROP[i]
        "FX"   → PCR_FX[i]
        else   → 0
```

---

## 56. PCR_TOTAL　`Scalar :: Num`

```
intBitingScen = mp.INT_BITING

if intBitingScen == "INT_UP":
    PCR_TOTAL[i] = ArrayAggregateByCorr("MKT_RISK_CORR_INTUP", PCR_MKT_RISK[i])
elif intBitingScen == "INT_DN":
    PCR_TOTAL[i] = ArrayAggregateByCorr("MKT_RISK_CORR_INTDN", PCR_MKT_RISK[i])
```

---

## 附：辅助函数（udf.go）

| 函数 | 说明 |
|------|------|
| `MULT(n, m)` | `n mod m == 0` |
| `Text(x)` | 数值/字符串转字符串 |
| `MonthlyRate(r)` | 年利率转月利率：`(1 + r/100)^(1/12) - 1` |
| `round4(x)` | 四舍五入保留 4 位小数 |
| `tIdx(sY, sM, cY, cM)` | 起始年月到当前年月的月数：`(cY-sY)*12 + (cM-sM)` |
| `Idx(dims, k)` | 安全取 dims 第 k 个元素（越界返回 0） |

