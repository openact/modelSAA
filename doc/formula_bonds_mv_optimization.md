# formula_bonds.go — MV 折现性能优化思路

> 状态：**已关闭 — 维持现状**  
> 结论：ZCB/SPD 曲线每步由 ESG 刷新，per-step 计算不可避免；MktSpreadPC 若每只债券各异则步内去重也无收益。当前实现已是正确且合理的写法，无需改动。  
> 背景：当前 `mv()` 的 general path（`MktSpreadPC ≠ 0`）每步每债券需要对全部剩余月份各调用一次 `math.Pow`，是唯一的热路径非零成本。

---

## 现状分析

```
fast path (MktSpreadPC == 0)
  → pvCurve(future, rdf_grid)
  → O(N) 纯乘加，零 Pow   ✓ 已优化

general path (MktSpreadPC ≠ 0)
  → for k in [0, lim):
      grossSpread = 1 + (spds[k] + MktSpreadPC) / 100
      pv += cf[k] * zcbDFs[k] / math.Pow(grossSpread, (k+1)/12.0)
  → 每月一次 Pow（≈ exp+log），约贵 20–50× 于乘法
```

`MktSpreadPC` 是每只债券独有的"噪音利差"，不能放入共享 grid，因此 general path 无法完全避免 per-month 计算。

### 根本约束（优化上界）

**`ZCB_CURVES` 和 `SPD_CURVES` 每个时步 t 都不同（来自 ESG 随机情景）。**

这意味着：
- 即使两只债券的 `MktSpreadPC` 完全相同，每个计算周期仍必须重新计算折现因子切片；
- 不存在跨步复用的可能，**per-step 计算不可避免**；
- 唯一可优化的维度是：**同一步内**，多只具有完全相同 `(Econ, SpreadBand, MktSpreadPC)` 的债券可共享一次计算。

所有跨步缓存（包括方案 A 的预计算切片）均因此**失效**。

---

## 优化方案

### 方案 A：预计算 per-bond MV 折现因子切片 ~~（推荐）~~ ❌ 无效

**原理**

`MktSpreadPC` 是 init 时读入的**静态常量**，整个投影期不变。  
看似可在 `loadExistingBond` 时一次性完成：

```
mvDFs[m-1] = zcbDFs_t0[m-1] / (1 + (spd_t0[m-1] + MktSpreadPC) / 100) ^ (m/12)
```

**为何无效**

`ZCB_CURVES` 和 `SPD_CURVES` 每步均由 ESG 刷新，t=0 的曲线在 t=1 已失效。  
预计算切片锁定了 t=0 的曲线，**MV 将无法反映后续利率/利差变化，数值错误**。  

→ **结论：方案 A 不可用于 ESG 随机投影，废弃。**

---

### 方案 B：Running-product 递推（适用条件受限）

**原理**

若 `spds[k]` 在整条曲线上为**常数**（flat spread），则月度因子 `df_m = (1+spd/100)^(1/12)` 可预计算，凑出递推：

```
累积折现因子[0] = df_m
累积折现因子[k] = 累积折现因子[k-1] × df_m
```

每月只需一次乘法，消除 Pow。

**局限**

`SPD_CURVES` 是 term-structure（每个期限利差不同），曲线不 flat，无法直接递推。  
仅在 spread band 对应的利差约为 flat 时才近似成立。

---

### 方案 C：每步按 (Econ, SpreadBand, MktSpreadPC) 组合缓存（**当前唯一有效的优化**）

**原理**

ZCB/SPD 曲线每步必须重算，无法跨步复用。  
但**同一步内**，若多只债券的 `(Econ, SpreadBand, MktSpreadPC)` 完全相同，  
它们的折现因子切片也完全相同——可共享计算，只调用一次 Pow 序列。

在 `evaluateAt` 开始时建立 step-local cache：

```
key = econ + ":" + spreadBand + ":" + strconv.FormatFloat(mktSpreadPC, 'f', 4, 64)
stepCache[key] = []float64{df[0], df[1], ...}  // 同组只算一次
```

之后每只债券调用 `pvCurve(future, stepCache[key])`，退化为纯乘加。

**收益取决于重复率**

| 场景 | 重复率 | 效果 |
|---|---|---|
| `MktSpreadPC` 每只债券唯一 | 0% | 无收益，还增加 map 开销 |
| `MktSpreadPC` 按评级分 5 档 | ~高 | 每档只算一次，效果显著 |
| 所有债券同一档 | 100% | 整个组合只算一次 Pow 序列 |

**代价**

- 每步需分配/释放 step-local map（可用 `sync.Pool` 复用）
- 实现复杂度中等

---

### 方案 D：将 MktSpreadPC 纳入 RDF grid（彻底消除 general path）

**原理**

`MktSpreadPC` 按 SpreadBand 分档时，可在 `CALC_RDF_MONTHLY` 中直接生成  
`RDF_CURVES[econ:spreadBand]` = ZCB × (1 + (band_spd + mktSpread_for_band) / 100)^(-m/12)  

每只债券直接走 fast path，general path 永不触发。

**局限**

- 每只债券 `MktSpreadPC` 独立时无法分档，方案不适用
- 需要对"每个 SpreadBand 有一个代表性 MktSpreadPC"的业务假设

---

## 结论：维持现状

综合以上分析：

1. **ZCB/SPD 曲线每步变动**（ESG 随机情景）→ 跨步缓存完全无效
2. **MktSpreadPC 可能每只债券各异** → 步内去重（方案 C）收益不确定
3. **方案 C 引入的 map 分配/复用复杂度** 在收益不明确时不值得

**当前 `mv()` 的 general path 已是该问题在正确性约束下的最优写法。** 不做改动。

若未来 profile 发现 general path 确实是瓶颈（占总运行时间 >10%），再重开此议题，按方案 C 实施并量化收益。

---

## 推荐实施路径（已关闭）

~~维持现状。如需重开，参见上方"结论"一节。~~

---

## 相关文件

- `lib/formula_bonds.go` — `mv()` general path（line ~107）
- `lib/formula_rdf.go` — `RDFFill`，ZCB + SPD → RDF grid
- `doc/formula_bonds_roadmap.md`

