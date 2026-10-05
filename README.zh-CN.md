# gdloc

[English](README.md) | 简体中文

Godot 4 项目代码行数统计工具（命令行，用 Go 编写）。理解 Godot 的项目结构：识别项目名和 `addons/` 插件，统计 `.tscn` / `.tres` 里的内嵌脚本，遵守 `.gdignore` 与 `.gitignore`。

```
$ gdloc
Project: WindupWonderland
Root:    D:\WindupWonderland\src
Language  Files   Lines    Code  Comments    Doc  Blanks
GDScript    115  24,943  19,680     2,267  1,499   2,996
Shader       17   1,358     891       287    160     180
Total       132  26,301  20,571     2,554  1,659   3,176
--------------------------------------------------------
Scene        35   3,217
Resource     12     187
```

- 支持 GDScript（`.gd`）与 Godot Shader（`.gdshader` / `.gdshaderinc`），区分代码、注释、文档注释（`##`、`/** */`）和空行。
- 按文件、顶层目录或插件分组；`--stats` 给出函数/信号数量、最长文件和最长函数。
- 只支持 Godot 4；不统计 C#（`.cs`），存在时会提示数量。
- 对被扫描的项目只读，不联网。

## 安装

**下载可执行文件**：在 [Releases](https://github.com/qwqzhanqwq/gdloc/releases) 下载对应平台的压缩包（Windows 为 `.zip`，macOS / Linux 为 `.tar.gz`），解压后把 `gdloc` 放到 `PATH` 中的目录。

**Scoop（Windows）**：

```powershell
scoop bucket add gdloc https://github.com/qwqzhanqwq/scoop-bucket
scoop install gdloc
```

升级：先 `scoop update` 拉取 bucket 最新内容，再 `scoop update gdloc`。

**用 Go 安装**（需要 Go 1.25+）：

```sh
go install github.com/qwqzhanqwq/gdloc/cmd/gdloc@latest
```

## 构建

```sh
go build ./cmd/gdloc
```

## 用法

```sh
gdloc [路径] [选项]
```

路径默认为当前目录。项目名与插件识别依赖 `project.godot`：它会在扫描目录及其上级中查找。如果仓库根目录下的 `project.godot` 在子目录里（如 `src/`），请在该子目录中运行，或把它作为路径传入。

选项（参数可放在路径前后）：

| 选项 | 说明 |
|---|---|
| `--by-file` | 每个文件一行，列为 Path / Language / Lines / Code / Comments / Doc / Blanks；内嵌代码用 `文件::id` 形式（主资源为 `文件::[resource]`） |
| `--by-dir` | 按扫描根目录下的顶层目录分组，根目录文件归入 `(root)`；列为 Group / Files / Lines / Code / Comments / Doc / Blanks |
| `--by-addon` | 按插件分组，非插件文件归入 `(project)`；列为 Addon / Version / Files / Lines / Code / Comments / Doc / Blanks |
| `--exclude-addons` | 整个 `addons/` 目录不统计；不能与 `--by-addon` 同用 |
| `--stats` | 进阶统计视图（注释分析、GDScript 结构、最长文件/函数）；不能与 `--by-file` / `--by-dir` / `--by-addon` 同用 |
| `--sort <列>` | `code`（默认）\| `comments` \| `blanks` \| `lines` \| `files`，降序 |
| `--top N` | 只显示前 N 行；Total 仍按全部计算 |
| `--json` | 以 JSON 输出（字段名 snake_case） |
| `--exclude-dir a,b` | 按目录名排除，任意层级命中即跳过 |
| `--no-ignore` | 不读取 `.gitignore` |
| `--version` | 显示版本 |

`--by-file`、`--by-dir`、`--by-addon` 三者互斥，同时使用报错退出码 1；`--exclude-addons` 与 `--by-addon` 同用也报错；`--stats` 与这三个分组参数互斥。`--sort` 与 `--top` 对分组结果生效，Total 始终按全部计算。

默认输出：按语言汇总的表格（Language / Files / Lines / Code / Comments / Doc / Blanks），末尾 Total 行。找到 `project.godot` 时，表格上方显示项目名与扫描根目录。`.tscn`/`.tres` 中提取出的内嵌代码单独列为 `GDScript (embedded)` / `Shader (embedded)`（Files 为含内嵌代码的文件数，内嵌块数量见 `--by-file` 与 JSON），并计入 Total。分隔线下方单独显示 Scene（`.tscn`）与 Resource（`.tres`）的 Files 和 Lines（不计入 Total）。存在 `.cs` 文件或 VisualShader 时，表格下方提示未统计数量。`--by-file` 时 `--sort files` 无意义，退回按 Lines 排序。

退出码：`0` 正常；`1` 参数错误；`2` 路径不存在或不可读。

### JSON 结构

```json
{
  "project_name": "WindupWonderland",
  "root": "D:/WindupWonderland/src",
  "languages": [
    {"language": "GDScript", "files": 115, "blocks": 0, "lines": 24927, "code": 19672, "comments": 2263, "doc": 1498, "blanks": 2992},
    {"language": "Shader", "files": 17, "blocks": 0, "lines": 1358, "code": 891, "comments": 287, "doc": 160, "blanks": 180},
    {"language": "GDScript (embedded)", "files": 3, "blocks": 3, "lines": 20, "code": 13, "comments": 4, "doc": 2, "blanks": 3}
  ],
  "total": {"language": "Total", "files": 132, "blocks": 3, "lines": 26285, "code": 20563, "comments": 2550, "doc": 1658, "blanks": 3172},
  "scenes": {"files": 35, "lines": 3210},
  "resources": {"files": 12, "lines": 187},
  "visual_shaders": 0,
  "addons": [
    {"name": "WW Water", "dir": "addons/ww_water", "version": "0.1.0", "has_plugin_cfg": true, "files": 23, "lines": 4201, "code": 3186, "comments": 368, "doc": 131, "blanks": 647},
    {"name": "(project)", "dir": "", "version": "", "has_plugin_cfg": false, "files": 46, "lines": 9998, "code": 7157, "comments": 1374, "doc": 961, "blanks": 1467}
  ],
  "dirs": [
    {"name": "addons", "files": 86, "lines": 16287, "code": 13406, "comments": 1176, "doc": 697, "blanks": 1705}
  ],
  "files": [
    {"path": "main.gd", "language": "GDScript", "lines": 24, "code": 20, "comments": 2, "doc": 1, "blanks": 2},
    {"path": "scenes/main.tscn::GDScript_qr1jj", "language": "GDScript (embedded)", "lines": 14, "code": 9, "comments": 2, "doc": 1, "blanks": 3}
  ],
  "stats": {
    "languages": [
      {"language": "GDScript", "comments": 2263, "suspected": 36, "ratio": 0.0159}
    ],
    "structure": {"funcs": 1367, "signals": 44, "class_names": 90, "exports": 360},
    "longest_files": [
      {"path": "scenes/player/puppet.gd", "language": "GDScript", "code": 1155}
    ],
    "longest_functions": [
      {"path": "addons/ww_water/river_manager.gd", "line": 158, "name": "_get_property_list", "length": 190}
    ]
  }
}
```

- `languages` 只含代码类语言（含内嵌），`blocks` 为内嵌块数量；`files` 为含该内嵌代码的文件数。
- `total` 汇总代码类语言；`scenes` / `resources` 单独给出 Files 与 Lines，不计入 `total`。
- `visual_shaders` 为 VisualShader 资源数量。
- `addons` 仅在 `--by-addon` 时出现（含 `name` / `dir` / `version` / `has_plugin_cfg` 与各项计数）；`dirs` 仅在 `--by-dir` 时出现。
- `stats` 仅在 `--stats` 时出现，含注释分析（`comments` / `suspected` / `ratio`）、`structure`、`longest_files`（`path` / `language` / `code`）、`longest_functions`（`path` / `line` / `name` / `length`）。
- `files` 仅在带 `--by-file` 时出现，内嵌块路径为 `文件::id`；`--top N` 会截断 `languages` / `addons` / `dirs` / `files` / `stats.longest_*` 数组，但 `total` 始终是全量。

## 扫描与排除

- 默认跳过 `.godot/`、`.git/` 等所有以 `.` 开头的目录，以及包含 `.gdignore` 的目录及其子树。
- `--exclude-dir a,b` 按目录名匹配，任意层级命中即跳过。
- 插件识别：`addons/` 以项目根目录（`project.godot` 所在目录）为准，即 `<项目根>/addons/<目录>/`；找不到 `project.godot` 时退回扫描根目录下的 `addons/`。`addons/` 的每个直接子目录视为一个插件，有 `plugin.cfg` 时读 `[plugin]` 的 `name` / `version`，没有时用目录名（标注无 `plugin.cfg`）；`plugin.cfg` 解析失败会警告并退回目录名。不在项目根 `addons/` 下的同名嵌套目录不算插件。`--exclude-addons` 跳过整个 `addons/` 目录。
- `.gitignore`：只在扫描根目录及其子目录中查找（不向上读取根目录以外），逐目录收集、就近优先，子目录规则可覆盖上级。支持 `#` 注释、`!` 否定、`/` 锚定、尾 `/` 仅匹配目录、`*` / `?` / `**` 通配。被忽略的目录整棵子树跳过。`--no-ignore` 时完全不读 `.gitignore`。

## 计数规则

每一行只归入一类，优先级：只含空白字符 → 空行；去掉注释后仍有非空白内容 → 代码行；其余 → 注释行。`Doc`（文档注释）是 `Comments` 的子集。

### GDScript（`.gd`）

- 注释以 `#` 开头到行尾。行首（忽略缩进）为 `##` 的注释行额外计为文档注释；`#region` / `#endregion` 计为注释。
- 字符串内的 `#` 不是注释，识别的形式：`"..."`、`'...'`、三引号 `"""..."""` / `'''...'''`、前缀 `&"..."`、`^"..."`、`r"..."`。
- 三引号多行字符串的每一行计为代码行，但内部的纯空白行仍计为空行。
- 单/双引号串不跨行；三引号串未闭合延续到文件末尾，其余内容按代码计。
- 文件末尾的换行符不产生额外一行；空文件 0 行；只有一个换行符的文件为 1 个空行。
- 读取时去掉开头的 UTF-8 BOM，兼容 `\n` 与 `\r\n`，末行无换行符也计入。

### Godot Shader（`.gdshader`、`.gdshaderinc`）

- 单行注释 `//`，多行注释 `/* ... */`（不嵌套）。
- `/** ... */` 是文档注释：覆盖的注释行同时计入 Comments 和 Doc；空的 `/**/` 不算文档注释。
- 预处理指令（`#include`、`#define`、`#ifdef`、`#endif` 等）计为代码行，不是注释。
- 字符串 `"..."` 内的 `//`、`/*` 不触发注释判定（`#include`、`hint_enum` 等均可能含字符串）。
- 块注释内部的纯空白行计为空行；未闭合的块注释延续到文件末尾，其余内容按注释计。
- 空文件、BOM、CRLF、末行无换行的处理同 GDScript。

### 场景与资源（`.tscn`、`.tres`）

- 文件本身只统计总行数，分别列为 Scene 与 Resource，不计入代码总量。
- 内嵌代码提取后交给对应计数器：
  - `[sub_resource type="GDScript" ...]` 的 `script/source` → `GDScript (embedded)`。
  - `[sub_resource type="Shader" ...]` 的 `code` → `Shader (embedded)`。
  - `[gd_resource type="Shader" ...]` 的 `[resource]` 段 `code` → `Shader (embedded)`（主资源为 Shader 的 `.tres`）。
- 字符串值支持真实换行与 `\n` 转义两种存储，`\"`、`\\` 等按 Godot 规则反转义。
- `VisualShader` 资源只计数量，不计行数。

## 进阶统计（`--stats`）

默认表格与其它输出不变，以下内容只在 `--stats` 视图出现。内嵌代码一并计入，路径用 `文件::id`；`--exclude-dir`、`--exclude-addons` 同样生效；最长列表条数由 `--top` 控制（默认 10）。

- **注释分析**：按语言给出注释行数、疑似"被注释掉的代码"行数及占比。
- **GDScript 结构**：`func`（含 `static func`，不含匿名 `func(...)`）、`signal`、`class_name`、`@export`（含 `@export_range` 等变体）数量。
- **最长文件**：代码行数最多的文件前 N 个。
- **最长函数**：GDScript 函数按长度（从 `func` 行到函数体结束的代码行数，以缩进判断结束，空行与注释行不计入长度）排序，输出 `路径:行号 函数名 长度`。

### 疑似被注释掉的代码（启发式估算）

**这是估算，不是精确判断。** 只考察普通注释行（不含文档注释、`#region`/`#endregion`）。去掉注释符号与首尾空白后：

- 含中文（CJK）字符的注释一律不算。
- 以 `TODO`、`FIXME`、`NOTE`、`HACK` 开头的不算。
- GDScript 满足以下任一即视为疑似：以 `func`、`var`、`const`、`if`、`elif`、`else`、`for`、`while`、`match`、`return`、`await`、`pass`、`extends`、`class_name`、`signal`、`@export`、`@onready` 开头；或以 `)` / `:` 结尾；或包含 `" = "`。
- Shader 满足以下任一即视为疑似：以 `uniform`、`void`、`float`、`int`、`vec2`、`vec3`、`vec4`、`if`、`return` 开头；或以 `;` 结尾。

## 与 scc 的差异

### GDScript

以 `D:\WindupWonderland\src` 为样本（115 个 `.gd` 文件）：

| 指标 | gdloc | scc |
|---|---|---|
| Files | 115 | 115 |
| Lines | 24,927 | 24,927 |
| Code | 19,672 | 19,676 |
| Comments | 2,263 | 2,263 |
| Blanks | 2,992 | 2,988 |

差异仅来自 `addons/ww_water/baker/river_baker.gd` 的第 75、78、80、1120 行：这些是多行字符串（`"""..."""`）内部的纯空白行。gdloc 按"只含空白字符即空行"（含多行字符串内部）计为空行，scc 把字符串内部的行一律计为代码。gdloc 的行为符合上面的通用判定规则。

另外两个项目 GDScript 完全一致：

| 项目 | Files | Lines | Code | Comments | Blanks |
|---|---|---|---|---|---|
| `D:\Godot\astral-mason` | 25 | 4,511 | 2,371 | 1,196 | 944 |
| `D:\Godot\flipped-sky` | 21 | 2,308 | 1,002 | 842 | 464 |

### Shader

以 `D:\WindupWonderland\src` 为样本（17 个 `.gdshader` / `.gdshaderinc` 文件，目录内无 `.glsl`）：

| 项目 | Files | Lines | Code | Comments | Blanks |
|---|---|---|---|---|---|
| `D:\WindupWonderland\src` | 17 | 1,358 | 891 | 287 | 180 |
| `D:\Godot\astral-mason` | 3 | 247 | 130 | 67 | 50 |
| `D:\Godot\flipped-sky` | 0 | 0 | 0 | 0 | 0 |

gdloc 的 Shader 组与 scc 的 GLSL 组逐文件对比 0 差异。scc 的 GLSL 组恰好只包含 `.gdshader` / `.gdshaderinc`；本机 scc 3.7.0 不识别该扩展名，对比时把样本复制到临时目录改名为 `.glsl` 交给 scc 解析。三个项目均无原生 `.glsl`。

### Scene / Resource

scc 的 "Godot Scene" **只统计 `.tscn`**（Windup 有 12 个 `.tres` 但未计入）。对比 Files 与 Lines：

| 项目 | gdloc Scene Files | scc Godot Scene Files | gdloc Scene Lines | scc Lines |
|---|---|---|---|---|
| `D:\WindupWonderland\src` | 35 | 35 | 3,210 | 3,210 |
| `D:\Godot\astral-mason` | 19 | 19 | 4,657 | 4,657 |
| `D:\Godot\flipped-sky` | 15 | 15 | 3,254 | 3,254 |

`.tscn` 的 Files 与 Lines 完全一致。`.tres` 没有对应的 scc 组（scc 不计入 Godot Scene），gdloc 单独列为 Resource：Windup `12 / 187`，另两个项目为 0。三个真实项目均无内嵌 GDScript / Shader（`script/source` 与 `type="Shader"` 命中数为 0），故表格中无 `(embedded)` 行。

## 发布

推送 `v*` 形式的 tag（如 `v0.1.0`）后，GitHub Actions 用 GoReleaser 编译 Windows / macOS / Linux 的 amd64 与 arm64 版本并发布到 Releases，同时把 Scoop manifest 推送到 [scoop-bucket](https://github.com/qwqzhanqwq/scoop-bucket)（需要仓库 secret `SCOOP_BUCKET_TOKEN`），配置见 `.goreleaser.yaml` 与 `.github/workflows/release.yml`。

## 许可证

MIT，见 [LICENSE](LICENSE)。
