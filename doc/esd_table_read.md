# ESD 大表读取设计文档

## 1. 背景

ESG（Economic Scenario Generator）输出的 ESD 表包含所有 simulation 的利率曲线数据。
生产环境下 simulation 可达数千条，ESD 表规模：

```
总行数 = K_sims × K_eco × K_term × K_class
       = 2000   ×  5   ×  10   ×  1    ≈ 100,000 行
列数   ≈ 50（月步 13 列 + 年步至投影末尾）
```

每个 `ProjContext` 对应一个 sim，只需读取其中 `K_eco × K_term ≈ 50` 行。

---

## 2. 表格结构

```
!6, SIMULATION, ECONOMY, CLASS, MEASURE, TERM, <yyyymm>, ...
*, 1, HKD_HKPAR, ZCB,    PRICE,   1,  0.9921, 0.9843, ...
*, 1, HKD_HKPAR, ZCB,    PRICE,   5,  0.9612, 0.9431, ...
*, 1, HKD_HKPAR, SPD,    PC,      1,  0.0023, 0.0025, ...
*, 1, HKD_HKPAR, EQUITY, RET_IDX, 0,  1.0000, 1.0031, ...
*, 1, HKD_HKPAR, EQUITY, RNY_PC,  0,  0.0521, 0.0519, ...
*, 1, HKD_HKPAR, VALN,   DEF,     0,  0.0012, 0.0013, ...
*, 2, HKD_HKPAR, ZCB,    PRICE,   1,  0.9918, 0.9836, ...   ← sim 2 紧跟 sim 1
...
```

- `numIdx = 6`：行键由 5 段组成，最后一段（TERM/key5）对 ZCB/SPD 为年期整数，对 EQUITY/VALN 为占位符（如 "0"）
- 行键格式：`"sim:economy:class:measure:key5"`
- **行按 sim 升序生成，同一 sim 内字段集合对所有 sim 完全一致**（生产约定）
- 物理存储：`cache/v2.Table`，`data map[string][]byte`，O(1) 哈希查找

### 支持的 CLASS:MEASURE 类型

| CLASS | MEASURE | key5 | 说明 |
|-------|---------|------|------|
| ZCB | PRICE | 年期（1,5,10,…） | 零息债券价格曲线 |
| SPD | PC | 年期（1,5,10,…） | 利差曲线 |
| EQUITY | RET_IDX | 占位符（如 "0"） | 权益收益指数 |
| EQUITY | RNY_PC | 占位符 | 风险中性收益率 |
| VALN | DEF | 占位符 | 违约估值参数 |

---

## 3. 读取流程

### 3.1 总览

```
程序启动
    └── LoadTable("esd.fac")
            └── data map[rowKey][]byte  ← 全表常驻内存

首个 sim 触发 initZCBStore
    └── zcbGetTemplate(RowKeys, tbl)
            └── 扫描 RowKeys，遇到第一个 sim 的 ZCB:PRICE 行收集模板
            └── sim 变化时立即 break  ← 只扫 K_eco×K_term 行，O(K)
            └── 模板存入 zcbTemplateCache（sync.Map，key=tbl指针）

后续每个 sim 触发 initZCBStore
    └── zcbGetTemplate → 直接命中 cache，O(1)
    └── 对模板中每条 (economy, class, measure, term)
            └── 构造 rowKey = sim + ":" + economy + ":" + ... 
            └── tbl.RawRow(...)  → data[rowKey]，O(1) 哈希查找
            └── parseRowSeries  → 一遍逗号扫描，O(cols)
    └── 存入 ZCBStore.prices[economy][term] []float64
```

### 3.2 模板发现：`esdGetTemplate`

**前提利用**：行按 sim 升序 + 所有 sim 字段相同。

```
扫描 RowKeys（有序切片）
    │
    ├─ 无 CLASS:MEASURE 过滤——ZCB/SPD/EQUITY/VALN 一并捕获
    │
    ├─ 记录 firstSim = 第一个行的 sim 字段
    │
    ├─ 收集 (economy, class, measure, key5) → tmpl[]
    │
    └─ 遇到 sim ≠ firstSim → break
```

代价：**O(K_rows_per_sim)**（所有 CLASS:MEASURE 行数之和），而非 O(N_total)。

### 3.3 行数据读取：`parseRowSeries`

对模板中每条记录，直接构造完整行键后调用 `RawRow`：

```
key    = "sim:economy:class:measure:key5"
raw    = tbl.data[key]              // O(1) map lookup
series = 逗号扫描 raw → []float64   // O(cols)，单遍，无重复扫描
```

内部存储键去掉 sim：`"economy:class:measure:key5"`，跨步复用。

### 3.4 投影步映射：`colIndex`

ESD 列按月/年混合步长，投影步 `t` 始终按月。`colIndex[t]` 用 hold-last 规则将每个 `t` 映射到最近的 ESD 列：

```
colIndex 构建（O(m+n) 单遍，m=ESD列数，n=投影步数）：

ji = 0  // ESD列游标，只向右移动
for t, tp in timepoints:
    while esdCols[ji+1] <= tp: ji++
    colIndex[t] = ji
```

### 3.5 快照机制：`ESDStore.stepTo`

`CALC_ESD_DATA`（StatefulFormula，原 `CALC_ZCB_CURVES`）在每个投影步 `t` 执行一次，
遍历 `series` map 的所有 key，将 `series[k][colIndex[t]]` 写入 `Snapshot[k]`。
后续所有类型化访问函数（`ZCBSnapshotPrice`、`SPDSnapshotPrice` 等）在同一步内 **O(1)** 读取。

### 3.6 类型化访问函数

```go
ZCBSnapshotPrice(ctx, economy, term)        // ZCB:PRICE，term=整数
SPDSnapshotPrice(ctx, economy, term)        // SPD:PC，   term=整数
EquityRetIdx    (ctx, economy, key5)        // EQUITY:RET_IDX
EquityRNYPC     (ctx, economy, key5)        // EQUITY:RNY_PC
ValnDef         (ctx, economy, key5)        // VALN:DEF
ESDSnapshot     (ctx, economy, class, measure, key5)  // 通用
```

---

## 4. 复杂度对比

| 方案 | 模板/索引建立 | 每 sim 初始化 | 2000 sims 合计 |
|------|-------------|-------------|---------------|
| 原始（全表扫描） | — | O(N_total) = O(100k) | **2亿次**字符串比较 |
| 共享索引（v1） | O(N_total) × 1次 | O(K) + O(1) lookup | O(100k) + O(2000×K) |
| **模板直查（当前）** | **O(K)** × 1次 | **O(K)** RawRow | **≈10万次**，提速 ~2000× |

`K = K_eco × K_term`（例：5 × 10 = 50）

---

## 5. 并发安全

- `zcbTemplateCache`：`sync.Map`，`LoadOrStore` 保证并发首次构建时只有一份模板生效
- `ZCBStore` 存入 `ctx.Cache`（per-ProjContext），不跨 sim 共享，无锁
- `stepTo` 仅在引擎保证单线程的 StatefulFormula 上下文中调用

---

## 6. 约束与假设

| 假设 | 影响 | 违反时的行为 |
|------|------|------------|
| RowKeys 按 sim 升序 | 模板只扫第一个 sim | 模板可能不完整（缺字段）→ 部分 sim 数据为 0 |
| 所有 sim 字段集合相同 | 模板可跨 sim 复用 | 同上 |
| CLASS=ZCB, MEASURE=PRICE 为目标行 | 过滤其他行 | 其他类型数据被忽略（设计如此） |

如 ESD 生成方式改变（如字段因情景而异），需退回 `zcbSimIndex`（全量索引）方案。

---

## 7. 存储格式讨论：`map[string][]byte` 是否最优？

### 当前格式

`cache/v2.Table` 将每行存储为逗号拼接的原始字节：

```
data["1:HKD_HKPAR:ZCB:PRICE:5"] = []byte("0.9921,0.9843,0.9756,...")
```

每次 `parseRowSeries` 需要：

1. **哈希计算**：对 rowKey 字符串做哈希（O(len(key))）
2. **bytes → float64**：每列调用 `strconv.ParseFloat`，约 50–100 ns/次
3. **内存分散**：100k 个独立 `[]byte` 切片在堆上随机分布，CPU cache miss 率高

对当前规模（2000 sims × 50 rows × 50 cols = 5M 次 ParseFloat ≈ 0.5s），尚可接受。

### 更优格式选项

| 格式 | 加载代价 | 每次读取 | 适用场景 |
|------|---------|---------|---------|
| `map[string][]byte`（当前） | 低 | O(cols) ParseFloat | 通用，文本+数字混合表 |
| `map[string][]float64` | 一次性全量解析 | O(1) 直接索引 | **纯数字表（ESD）最优** |
| 行主序平铺 `[]float64` | 一次性 | O(1)，连续内存 | 超大表，cache 友好 |
| 按 sim 预组织 `ESDCube` | 全量解析+重组 | O(1)，零重建 | ESD 专用，整 run 共享 |

### 建议

- **当前规模（≤50 万行）**：现有方案够用，template+RawRow 已是瓶颈之外
- **超大表（>50 万行或 sim>5000）**：在 `cache/v2.Table` 加可选预解析：
  ```go
  // LoadTable 后调用，将 data 中纯数字表一次性解析为 []float64
  func (t *Table) PreparseNumeric()
  ```
  消除 per-sim `strconv.ParseFloat`，提速约 2–5×

---

## 8. 相关文件

| 文件 | 说明 |
|------|------|
| `lib/formula_esg.go` | 本文档描述的读取逻辑全部实现 |
| `kit/cache/v2/table.go` | `Table.RawRow`、`Table.RowKeys` 定义 |
| `input/tables/ECON_202512.fac` | ESD 表格示例输入文件 |
| `lib/formula_bonds.go` | `ZCBSnapshotPrice` 的消费方 |






