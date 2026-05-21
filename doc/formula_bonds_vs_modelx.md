# formula_bonds (Go/formulae) vs modelx — 客观评价

对比对象：`als/app/modelALM/lib/formula_bonds.go` 中实现的债券组合估值引擎
（基于 `formulae` + `fin` 库），与 lifelib/modelx（Python，cell+space 图模型）。
评价维度：**计算效率**、**易用性**、**可扩展性**。

---

## 1. 计算效率

| 维度 | formula_bonds (本库) | modelx |
|---|---|---|
| 语言/运行时 | Go，AOT 编译，原生 float64 | Python，解释执行（依赖 numpy 时局部加速）|
| 单只债券生命周期 CF | `BuildBondCF` 一次性构造切片，运行时只读 | 每个 cell 按 t 惰性求值，逐 t 计算 |
| ABV/MV 折现 | `fin.PvFlat` 内循环：切片遍历 + FMA + df 累乘，**无 Pow、无函数调用、无日历算术** | cell 递归：每个 t 触发 graph 遍历 + memoization 字典查找 |
| 折现因子 | `amortDF` 预计算（init 时一次 Pow），热路径零 Pow | 每次表达式计算可能重新 `(1+r)**(-1/12)` |
| 组合层聚合 | 单 pass `for _, b := range bonds` + 单一 `BondMetrics` 结构体写入 | 每个 cell 独立缓存（dict），跨 cell 聚合需多次遍历 |
| 重复读取代价 | `BOND_*` 公式 = O(1) 读 `bp.Metrics` 字段 | cell 命中 cache 仍走字典 + 装箱/拆箱 |
| 内存占用 | 每只债券一条 cfs 切片（连续内存）；组合只保留当期快照 | 每个 (cell, t) 存一项，长期空间随 T 线性增长 |
| 大规模组合 | 数千~数万只债券、月度全期，单核毫秒级 | 同规模 modelx 通常需要秒级 |

**结论**：本库在热路径上是 **数量级优于 modelx**（典型 10×~100×），核心来自
（a）AOT + 切片连续内存、（b）一次性构造 cfs 后只做线性扫描、
（c）`evaluateAt` 单 pass 同时算 CF/ABV/MV 且原地刷新 `Metrics`。
modelx 的优势是 numpy 矢量化时段较多，但债券逐月窗口推进型逻辑天然不适合矢量化。

---

## 2. 易用性

| 维度 | formula_bonds | modelx |
|---|---|---|
| 学习曲线 | 需要理解 `formulae` 注册机制（`RegisterStatefulFormula`、Batch Asc/Desc）和 cfs/pos/EntryT 时间锚点约定 | 直接在 Excel 风格的 cell 里写公式，所见即所得 |
| 公式编写 | Go 函数，强类型、需重编译；可被 IDE 跳转/重构 | Python 表达式，REPL 友好，热改 |
| 调试 | go test、断点、`go run`，但单只债券调试要追切片下标 | modelx 提供 cell 追溯（依赖图、t 切片）很直观 |
| 错误诊断 | `solveAmortRate` 失败时打印完整债券诊断；其它逻辑错误靠 panic+stack | cell 内异常自动定位到表达式 |
| 写一条新指标（如 BOND_DURATION） | 加一行 `RegisterScalarNum`，从 `bondStep` 读快照即可 | 加一个 cell，写表达式即可 |
| 配置/数据 | aBonds 表 + Assembly 表（Batch 顺序约束）；建模师需懂"CALC_BOND_PORTFOLIO 必须排在所有 BOND_* 之前" | space/cell 隐式依赖图，引擎自己排序 |
| 用户群 | Go 开发者 / 工程化精算 | 精算师友好（接近 Excel 心智模型）|

**结论**：modelx **对精算师更易用**（接近 Excel、所见即所得、依赖关系自动）；
本库 **对工程化交付更易用**（强类型、可重构、单元测试、CI/CD、二进制部署、无 Python 环境依赖）。
本库的"批次/顺序约束"是性能的代价，需要文档/校验保护。

---

## 3. 可扩展性

| 维度 | formula_bonds | modelx |
|---|---|---|
| 新增资产类型（股票、贷款、衍生品）| 复制 `ExistingBond/BondPortfolio` 模式：一个 struct + `evaluateAt` + 一组 `RegisterScalarNum` | 新建 space + cells |
| 与负债/资本/利率联动 | `formulae` 引擎统一调度，负债公式可直接 `bondStep(ctx).CF` 读取 | space 间引用 |
| 接入 ESG / 利率曲线 | `mv()` 内目前是 dummy 5%，预留 `ctx` 参数；改成 `ctx.Cache["esg"].DF(econ, term)` 即可 | cell 引用利率 cell |
| 多场景/随机情景 | `ProjContext` 已是单情景容器；外层并发跑 N 个 ctx 即可线性扩展（goroutine + 共享只读 cfs） | modelx 多场景靠 Space 复制，内存压力大；并行度受 GIL 限制 |
| 新增时间频率（季度/年度）| 框架 t 仍是月度；`evaluateAt` 幂等，跳步调用即可（窗口自然合并）| cell 重定义粒度 |
| 中途新增/卖出资产 | `EntryT` 锚点已设计好；`bonds = append(bp.bonds, newBond)` 即可，无需 padding | 较繁琐 |
| 重构成本 | Go 强类型 + IDE，跨文件改动安全 | Python 动态、cell 串接易留隐性引用 |
| 公式版本化 / DSL 输出 | `formulae` 是注册器，未来可把公式拓扑序列化、生成审计文档 | modelx 也支持但更偏交互式 |

**结论**：本库 **横向扩展（资产种类、并发场景）和工程化演进（重构、CI、文档化）显著更强**；
modelx 在 **快速原型 / 精算师手工探索** 上更顺手。

---

## 4. 总结

| | 效率 | 易用性 | 可扩展性 |
|---|---|---|---|
| **formula_bonds** | ★★★★★（AOT + 切片 + O(1) 快照）| ★★★（需懂 Go 与 Batch 顺序）| ★★★★★（强类型、并发、生产级）|
| **modelx** | ★★（解释执行）| ★★★★★（精算师友好）| ★★★（原型快、规模化吃力）|

**适用建议**：
- 生产环境 / 大组合 / 高频跑批 / 工程化交付 → 本库。
- 精算师快速探索 / 模型原型 / 监管沟通用最小可执行示例 → modelx。
- 长期路线：用 modelx 做需求/原型，用本库做生产实现，两者通过 aBonds 等输入表 + 校验测试对齐。

