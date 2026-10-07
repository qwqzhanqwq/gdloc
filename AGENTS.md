# AGENTS.md — gdloc

> gdloc：一个专注于 Godot 4 项目的命令行代码行数统计工具，用 Go 编写。
> 本文件是所有 AI agent 在本仓库工作时必须遵守的约束。开始任何任务前先完整读一遍。

---

## 1. 项目目标与边界

**做什么**
- 统计 Godot 4 项目中的代码、注释、空行，输出到命令行。
- 理解 Godot 的项目结构：`project.godot`、`addons/`、`.tscn`/`.tres` 中的内嵌代码、`.gdignore`。
- 给独立开发者有用的信息，而不是给公司估算成本。

**不做什么**
- 不支持 Godot 3（不处理 `.shader`、Godot 3 的 tscn 格式）。
- 不做 COCOMO 或任何成本/人力估算。
- 不追求极致性能：几万行项目体感瞬间完成即可，不要为性能引入并发或复杂优化，除非用户要求。
- 不修改、不写入被扫描的项目中的任何文件。本工具对被扫描目录是**只读**的。
- 不联网。

---

## 2. 技术约束

- 语言：Go，版本与本机 `go version` 一致，`go.mod` 中声明。
- **优先只用标准库**：命令行参数用 `flag`，表格用 `text/tabwriter`，JSON 用 `encoding/json`。
- 引入任何第三方依赖前**必须先询问用户**并说明理由。目前唯一已认可的依赖：`github.com/sabhiram/go-gitignore`（解析 `.gitignore`，阶段 3 起使用）。
- 编译产物为单个可执行文件，主要运行环境是 **Windows**（PowerShell 和 Git Bash 都要能用），同时保持跨平台。
- 路径处理一律用 `path/filepath`；输出给用户看的相对路径统一用 `/` 分隔。
- 文件读取：按 UTF-8 处理，去掉开头的 BOM，同时兼容 `\n` 和 `\r\n`，最后一行没有换行符也要计入。
- 单个文件读取失败：输出警告到 stderr 并继续，不中断整个统计。

---

## 2.1 发布

- 模块路径为 `github.com/qwqzhanqwq/gdloc`，内部包一律用完整路径导入。
- 发布由 GoReleaser 完成：推送 `v*` tag 触发 `.github/workflows/release.yml`，产出 Windows / macOS / Linux 的 amd64 与 arm64 版本。Windows 包必须是 zip（Scoop、winget 只接受 zip）。
- 版本号通过 `-ldflags "-X main.version=..."` 注入；`go install ...@vX.Y.Z` 安装的版本从模块信息读取。不要在代码里手写版本号。
- 打 tag、推送、创建 GitHub 仓库或 token、配置 Actions secrets 都需要用户亲自操作或明确同意，agent 不自行执行。
- Scoop：manifest 由 GoReleaser 的 `scoops` 推送到 `qwqzhanqwq/scoop-bucket` 的 `bucket/` 目录，凭证为 gdloc 仓库的 Actions secret `SCOOP_BUCKET_TOKEN`（只授权 scoop-bucket 的 Contents 读写）。不要手动编辑 bucket 里的 manifest。
- 包管理器渠道按顺序推进：GitHub Release + `go install` → Scoop bucket → Homebrew tap → winget。未经用户确认不提前配置后续渠道。
- 修改发布配置后，本机装有 goreleaser 时运行 `goreleaser check` 和 `goreleaser release --snapshot --clean` 验证；没有装就在报告中说明未验证。

---

## 3. 项目结构

```
gdloc/
├── AGENTS.md
├── README.md                 # 英文（默认）：使用说明 + 完整计数规则（与代码行为保持一致）
├── README.zh-CN.md           # 简体中文版，内容与 README.md 一一对应
├── LICENSE                   # MIT
├── go.mod                    # 模块路径 github.com/qwqzhanqwq/gdloc
├── .goreleaser.yaml          # 发布配置（GoReleaser v2）
├── .github/workflows/        # 推送 v* tag 时自动发布
├── cmd/gdloc/                # 入口：参数解析、调用各模块、退出码
├── internal/scan/            # 目录遍历、排除规则、按文件类型分类
├── internal/counter/         # 各语言的逐行计数器（GDScript、Shader、C#）+ 公共结果类型
├── internal/godot/           # Godot 专属解析：project.godot、plugin.cfg、tscn/tres 内嵌代码提取
├── internal/report/          # 结果汇总、排序、表格输出、JSON 输出
├── internal/history/         # 阶段 8：读取 git 历史，按天/周统计新增与删除
└── testdata/                 # 测试用的样例文件和期望结果
```

- 模块之间单向依赖：`cmd` → `report`/`scan`/`godot`/`counter`/`history`；`history` 只依赖 `counter`/`godot`/`scan`；`counter` 不依赖其他内部包。
- 计数器只接收"文本内容"，不关心文件来自磁盘还是从 tscn 里提取出来的。这样内嵌代码可以复用同一个计数器。
- 单个源文件尽量不超过 300 行，超过就拆分。

---

## 4. 计数规则（核心，必须严格遵守）

### 4.1 通用的单行判定

每一行只归入一类，优先级如下：
1. 只含空白字符 → **空行**（即使位于多行注释内部）。
2. 去掉注释后仍有非空白内容 → **代码行**（行尾带注释的代码行算代码）。
3. 其余 → **注释行**。

### 4.2 GDScript（`.gd`）

- 注释以 `#` 开头，到行尾结束。
- 行首（忽略缩进）为 `##` 的注释行 → 额外计为**文档注释**（文档注释是注释的子集）。
- `#region` / `#endregion` 行计为注释。
- 字符串内的 `#` 不是注释。需要识别的字符串形式：
  - `"..."`、`'...'`，含转义 `\"`、`\'`、`\\`
  - `"""..."""`、`'''...'''` 多行字符串
  - 前缀形式：`&"..."`（StringName）、`^"..."`（NodePath）、`r"..."`（原始字符串，内部反斜杠不转义）
- 多行字符串（包括用作"文档字符串"的 `"""`）的每一行计为**代码行**，但内部纯空白行仍按 4.1 计为**空行**。

### 4.3 Godot Shader（`.gdshader`、`.gdshaderinc`）

- 单行注释 `//`，多行注释 `/* ... */`（不嵌套）。
- `/** ... */` 是文档注释：其覆盖的注释行同时计入 **Comments** 和 **Doc**；空的 `/**/` 不算文档注释。
- 预处理指令（`#include`、`#define`、`#ifdef` 等）是**代码行**，不是注释。
- 字符串 `"..."` 内的 `//`、`/*` 不触发注释判定；字符串除 `#include` 外也出现在 `hint_enum("A", "B")` 等 uniform hint 中。

### 4.4 C#（`.cs`，阶段 7，可选）

- `//`、`/* */`，`///` 计为文档注释。
- 需处理普通字符串、`@"..."` 逐字字符串、`$"..."` 插值字符串、`"""..."""` 原始字符串。

### 4.5 场景与资源（`.tscn`、`.tres`）

- 文件本身只统计**总行数**，分别列为 "Scene" 和 "Resource"，**不计入代码总量**。
- 内嵌代码（阶段 4）：
  - `[sub_resource type="GDScript" ...]` 中的 `script/source = "..."` → 计入 "GDScript (embedded)"。
  - `[sub_resource type="Shader" ...]` 中的 `code = "..."` → 计入 "Shader (embedded)"。
  - `[gd_resource type="Shader" ...]` 的 `[resource]` 段中的 `code = "..."` → 计入 "Shader (embedded)"（主资源为 Shader 的 `.tres`；Godot 不支持把 GDScript 保存为 `.tres` 主资源，故无对应情况）。
  - 先把字符串值完整取出并反转义，再交给对应计数器。
  - 字符串可能跨越多个物理行，也可能使用 `\n` 转义，两种情况都要处理。
  - **实现前必须先用用户真实项目里的 tscn/tres 样本确认格式**，把样本放进 `testdata/`，不要凭猜测实现。

### 4.6 不统计的内容

- `VisualShader` 资源只计数量，不计行数。
- `.import`、`.uid`、二进制资源（图片、音频、模型、`.res`、`.scn`）不统计。

### 4.7 历史增量统计（`--daily` / `--weekly`，阶段 8）

**数据来源**
- 只读取 git 历史。通过 `os/exec` 调用系统 `git`，不引入 go-git 等第三方库。
- 只允许只读命令（`log`、`diff-tree`、`diff`、`cat-file --batch`、`rev-parse` 等），且全部加 `--no-optional-locks`。禁止 `checkout`、`stash`、`reset` 等任何会改动工作区、索引或引用的命令。
- 扫描根目录不在 git 仓库内，或系统找不到 `git` → 错误信息输出到 stderr，退出码 2。

**统计口径**
- 范围：HEAD 可达的**非合并提交**，每个提交与其**第一个父提交**比较；根提交与空树比较。合并提交跳过，不计入 Commits。
- 日期：使用**作者日期**（author date），按**本地时区**分组。按提交逐个归入日期桶，因此历史中日期不单调也不影响结果。
- 周：ISO 周，**周一开始**。
- 新增/删除的分类：对文件改动前后两个版本分别用 4.2 / 4.3 的计数器做逐行分类，再按 `git diff -U0`（或等价的逐行 diff）给出的行号，取新增行在新版本中的分类、删除行在旧版本中的分类，分别累计 Code / Comments / Blanks（Doc 不单列）。这样多行字符串、块注释等跨行结构的判定与总量统计完全一致。
- 净变化 = 新增 − 删除。
- Commits 列：该时段内、至少改动了一个"扫描根下且未被排除的 `.gd` / `.gdshader` / `.gdshaderinc` / `.tscn` / `.tres`"文件的非合并提交数。
- 时段末的 Code 总量：以 HEAD 树按同一口径统计的代码总量为基准，减去作者日期晚于该时段结束的提交的净变化得到（不包含未提交改动）。
- 内嵌代码：`.tscn` / `.tres` 改动时，对前后两个版本分别提取内嵌块（同 4.5），按块 id 配对后同样做逐行 diff；新增的块全部计为新增，消失的块全部计为删除。计入对应语言（GDScript / Shader）。Scene / Resource 文件本身的行数不参与。
- 重命名：按 git 的重命名检测处理，只统计内容差异，不把整个文件算作删除+新增。
- 路径范围：只统计位于扫描根目录下的文件（扫描根可以是仓库的子目录，例如 `src/`）。路径前缀需正确换算。
- 排除规则：以 `.` 开头的目录、`--exclude-dir`、`--exclude-addons` 按文件**在该提交中的路径**判断；`.gdignore` 按**当前工作区**判断（不回溯历史中的 `.gdignore`）；`.gitignore` 无需处理（被忽略的文件不在历史中）。
- 未提交改动：工作区（含暂存区）相对 HEAD 的变化单独列为 `(uncommitted)` 一行，不并入任何日期；未跟踪的新文件也计入（作为全量新增），但要遵守上面的排除规则和 `.gitignore`。
- 暂不支持按作者过滤（`--author`），工具面向独立开发者。

规则有任何不明确的地方，**先问用户，不要自行决定**。规则一旦改变，同步更新 README 的"计数规则"一节（中英两版）。

---

## 5. 扫描与排除

- 根目录：命令行参数给出的路径，默认当前目录。如果该目录或其上级存在 `project.godot`，在输出中显示项目名（读取 `config/name`）。
- 默认排除：`.godot/`、`.git/`、其他以 `.` 开头的目录。
- 包含 `.gdignore` 文件的目录及其子目录全部跳过（与 Godot 行为一致）。
- `.gitignore` 支持放在阶段 3，可以用 `--no-ignore` 关闭。
- 插件识别（阶段 5）：
  - `addons/` 目录以项目根目录（`project.godot` 所在目录）为准，即 `<项目根>/addons/<目录>/`；找不到 `project.godot` 时退回扫描根目录下的 `addons/`。
  - `addons/` 的每个直接子目录视为一个插件：有 `plugin.cfg` 时读取 `[plugin]` 段的 `name` 与 `version` 作为显示名和版本；没有 `plugin.cfg`（纯脚本库、GDExtension 等）时用目录名作为显示名并标注为无 `plugin.cfg`。
  - `plugin.cfg` 解析失败时警告到 stderr，退回目录名，继续。
  - 不在项目根 `addons/` 下的同名嵌套目录不视为插件。

---

## 6. 命令行接口（目标形态）

```
gdloc [路径] [选项]

--by-file            按文件列出
--by-dir             按顶层目录汇总
--by-addon           按插件汇总，插件之外的部分列为 "(project)"
--exclude-addons     不统计 addons/ 目录
--exclude-dir a,b    额外排除目录
--sort <列>          code | comments | blanks | lines | files，默认 code
--top N              只显示前 N 行
--json               以 JSON 输出（字段名使用英文 snake_case）
--no-ignore          不读取 .gitignore
--daily              按天统计新增/删除（阶段 8），默认最近 14 天
--weekly             按 ISO 周统计新增/删除（阶段 8），默认最近 12 周
--since YYYY-MM-DD   历史统计的起始日期（含），需配合 --daily / --weekly
--until YYYY-MM-DD   历史统计的结束日期（含），需配合 --daily / --weekly
--version
```

- 默认输出：按语言汇总的表格，列为 Language / Files / Lines / Code / Comments / Doc / Blanks，末尾一行 Total。
- Total 只汇总代码类语言；Scene/Resource 单独显示在分隔线下方。
- 历史统计输出（`--daily` / `--weekly`，规则见 4.7）：
  - 列为 Date（`--weekly` 时为 Week，显示为该周周一的日期）/ Commits / +Code / -Code / Net / +Comments / -Comments / +Blanks / -Blanks / Code，其中最后一列 Code 是该时段结束时的代码总量。
  - 范围内没有提交的日期/周也显示一行，数值为 0，不跳过。按时间升序排列。
  - 末尾依次为 `(uncommitted)` 一行（无未提交改动时数值为 0）和 Total 一行（汇总整个时间范围，不含 `(uncommitted)`）。不显示日均/周均。
  - `--since` / `--until` 覆盖默认的 14 天 / 12 周；只给其中一个时，另一端分别取"最早提交"/"今天"。日期格式错误或 since 晚于 until → 退出码 1。
  - `--daily` 与 `--weekly` 互斥；二者都不能与 `--by-file` / `--by-dir` / `--by-addon` / `--stats` 同时使用；单独给 `--since` / `--until` 而没有 `--daily` / `--weekly` 也是参数错误。可与 `--exclude-dir`、`--exclude-addons`、`--json`、`--top`（保留最近的 N 行）同时使用。
  - JSON：字段使用 snake_case，至少包含时间段列表、`uncommitted`、`total`。
- 退出码：0 正常；1 参数错误；2 路径不存在或不可读，或历史统计时不在 git 仓库内 / 找不到 git。
- 新增或修改参数时，同步更新 README（中英两版）和本节。

---

## 7. 开发阶段

**每次只做一个阶段。完成并经用户确认后，才开始下一个。不要提前实现后续阶段的功能。**

| 阶段 | 内容 | 完成标准 |
|---|---|---|
| 0 | 搭建骨架：go.mod、目录结构、`--version`、能遍历目录并按扩展名分类 | 能列出被识别的文件清单 |
| 1 | GDScript 计数器 | 4.2 节每条规则都有测试用例且通过 |
| 2 | Shader 计数器 | 4.3 节每条规则都有测试用例且通过 |
| 3 | 表格输出、默认排除、`.gdignore`、`.gitignore`、`--by-file`、`--sort`、`--top`、`--json`、`--exclude-dir`、project.godot 项目名识别 | 在真实项目上运行，与 scc 结果对比，差异全部可解释 |
| 4 | tscn/tres 统计 + 内嵌 GDScript/Shader 提取 | 用真实样本测试通过 |
| 5 | 插件识别、`--by-addon`、`--by-dir`、`--exclude-addons` | 在含多个插件的项目上验证 |
| 6 | 进阶统计：疑似"被注释掉的代码"（启发式，单独一列，标明为估算）、`func`/`signal`/`class_name`/`@export` 数量、最长文件和最长函数排行 | 启发式规则写进 README 并有测试 |
| 7 | C# 支持（可选） | 4.4 节规则有测试且通过 |
| 8 | 历史增量统计：`--daily`、`--weekly`、`--since`、`--until`、`(uncommitted)` 行（规则见 4.7、6） | 4.7 节每条规则都有测试（测试中现场创建临时 git 仓库并固定作者日期）；在用户的真实 git 项目上运行，结果能用 `git log` 抽查解释 |

阶段 7 为可选，经用户同意暂时跳过，阶段 8 可以在其之前进行。

---

## 8. 测试要求

- 每条计数规则至少一个测试用例，用表驱动测试（table-driven tests）。
- 样例文件放在 `testdata/`，期望结果写在测试代码或同名的期望文件中。
- 边界情况必须覆盖：字符串中的注释符号、行尾注释、多行字符串/注释的开始和结束在同一行、空文件、只有 BOM 的文件、CRLF 换行、最后一行无换行符。
- **严禁为了让测试通过而修改期望结果**。如果认为期望结果本身错了，停下来向用户说明原因，由用户决定。
- 与 scc 对比时出现的差异，要在 README（中英两版）中记录原因（例如对多行字符串、`#region` 的处理不同）。

---

## 9. 每次修改后必须执行

```
gofmt -l .        # 无输出才算通过
go vet ./...
go test ./...
go build ./cmd/gdloc
```

全部通过才可以报告完成。报告时简要说明：改了什么、新增了哪些测试、有没有未解决的问题。

---

## 10. Agent 行为准则

- 只做当前任务要求的事。发现其他可改进之处，列出来建议，不要顺手改。
- 不确定时提问，不要猜测用户意图，尤其是计数规则和命令行行为。
- 不删除或重写已有功能，除非任务明确要求。
- 不引入第三方依赖，除非用户同意（见第 2 节）。
- 代码标识符用英文；代码注释用简体中文，简洁说明"为什么"，不复述代码。
- 错误信息和命令行帮助文本用英文。
- README 中英双语：`README.md` 为英文（默认），`README.zh-CN.md` 为简体中文，两版内容保持一致，修改时同时更新。
- 提交信息格式：`阶段N: 简短描述`。
- 每个阶段完成并经用户确认后，按"阶段N: 简短描述"格式提交，未经确认不提交。
