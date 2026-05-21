# ZCB / SPD 月度插值方法

> 实现文件：`fin/interp.go`、`app/modelALM/lib/formula_zcb.go`、`app/modelALM/lib/formula_spd.go`  
> 最后更新：2026-05

---

## 背景

ESG（经济情景生成器）输出的期限结构曲线是**年级稀疏样本**，典型 term 节点为：

```
1, 2, 3, 4, 5, 7, 10, 15, 20, 30, 50, 75, 100  （年）
```

精算投影以**月**为步长，需要每个整数月（1 月、2 月……1200 月）对应的值。  
本文档描述将年级样本扩展为月度值的两种插值方法：

| 曲线类型 | 插值方法 | 函数 |
|---|---|---|
| ZCB 折现因子（`ZCB:PRICE`） | 几何加权（形状函数） | `fin.GeometricInterpolateMonthly` |
| 信用利差（`SPD_PC:{n}`） | 线性 | `fin.LinearInterpolateMonthly` |

---

## ZCB — 几何加权插值

### 输入与输出

| 输入 | 来源 | 说明 |
|---|---|---|
| ZCB 价格样本 | ESD 表（`ZCB:PRICE`） | 每条 sim、每个 economy 的稀疏年级 ZCB 价格 |
| `baseFrate` | ECON 表（`ZCB_BASE_FRATE`） | 连续年化基准利率，每个 economy/entity 一个 |

| 输出 | 格式 | 说明 |
|---|---|---|
| `df[m-1]` | `[]float64`，长度 1200 | 第 m 月的 ZCB 折现因子（m = 1..1200） |

### 前置定义

$$
v = e^{-r}  \qquad \text{（年度折现因子，} r = \text{baseFrate）}
$$

$$
v_M = e^{-r/12}  \qquad \text{（月度折现因子）}
$$

注：$v_M^{12} = v$，保证月度与年度自洽。

### 形状函数

插值的核心是**形状函数** $f(n)$：

$$
f(n) = \frac{1}{1 - v^n}
$$

其中 $n$ 为**样本区间的跨度**（年）。

**直觉**：$f(n)$ 等价于"$n$ 年等额年金因子的倒数"，在利率曲线形状符合几何级数递减时，提供内生一致的权重。

### 两层插值

**层 1：年级插值**（整数年 $n = 1 \ldots N_{\max}$）

$$
\text{shape}_0 = \frac{1}{1 - v^d}, \quad
\alpha = \text{shape}_0 \cdot \bigl(1 - v_M^{k}\bigr), \quad
k = (n - T_{lo}) \times 12
$$
$$
P(n) = P(T_{lo}) + \alpha \times \bigl[P(T_{hi}) - P(T_{lo})\bigr]
$$

边界：$P(0)=1.0$（`anchor0`）；左/右端平坦外延。

**层 2：月级插值**（$m = 12n + k$，$k \in \{1,\ldots,11\}$）

$$
\text{shape}_0^{\text{yr}} = \frac{1}{1 - v}, \quad
\alpha = \text{shape}_0^{\text{yr}} \cdot \bigl(1 - v_M^k\bigr)
$$
$$
\mathrm{df}[m-1] = P(n) + \alpha \times \bigl[P(n+1) - P(n)\bigr]
$$

调用方式：`fin.GeometricInterpolateMonthly(terms, prices, baseFrate, 1200, 1.0, false)`

---

## SPD — 线性插值

### 输入与输出

| 输入 | 来源 | 说明 |
|---|---|---|
| 利差样本 | ESD 表（`SPD_PC:{n}`） | CLASS=`SPD_PC`，MEASURE=spreadband 编号（"1","2",…） |

| 输出 | 格式 | 说明 |
|---|---|---|
| `spd[m-1]` | `[]float64`，长度 1200 | 第 m 月的年化利差（m = 1..1200） |

### 两层线性插值

**层 1（年级）**：

$$
\alpha = \frac{n - T_{lo}}{T_{hi} - T_{lo}}, \quad
S(n) = S(T_{lo}) + \alpha \times \bigl[S(T_{hi}) - S(T_{lo})\bigr]
$$

**层 2（月级）**：

$$
\alpha = \frac{k}{12}, \quad
\mathrm{spd}[m-1] = S(n) + \alpha \times \bigl[S(n+1) - S(n)\bigr]
$$

`flatYear0=true`：$m < 12$ 月平坦取第 1 年利差水平（`S(0) = S(1)`）。

调用方式：`fin.LinearInterpolateMonthly(terms, vals, 1200, 0.0, true)`

---

## 性质对比

| 性质 | ZCB 几何插值 | SPD 线性插值 |
|---|---|---|
| 样本点精确通过 | ✓ | ✓ |
| 需要 baseFrate | ✓ | — |
| 符合折现几何结构 | ✓ | — |
| 实现简单 | — | ✓ |
| 适用曲线类型 | 折现因子（乘法空间） | 利差（加法空间） |

---

## 代码结构

```
fin/interp.go
  GeometricInterpolateMonthly(terms, vals, baseFrate, maxMonths, anchor0, flatYear0) []float64
    ├── 层1：形状函数加权 → yearP[n]
    └── 层2：形状函数加权月级插值

  LinearInterpolateMonthly(terms, vals, maxMonths, anchor0, flatYear0) []float64
    ├── 层1：线性 → yearP[n]
    └── 层2：k/12 月级插值

app/modelALM/lib/formula_zcb.go  → 调用 GeometricInterpolateMonthly
app/modelALM/lib/formula_spd.go  → 调用 LinearInterpolateMonthly
```

### 触发方式

`RiskCalc.csv` 中优先级顺序：

| 优先级 | 公式 | 说明 |
|---|---|---|
| 61 | `CALC_ESD_DATA` | ESD 按投影步 T 刷新快照 |
| 62 | `CALC_ZCB_MONTHLY` | 每 T 重算 ZCB 月度 DF |
| 63 | `CALC_SPD_MONTHLY` | 每 T 重算 SPD 月度利差 |

### Trace 开关（`diagnostics/diagnostics.yaml`）

```yaml
zcbTrace:
  enabled: 1
  econs: USD_HKPLG    # 逗号分隔白名单；空 = 全部
spdTrace:
  enabled: 1
  econs: USD_HKPLG
  spdBands: 1,2       # 逗号分隔 spreadBand 白名单；空 = 全部
```

---

## 与其他方案的对比

| 方案 | 优点 | 缺点 |
|---|---|---|
| **几何加权（本方案，ZCB）** | 与折现率内生一致；样本点精确通过 | 需要提供 `baseFrate` |
| **线性插值（本方案，SPD）** | 实现最简单；利差加法空间直觉好 | 样本点间曲率为零 |
| 对数线性（收益率空间线性） | 金融直觉好 | 需先转利率再插值，步骤较多 |
| 三次样条 | 光滑性最好 | 可能产生非单调或负价格；对极端情景不稳健 |

