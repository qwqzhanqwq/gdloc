# gdloc

Godot 4 项目代码行数统计工具（命令行，用 Go 编写）。

## 构建

```sh
go build ./cmd/gdloc
```

## 用法

```sh
gdloc [路径] [选项]
```

选项（参数可放在路径前后）：

| 选项 | 说明 |
|---|---|
| `--by-file` | 每个文件一行，列为 Path / Language / Lines / Code / Comments / Doc / Blanks |
| `--sort <列>` | `code`（默认）\| `comments` \| `blanks` \| `lines` \| `files`，降序 |
| `--top N` | 只显示前 N 行；Total 仍按全部计算 |
| `--json` | 以 JSON 输出（字段名 snake_case） |
| `--exclude-dir a,b` | 按目录名排除，任意层级命中即跳过 |
| `--no-ignore` | 不读取 `.gitignore`（`.gitignore` 支持待定，当前无效果） |
| `--version` | 显示版本 |

默认输出：按语言汇总的表格（Language / Files / Lines / Code / Comments / Doc / Blanks），末尾 Total 行。找到 `project.godot` 时，表格上方显示项目名与扫描根目录。存在 `.cs` 文件时表格下方提示未统计数量。`--by-file` 时 `--sort files` 无意义，退回按 Lines 排序。

退出码：`0` 正常；`1` 参数错误；`2` 路径不存在或不可读。

### JSON 结构

```json
{
  "project_name": "WindupWonderland",
  "root": "D:/WindupWonderland/src",
  "languages": [
    {"language": "GDScript", "files": 115, "lines": 24927, "code": 19672, "comments": 2263, "doc": 1498, "blanks": 2992}
  ],
  "total": {"language": "Total", "files": 132, "lines": 26285, "code": 20563, "comments": 2550, "doc": 1658, "blanks": 3172},
  "files": [
    {"path": "main.gd", "language": "GDScript", "lines": 24, "code": 20, "comments": 2, "doc": 1, "blanks": 2}
  ]
}
```

`files` 仅在带 `--by-file` 时出现；`--top N` 会截断 `languages` / `files` 数组，但 `total` 始终是全量。

## 计数规则

每一行只归入一类，优先级：只含空白字符 → 空行；去掉注释后仍有非空白内容 → 代码行；其余 → 注释行。`Doc`（文档注释）是 `Comments` 的子集。

### GDScript（`.gd`，阶段 1）

- 注释以 `#` 开头到行尾。行首（忽略缩进）为 `##` 的注释行额外计为文档注释；`#region` / `#endregion` 计为注释。
- 字符串内的 `#` 不是注释，识别的形式：`"..."`、`'...'`、三引号 `"""..."""` / `'''...'''`、前缀 `&"..."`、`^"..."`、`r"..."`。
- 三引号多行字符串的每一行计为代码行，但内部的纯空白行仍按 4.1 计为空行。
- 单/双引号串不跨行；三引号串未闭合延续到文件末尾，其余内容按代码计。
- 文件末尾的换行符不产生额外一行；空文件 0 行；只有一个换行符的文件为 1 个空行。
- 读取时去掉开头的 UTF-8 BOM，兼容 `\n` 与 `\r\n`，末行无换行符也计入。

### Godot Shader（`.gdshader`、`.gdshaderinc`，阶段 2）

- 单行注释 `//`，多行注释 `/* ... */`（不嵌套）。
- `/** ... */` 是文档注释：覆盖的注释行同时计入 Comments 和 Doc；空的 `/**/` 不算文档注释。
- 预处理指令（`#include`、`#define`、`#ifdef`、`#endif` 等）计为代码行，不是注释。
- 字符串 `"..."` 内的 `//`、`/*` 不触发注释判定（`#include`、`hint_enum` 等均可能含字符串）。
- 块注释内部的纯空白行计为空行；未闭合的块注释延续到文件末尾，其余内容按注释计。
- 空文件、BOM、CRLF、末行无换行的处理同 GDScript。

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

差异仅来自 `addons/ww_water/baker/river_baker.gd` 的第 75、78、80、1120 行：这些是多行字符串（`"""..."""`）内部的纯空白行。gdloc 按"只含空白字符即空行"（含多行字符串内部）计为空行，scc 把字符串内部的行一律计为代码。gdloc 的行为与 AGENTS.md 4.1 一致。

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
