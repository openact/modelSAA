# modelSAA — 运维手册

> 本手册说明如何配置和运行 modelSAA 模型。所有操作仅涉及配置文件与输入数据，无需修改代码。技术细节（变量公式、输入表结构）见 `modelDoc.md`。

---

## 目录

1. [目录结构总览](#1-目录结构总览)
2. [运行模型](#2-运行模型)
3. [config/ — 模型级配置](#3-config--模型级配置)
4. [input/ — 运行级配置](#4-input--运行级配置)
5. [input/mpfs/ — 模型点文件](#5-inputmpfs--模型点文件)
6. [输出说明](#6-输出说明)
7. [常见操作指引](#7-常见操作指引)

---

# 1. 目录结构总览

```
modelSAA/
├── main.go                         # 入口（只读，无需修改）
│
├── config/                         # 模型级静态配置（不随 run 变化）
│   ├── config.yaml                 # 路径映射
│   ├── structures/
│   │   └── StructureALM.csv        # 产品结构定义
│   ├── products/
│   │   └── RiskCalc.csv            # 变量列表及计算顺序
│   └── variablesets/               # 输出变量集定义
│       ├── ALM.yaml                # 全量变量集
│       ├── ALM_KRI.yaml            # KRI 汇总集
│       ├── ALM_KRI_1.yaml          # KRI 子集 1
│       └── ALM_KRI_2.yaml          # KRI 子集 2
│
├── input/                          # 每次运行可替换的输入数据 ← 日常操作区
│   ├── config.yaml                 # 输入路径配置
│   ├── enums/                      # 枚举维度定义（.csv）
│   ├── matrices/                   # 相关系数矩阵（.csv）
│   ├── mpfs/                       # 模型点文件（.csv）
│   ├── tables/                     # 假设表（.fac）        ← 替换假设数据
│   └── runsettings/                # run 级参数配置        ← 配置 run
│       ├── runsettings.yaml        # run 清单             ← 增删 run
│       ├── filenames/              # 逻辑表名 → 文件名映射 ← 切换假设版本
│       ├── parameters/             # 全局数值参数          ← 调整日期等参数
│       ├── variablesets/           # 各 run 的输出变量集选择
│       ├── filelocations/          # 数据文件目录映射
│       ├── diagnostics/            # 诊断配置（当前默认关闭）
│       ├── options/                # 分组、跨情景统计与输出时点
│       └── switches/               # （预留）
│
└── doc/
    ├── modelDoc.md
    └── modelOperationalManual.md
```

---

# 2. 运行模型

## 2.1 CLI 参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-config-dir` | `config` | `config/` 目录路径 |
| `-input-dir` | `input` | `input/` 目录路径 |
| `-output-dir` | `output` | 结果输出目录（自动创建） |
| `-config-file` | — | 指定单个配置文件路径（可选） |
| `-input-file` | — | 指定单个输入文件路径（可选） |
| `-output-file` | — | 指定单个输出文件路径（可选） |
| `-task-file` | — | 任务定义文件（可选） |

> `WorkDir` 自动创建于 `output-dir/.work/`，无需指定。

**典型调用**：

```powershell
# 使用默认路径（config/ input/ output/ 均在 exe 同目录下）
.\modelSAA.exe

# 指定自定义路径
.\modelSAA.exe -config-dir .\config -input-dir .\input -output-dir C:\appData\modelSAA\outputs\v21
```

## 2.2 执行流程（概览）

```
读取 config/ 与 input/runsettings/runsettings.yaml
    └─ 逐 run 执行:
           加载假设表、枚举、矩阵、模型点
           并发预测（每个 simulation 独立运行）
           └─ 写逐模型点结果（storedVariables，可选）
              按 groupBy 分别汇总产品结果（aggregatedVariables）
              对 statistics 选中的组件/视图计算跨情景 average/min/max
              分组与统计文件仅导出 storedVariables 中的汇总变量
              全部成功后发布 run 级 metadata 与 completion
```

---

# 3. config/ — 模型级配置

> 通常**不需要修改**。仅在新增产品结构或调整输出路径时变更。

## 3.1 config/config.yaml

```yaml
confpaths:
  structureDir: structures        # StructureALM.csv 所在子目录
  productDir: products            # RiskCalc.csv 所在子目录
  accumulationDir: accumulations
  variableSetDir: variablesets    # ALM.yaml 等变量集所在子目录

outputpaths:
  logDir: logs
  logFile: ALM.log
  resultDir: results
```

`StructureALM.csv` 使用列 `Name,Kind,Factory,Description,PreloadFrom,PreloadField,Priority`。
当前只有 `ALM` 产品，工厂为 `RiskCalc`，没有 preload 或 accumulation 组件。
`ACCUM` 输出是该产品的独立分组视图，不代表另一个组件。

## 3.2 config/variablesets/ — 输出变量集

控制各 run 可选择的输出变量范围，与 `input/runsettings/variablesets/` 配合使用。

| 文件 | 包含变量 | 典型用途 |
|------|---------|---------|
| `ALM.yaml` | 全部 63 个输出变量 | 开发调试、完整输出 |
| `ALM_KRI.yaml` | `PORT_RETURN_RATE`, `PORT_RISK` | 组合风险收益指标 |
| `ALM_KRI_1.yaml` | `PORT_RETURN_RATE`, `PORT_RISK` | 组合风险收益指标 |
| `ALM_KRI_2.yaml` | `PCR_TOTAL` | PCR 资本指标 |

---

# 4. input/ — 运行级配置

## 4.1 input/config.yaml

```yaml
inputpaths:
  runsettingDir: runsettings
  runsettingFile: runsettings.yaml
  matrixDir: matrices             # 自动全量预加载
  enumDir: enums                  # 自动全量预加载
```

> `matrices/` 和 `enums/` 下所有文件在启动时自动加载，无需在 filenames 中注册。

## 4.2 input/runsettings/runsettings.yaml — run 清单

每个 run 是一次完整的预测执行单元，程序按顺序逐一运行。

```yaml
runs:
  RUN_001:
    name: RUN_001
    structure: StructureALM       # 引用 config/structures/StructureALM.csv 中的 Name
    simulations: 1~4              # 模拟情景范围（闭区间，对应 SAA_Sims.fac 的行）
    variableSets:
      - default                   # 引用 runsettings/variablesets/default.yaml
    parameters:
      - default                   # 引用 runsettings/parameters/default.yaml
    options:
      - default                   # 引用 runsettings/options/default.yaml
    diagnostics:
      - default                   # 引用 runsettings/diagnostics/default.yaml（默认关闭）
    filenames:
      - default                   # 引用 runsettings/filenames/default.yaml
    fileLocations:
      - default                   # 引用 runsettings/filelocations/default.yaml
    seriatimResults: true         # true = 同时输出每个情景、每个模型点的明细
```

**当前 run 清单**：

| Run | Structure | Simulations | VariableSet | 输出变量 |
|-----|-----------|-------------|-------------|---------|
| `RUN_001` | StructureALM | 1~4 | `default` | 全量（ALM）存储，KRI 汇总 |

`run2.yaml` 是可选的变量集配置，当前没有启用名为 `RUN_002` 的 run。

## 4.3 input/runsettings/filenames/default.yaml — 表文件映射

将公式代码中使用的**逻辑表名**映射到 `input/tables/` 下的**物理文件名**（不含扩展名）。

```yaml
globals:
  filenames:
    Param:                           Param
    Assump:                          Assump
    AssetAltBondAssump:              AssetAltBondAssump
    AssetAltBondDistByRatingByTerm:  AssetAltBondDistByRatingByTerm
    AssetTradBondAssump:             AssetTradBondAssump
    AssetTradBondDistByRatingByTermForIntPCR: AssetTradBondDistByRatingByTermForIntPCR
    AssetTradBondDistByRatingByTermForCsPCR: AssetTradBondDistByRatingByTermForCsPCR
    RF_Curves:                       RF_Curves
    SAA_Sims:                        SAA_Sims
    StressBondSpdByRatingByTerm:     StressBondSpdByRatingByTerm
```

**逻辑表名与对应变量的关系**：

| 逻辑表名 | 用于变量 | 内容 |
|---------|---------|------|
| `Assump` | `ASSET_RETURN_RATE`, `ASSET_SD_RATE`, `ASSET_MSCI_MIX`, `NAR_*`, `RC_ASSET_*` 等 | 资产假设 |
| `Param` | `RC_FAC_HKD`, `RC_FAC_USD`, `RC_FAC_CNY`, `RC_FAC_MSCI` | 时间步级风险资本因子 |
| `SAA_Sims` | `ASSET_MIX` | SAA 权重（按情景） |
| `RF_Curves` | `RF_ECON_SCENARIO`, `HKD_BOND_MV_CHG`, `USD_BOND_MV_CHG` | 无风险利率曲线 |
| `AssetTradBondAssump` | `TRAD_BOND_DUR` | 传统债券久期假设 |
| `AssetAltBondAssump` | `ALT_BOND_DUR` | 另类债券久期假设 |
| `AssetTradBondDistByRatingByTermForIntPCR` | `TRAD_BOND_DIST_FOR_INT_PCR` | 传统债券评级期限分布（利率 PCR） |
| `AssetTradBondDistByRatingByTermForCsPCR` | `TRAD_BOND_DIST_FOR_CS_PCR` | 传统债券评级期限分布（信用利差 PCR） |
| `AssetAltBondDistByRatingByTerm` | `ALT_BOND_DIST` | 另类债券评级期限分布 |
| `StressBondSpdByRatingByTerm` | `STRESS_BOND_SPD_PARAMS` | 利差压力参数 |

## 4.4 input/runsettings/parameters/default.yaml — 全局参数

```yaml
globals:
  parameters:
    startYear:      2024    # 预测起始年
    startMonth:     12      # 预测起始月
    fiscalYearEnd:  12      # 财年结束月
    lenProjYears:   1       # 预测年数
```

| 参数 | 用于变量 | 说明 |
|------|---------|------|
| `startYear` | `CALENDAR_YR` | 预测时间轴起点年 |
| `startMonth` | `CALENDAR_MTH`, `CALENDAR_YR` | 预测时间轴起点月 |
| `fiscalYearEnd` | （预留） | 财年末月 |
| `lenProjYears` | （预留） | 全局预测年数上限，单点预测月数以 `PROJ_TERM_M` 为准 |

## 4.5 input/runsettings/variablesets/ — run 的输出变量选择

| 键 | 说明 |
|----|------|
| `calculatedVariables` | 需要计算的变量集（必须覆盖输出变量的完整依赖链） |
| `storedVariables` | 导出变量范围；逐模型点文件可包含全量，分组/统计文件只导出已汇总的变量 |
| `aggregatedVariables` | 在模型点之间按 groupBy 加总，并供跨情景统计使用的数值变量 |

**default.yaml**（RUN_001）：

```yaml
globals:
  variableSets:
    calculatedVariables: [ALM]          # 计算全部变量
    storedVariables:     [ALM]          # 存储全部变量
    aggregatedVariables: [ALM_KRI, ALM_KRI_2] # 汇总 PORT_RETURN_RATE / PORT_RISK / PCR_TOTAL
```

**run2.yaml**（可选，当前未启用）：

```yaml
globals:
  variableSets:
    calculatedVariables: [ALM]                  # 仍计算全部（保证依赖链完整）
    storedVariables:     [ALM_KRI_1, ALM_KRI_2] # 仅存储精简 KRI
    aggregatedVariables: [ALM_KRI_1, ALM_KRI_2]
```

---

## 4.6 input/runsettings/options/ — 分组和统计选择

```yaml
globals:
  options:
    outputStep: "true"
    groupBy: [SP_CODE, ACCUM]
    statistics:
      - component: ALM
        views: [SP_CODE]
```

`groupBy` 中每个字段形成独立视图，不是字段组合。默认每个情景分别按
`SP_CODE` 和 `ACCUM` 汇总 `ALM` 的 `PORT_RETURN_RATE`、`PORT_RISK`、
`PCR_TOTAL`；跨四个情景只对 `ALM/SP_CODE` 计算 average/min/max。
统计为各时点分别计算，不跨时间求和。多个模型点分组时，即使是收益率、
风险率等指标，也按引擎的数值加总规则处理，不自动加权或重新计算。

可用 MP 中任意 `txt`/`int` 字段替换分组字段；`num` 字段不能作为分组键。
默认值来自 `globals.options`，run 顶层的 `groupBy`、`statistics` 可分别覆盖；
显式 `groupBy: []` 禁用产品分组，显式 `statistics: []` 禁用跨情景统计。
禁用分组时也必须禁用或调整引用这些视图的统计选择。

# 5. input/mpfs/ — 模型点文件

文件名（不含扩展名）须与 `config/structures/StructureALM.csv` 中的 `Name` 字段对应。`.csv` 格式，**第二行为字段类型声明行**（`txt` / `int` / `num`），不属于数据行。旧 `core` 类型不再支持；分组与 preload 只接受 `txt`、`int`，`num` 明确报错。

## 5.1 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `ID` | `int` | 模型点唯一标识，与情景编号独立；每个情景均运行这些模型点 |
| `PLAN_CODE` | `txt` | 产品代码（如 `FUND01`） |
| `SP_CODE` | `txt` | 子产品/分组代码 |
| `ACCUM` | `txt` | 普通业务分组字段，当前值为 `FUND`，不是保留字段或 accumulation 规则 |
| `PROJ_TERM_M` | `int` | 该模型点的预测月数 |
| `TOT_ASSET` | `num` | 资产总额 |
| `TOT_LIAB` | `num` | 负债总额（用于 `INT_RISK_SCAL`） |
| `LIAB_DUR_PCR` | `num` | 负债 PCR 久期（用于 `INT_RISK_SCAL`） |
| `LIAB_DUR_RW` | `num` | 负债 RW 久期（预留） |
| `LIAB_COST_PC` | `num` | 负债成本率（预留） |
| `LAC_EQ_FAC` | `num` | 权益 LAC 因子 |
| `LAC_EQ_AMT` | `num` | 权益 LAC 金额 |
| `LAC_CS_FAC` | `num` | 信用利差 LAC 因子 |
| `LAC_CS_AMT` | `num` | 信用利差 LAC 金额 |
| `INT_BITING` | `txt` | 利率 biting 方向：`INT_UP` 或 `INT_DN` |

## 5.2 当前文件

| 文件 | 对应 Structure | 模型点数 | 内容 |
|------|---------------|---------|------|
| `ALM.csv` | `ALM` | 1 | FUND01，`INT_DN` |

---

# 6. 输出说明

## 6.1 目录结构

```
<output-dir>/
└── results/
  └── <RUN_NAME>/
    ├── .log/
    │   └── ALM.log        # 结构化日志（debug 级别）
    ├── ALM_by_SP_CODE.fac
    ├── ALM_by_ACCUM.fac
    ├── ALM_by_SP_CODE_average.fac
    ├── stats/
    │   ├── ALM_by_SP_CODE_min.fac
    │   └── ALM_by_SP_CODE_max.fac
    ├── metadata/
    │   ├── metadata.yaml
    │   └── completion.yaml
    └── seriatim/          # seriatimResults = true 时生成四个逐模型点 FAC
```

## 6.2 seriatimResults

`seriatimResults: true` 时，额外输出每个 simulation × 每个模型点的 FAC，
文件名为 `ALM-<ID>-<PLAN_CODE>-sp<SP_CODE>-sim<SIM_ID>.fac`，
行键为 SIM_ID、NAME、MP_ID、PLAN_CODE、SP_CODE、VAR_NAME。
变量范围由 `storedVariables` 决定，时点由 `outputStep` 决定。
设为 `false` 只跳过逐 MP 文件，不禁用分组和已选统计。

普通分组结果的行键为 SIM_ID、NAME、分组字段、VAR_NAME；
统计结果的行键为 NAME、分组字段、VAR_NAME，**不含 SIM_ID**。
文本仅保留在逐 MP 结果，不参与分组加总或跨情景统计。
分组字段缺失、为空或类型为 `num` 时明确报错。

所有分组/统计共用 run 根目录下的 `metadata\metadata.yaml`，
catalog 按 component/view/group/statistic 标识结果，并记录相对于 run 的文件路径。
只有整个 run 输出成功后才发布 `metadata\completion.yaml`；
输出失败由 `RunProjection` 返回入口并非零退出。不要用旧版叶目录 metadata 或
历史 FAC 判断本次运行成功，读取方应以本次 run catalog 和 completion 为准。
旧版 resReader 已停用，resParser 尚未适配此输出协议。

---

# 7. 常见操作指引

## 7.1 新增一个 run

1. 在 `input/runsettings/runsettings.yaml` 中追加新的 run entry。
2. 按需新建以下配置文件并在 run 中引用（不改的字段可直接复用 `default`）：
   - `parameters/<name>.yaml` — 调整日期或预测年数
   - `filenames/<name>.yaml` — 切换假设表版本
   - `variablesets/<name>.yaml` — 调整输出范围

## 7.2 切换假设数据版本（如换一套利率曲线）

1. 将新的 `.fac` 文件（如 `RF_Curves_v2.fac`）放入 `input/tables/`。
2. 新建 `input/runsettings/filenames/scenario_v2.yaml`，只需覆盖变化的表名：
   ```yaml
   globals:
     filenames:
       RF_Curves: RF_Curves_v2
   ```
3. 在 `runsettings.yaml` 中为该 run 的 `filenames` 字段引用 `scenario_v2`（多个 filenames 按顺序合并，后者覆盖前者）。

## 7.3 调整预测起始日期

修改 `input/runsettings/parameters/default.yaml` 中的 `startYear` 和 `startMonth`。

## 7.4 扩展模拟情景数量

1. 在 `input/tables/SAA_Sims.fac` 中添加新情景行（行键为新的 simulation 编号）。
2. 情景与模型点独立；不需要为每个新情景新增模型点。只有新增业务模型点时才修改 `input/mpfs/ALM.csv`。
3. 在 `runsettings.yaml` 中将 `simulations` 范围扩大，如 `1~10`。

## 7.5 精简输出（减少结果文件体积）

- 将 `variablesets/<name>.yaml` 中的 `storedVariables` 改为精简集合（如 `ALM_KRI`）。
- 将 `seriatimResults` 设为 `false` 可跳过逐模型点明细输出，仍保留逐情景分组结果。

## 7.6 更新模型点数据

直接编辑 `input/mpfs/ALM.csv`，保持字段顺序和类型声明行（第二行）不变。常见更新：
- 调整 `TOT_ASSET` / `TOT_LIAB`
- 修改 `LIAB_DUR_PCR`（负债久期变化时）
- 切换 `INT_BITING`（`INT_UP` ↔ `INT_DN`）
