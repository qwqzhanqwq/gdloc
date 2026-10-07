# gdloc

English | [简体中文](README.zh-CN.md)

A command-line line counter for Godot 4 projects, written in Go. It understands how Godot projects are laid out: it reads the project name and the plugins under `addons/`, counts scripts embedded in `.tscn` / `.tres` files, and respects `.gdignore` and `.gitignore`.

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

- Counts GDScript (`.gd`) and Godot shaders (`.gdshader` / `.gdshaderinc`), separating code, comments, doc comments (`##`, `/** */`) and blank lines.
- Groups results by file, top-level directory or plugin; `--stats` reports function/signal counts and the longest files and functions.
- Reports added/removed lines per day or ISO week from git history (`--daily` / `--weekly`), with an `(uncommitted)` row. Git is only ever called with read-only commands.
- Godot 4 only. C# (`.cs`) is not counted; if any `.cs` files exist, their number is shown as a note.
- Read-only on the scanned project. No network access.

## Install

**Download a binary**: grab the archive for your platform from [Releases](https://github.com/qwqzhanqwq/gdloc/releases) (`.zip` on Windows, `.tar.gz` on macOS / Linux), extract it, and put `gdloc` in a directory on your `PATH`.

**Scoop (Windows)**:

```powershell
scoop bucket add gdloc https://github.com/qwqzhanqwq/scoop-bucket
scoop install gdloc
```

To upgrade, run `scoop update` to refresh the bucket, then `scoop update gdloc`.

**Go** (requires Go 1.25+):

```sh
go install github.com/qwqzhanqwq/gdloc/cmd/gdloc@latest
```

## Build

```sh
go build ./cmd/gdloc
```

## Usage

```sh
gdloc [path] [options]
```

The path defaults to the current directory. The project name and plugin detection depend on `project.godot`, which is looked up in the scanned directory and its parents. If your repository keeps `project.godot` in a subdirectory (e.g. `src/`), run gdloc in that subdirectory or pass it as the path.

Options (may appear before or after the path):

| Option | Description |
|---|---|
| `--by-file` | One row per file, columns Path / Language / Lines / Code / Comments / Doc / Blanks. Embedded code is shown as `file::id` (`file::[resource]` for a main resource) |
| `--by-dir` | Group by top-level directory under the scan root; files in the root itself go to `(root)`. Columns Group / Files / Lines / Code / Comments / Doc / Blanks |
| `--by-addon` | Group by plugin; files outside plugins go to `(project)`. Columns Addon / Version / Files / Lines / Code / Comments / Doc / Blanks |
| `--exclude-addons` | Skip the whole `addons/` directory. Cannot be combined with `--by-addon` |
| `--stats` | Advanced statistics (comment analysis, GDScript structure, longest files/functions). Cannot be combined with `--by-file` / `--by-dir` / `--by-addon` |
| `--sort <col>` | `code` (default) \| `comments` \| `blanks` \| `lines` \| `files`, descending |
| `--top N` | Show only the first N rows; Total still covers everything |
| `--json` | JSON output (snake_case field names) |
| `--exclude-dir a,b` | Exclude directories by name, at any depth |
| `--no-ignore` | Do not read `.gitignore` |
| `--daily` | Added/removed lines per day from git history (default: last 14 days) |
| `--weekly` | Added/removed lines per ISO week, rows labelled with that week's Monday (default: last 12 weeks) |
| `--since YYYY-MM-DD` | First day of the history range, inclusive; requires `--daily` or `--weekly` |
| `--until YYYY-MM-DD` | Last day of the history range, inclusive; requires `--daily` or `--weekly` |
| `--version` | Print the version |

`--by-file`, `--by-dir` and `--by-addon` are mutually exclusive; combining them is an error (exit code 1). Combining `--exclude-addons` with `--by-addon` is also an error, and `--stats` cannot be combined with any of the three grouping options. `--sort` and `--top` apply to the grouped rows; Total is always computed over everything.

`--daily` and `--weekly` are mutually exclusive as well, cannot be combined with `--by-file` / `--by-dir` / `--by-addon` / `--stats`, and `--since` / `--until` without one of them is an error (exit code 1). In history mode they can be combined with `--exclude-dir`, `--exclude-addons`, `--json` and `--top`; `--sort` and `--no-ignore` are accepted but `--sort` has no effect (`--no-ignore` still decides whether ignored untracked files are part of the `(uncommitted)` row).

Default output: a per-language table (Language / Files / Lines / Code / Comments / Doc / Blanks) ending with a Total row. When `project.godot` is found, the project name and scan root are printed above the table. Code extracted from `.tscn` / `.tres` files is listed separately as `GDScript (embedded)` / `Shader (embedded)` (Files is the number of files containing embedded code; the number of embedded blocks is available in `--by-file` and JSON) and is included in Total. Below the separator, Scene (`.tscn`) and Resource (`.tres`) show Files and Lines only and are not part of Total. If `.cs` files or VisualShader resources exist, a note below the table shows how many were not counted. With `--by-file`, `--sort files` is meaningless and falls back to sorting by Lines.

Exit codes: `0` success; `1` invalid arguments (including a malformed `--since` / `--until` and `--since` later than `--until`); `2` path does not exist or is not readable, or `--daily` / `--weekly` was used outside a git repository or without `git` on `PATH`.

### JSON format

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

- `languages` contains only code languages (including embedded ones). `blocks` is the number of embedded blocks; `files` is the number of files containing that embedded code.
- `total` sums the code languages. `scenes` / `resources` give Files and Lines separately and are not part of `total`.
- `visual_shaders` is the number of VisualShader resources.
- `addons` appears only with `--by-addon` (with `name` / `dir` / `version` / `has_plugin_cfg` and the counts); `dirs` appears only with `--by-dir`.
- `stats` appears only with `--stats`: comment analysis (`comments` / `suspected` / `ratio`), `structure`, `longest_files` (`path` / `language` / `code`) and `longest_functions` (`path` / `line` / `name` / `length`).
- `files` appears only with `--by-file`; embedded blocks use `file::id` paths. `--top N` truncates the `languages` / `addons` / `dirs` / `files` / `stats.longest_*` arrays, but `total` always covers everything.

## History statistics (`--daily` / `--weekly`)

`--daily` and `--weekly` show how many lines were added and removed over time, straight from the repository's git history. They need `git` on `PATH` and a scan root inside a git work tree; only read-only git commands are used (always with `--no-optional-locks`), so the scanned repository is never modified.

```
$ gdloc --daily
Project: WindupWonderland
Root:    D:\WindupWonderland\src
Date           Commits   +Code  -Code     Net  +Comments  -Comments  +Blanks  -Blanks    Code
2026-10-01           7   8,618     71   8,547      1,074         22    1,721        3   8,547
2026-10-02           8   2,586    169   2,417        473         56      471        4  10,964
2026-10-03           0       0      0       0          0          0        0        0  10,964
...
(uncommitted)        0       0      0       0          0          0        0        0  22,263
Total               40  23,821  1,558  22,263      3,171        248    3,512       99  22,263
```

Columns are Date (Week for `--weekly`) / Commits / +Code / -Code / Net / +Comments / -Comments / +Blanks / -Blanks / Code. Rows are in ascending order and periods without commits are listed with zeros instead of being skipped. `(uncommitted)` and `Total` come last, and no averages are shown.

- **Dates**: the *author date* of each commit, grouped in the local timezone. Weekly buckets are ISO weeks and start on Monday; the row is labelled with that Monday's date. Commits are bucketed one by one, so non-monotonic dates do not distort the result.
- **Commits**: non-merge commits reachable from `HEAD` that changed at least one counted file (`.gd`, `.gdshader`, `.gdshaderinc`, `.tscn`, `.tres`) under the scan root. Merge commits are skipped entirely; every other commit is compared with its first parent, and a root commit with the empty tree.
- **Added / removed**: both versions of a change are classified line by line with the same counters used for the totals; added lines take their class from the new version, removed lines from the old one. Doc comments are folded into Comments here (there is no separate Doc column). `Net` is `+Code` minus `-Code`.
- **`Code` column**: the code total at the end of that period. It is the code in the `HEAD` tree under the same rules, minus the net change of every commit authored after the period ends, so a period always shows the state at its own end (a weekly row always shows Sunday's state). Uncommitted work is not included.
- **Embedded code**: when a `.tscn` / `.tres` changes, embedded blocks are extracted from both versions and paired by block id; a changed block is diffed line by line, a new block counts as an addition and a removed block as a deletion. The scene/resource file's own lines never count as code.
- **Renames**: git's rename detection (`-M`) is used, so a rename contributes only its content difference. A rename that crosses the scan root or the exclusion rules, or that changes the file type, is counted as a full deletion plus a full addition instead, since the two versions are not comparable under the rules.
- **Scope**: only files under the scan root are counted, and the scan root may be a subdirectory of the repository (e.g. `src/`). The usual exclusions apply, evaluated on the path a file had in that commit: hidden directories, `--exclude-dir`, `--exclude-addons` and `.gdignore` directories — `.gdignore` is taken from the *current workspace*, it is never read from history. `.gitignore` needs no handling, since ignored files are not committed.
- **`(uncommitted)`**: the working tree including the index, compared with `HEAD`, plus untracked files, which count as whole new files. It gets its own row and is never folded into a date. Untracked files ignored by `.gitignore` are skipped unless `--no-ignore` is given. A worktree file that differs from its blob only by CRLF line endings is not reported as changed. Renames are detected there only once they are staged (git compares the index with `HEAD`), so a rename that exists only in the working tree appears as a deletion of the old file plus an untracked new file.
- **Ranges**: `--daily` defaults to the last 14 days and `--weekly` to the last 12 ISO weeks. `--since` / `--until` (`YYYY-MM-DD`, inclusive, interpreted in the local timezone) override that; if only one of them is given the other end is the earliest commit (`--until` alone) or today (`--since` alone).
- **`--top N`** keeps the newest N periods; `(uncommitted)` and `Total` are always shown, and `Total` always covers the whole range.

Reconciling with `git log --numstat`: gdloc computes a minimal line diff, which `git diff --minimal` matches line for line, while git's default heuristic can add a few redundant add/remove pairs on large rewrites. On top of that, gdloc classifies every line and also counts code embedded in scenes/resources, neither of which `--numstat` can show, and it never counts `.tscn` / `.tres` file lines.

### History JSON

```json
{
  "project_name": "WindupWonderland",
  "root": "D:/WindupWonderland/src",
  "mode": "daily",
  "since": "2026-09-24",
  "until": "2026-10-07",
  "head_code": 22263,
  "periods": [
    {"date": "2026-10-01", "commits": 7, "added_code": 8618, "deleted_code": 71, "net_code": 8547, "added_comments": 1074, "deleted_comments": 22, "added_blanks": 1721, "deleted_blanks": 3, "code": 8547}
  ],
  "uncommitted": {"commits": 0, "added_code": 0, "deleted_code": 0, "net_code": 0, "added_comments": 0, "deleted_comments": 0, "added_blanks": 0, "deleted_blanks": 0, "code": 22263},
  "total": {"commits": 40, "added_code": 23821, "deleted_code": 1558, "net_code": 22263, "added_comments": 3171, "deleted_comments": 248, "added_blanks": 3512, "deleted_blanks": 99, "code": 22263}
}
```

- `mode` is `daily` or `weekly`; `date` is the day, or the Monday of the ISO week for `weekly`.
- `uncommitted` and `total` use the same fields as a period; `commits` is always 0 for `uncommitted`, whose `code` is the current working tree total (the `HEAD` total plus the uncommitted net change).
- `head_code` is the code total of the committed `HEAD` tree, and `total.code` is the total at the end of the range; both ignore uncommitted work.
- `--top N` truncates `periods` to the newest N entries but never truncates `total`.

## Scanning and exclusions

- Directories starting with `.` (such as `.godot/` and `.git/`) are skipped, as are directories containing a `.gdignore` file and everything below them.
- `--exclude-dir a,b` matches directory names at any depth.
- Plugin detection: `addons/` is resolved against the project root (the directory containing `project.godot`), i.e. `<project root>/addons/<dir>/`. Without `project.godot`, `addons/` under the scan root is used. Each direct subdirectory of `addons/` is one plugin: if it has a `plugin.cfg`, `name` / `version` are read from its `[plugin]` section; otherwise the directory name is used (marked as having no `plugin.cfg`). If `plugin.cfg` fails to parse, a warning is printed and the directory name is used. Nested directories named `addons` elsewhere are not plugins. `--exclude-addons` skips the whole `addons/` directory.
- `.gitignore`: only files inside the scan root and its subdirectories are read (nothing above the root). Rules are collected per directory and the nearest one wins, so a subdirectory can override its parents. Supports `#` comments, `!` negation, `/` anchoring, trailing `/` for directories only, and `*` / `?` / `**` wildcards. An ignored directory is skipped with its whole subtree. `--no-ignore` disables `.gitignore` entirely.

## Counting rules

Each line falls into exactly one category, in this order: whitespace only → blank; non-whitespace content left after removing comments → code; otherwise → comment. `Doc` (doc comments) is a subset of `Comments`.

### GDScript (`.gd`)

- Comments start with `#` and run to the end of the line. A comment line starting with `##` (ignoring indentation) also counts as a doc comment. `#region` / `#endregion` count as comments.
- `#` inside a string is not a comment. Recognized string forms: `"..."`, `'...'`, triple-quoted `"""..."""` / `'''...'''`, and the prefixed `&"..."`, `^"..."`, `r"..."`.
- Every line of a triple-quoted string counts as code, except lines inside it that are whitespace only, which count as blank.
- Single- and double-quoted strings do not span lines. An unterminated triple-quoted string runs to the end of the file, and its content counts as code.
- A trailing newline at the end of the file does not create an extra line. An empty file has 0 lines; a file containing only a newline has 1 blank line.
- A leading UTF-8 BOM is stripped, both `\n` and `\r\n` are accepted, and a last line without a newline is counted.

### Godot Shader (`.gdshader`, `.gdshaderinc`)

- Line comments `//`, block comments `/* ... */` (not nested).
- `/** ... */` is a doc comment: the comment lines it covers count toward both Comments and Doc. An empty `/**/` is not a doc comment.
- Preprocessor directives (`#include`, `#define`, `#ifdef`, `#endif`, etc.) are code, not comments.
- `//` and `/*` inside a string `"..."` do not start a comment (strings can appear in `#include`, `hint_enum`, etc.).
- Whitespace-only lines inside a block comment count as blank. An unterminated block comment runs to the end of the file, and its content counts as comment.
- Empty files, BOM, CRLF and a missing final newline are handled as for GDScript.

### Scenes and resources (`.tscn`, `.tres`)

- The files themselves only contribute a total line count, listed as Scene and Resource, and are not part of the code total.
- Embedded code is extracted and passed to the matching counter:
  - `script/source` in `[sub_resource type="GDScript" ...]` → `GDScript (embedded)`.
  - `code` in `[sub_resource type="Shader" ...]` → `Shader (embedded)`.
  - `code` in the `[resource]` section of `[gd_resource type="Shader" ...]` → `Shader (embedded)` (a `.tres` whose main resource is a Shader).
- String values stored with real newlines or with `\n` escapes are both supported; `\"`, `\\` and other escapes are unescaped following Godot's rules.
- `VisualShader` resources are only counted, not line-counted.

History statistics (`--daily` / `--weekly`) classify both versions of every change with exactly these rules, so the two views stay consistent; how buckets, renames and exclusions are handled there is described in the History statistics section above.

## Advanced statistics (`--stats`)

The default table and other outputs are unchanged; the following only appears in the `--stats` view. Embedded code is included, with `file::id` paths. `--exclude-dir` and `--exclude-addons` apply as usual. The length of the "longest" lists is controlled by `--top` (default 10).

- **Comment analysis**: per language, the number of comment lines, the number of suspected "commented-out code" lines, and their ratio.
- **GDScript structure**: counts of `func` (including `static func`, excluding anonymous `func(...)`), `signal`, `class_name` and `@export` (including variants such as `@export_range`).
- **Longest files**: the top N files by code lines.
- **Longest functions**: GDScript functions sorted by length (code lines from the `func` line to the end of the body, determined by indentation; blank and comment lines are not counted), shown as `path:line name length`.

### Suspected commented-out code (heuristic estimate)

**This is an estimate, not an exact judgement.** Only plain comment lines are considered (not doc comments or `#region` / `#endregion`). After removing the comment marker and surrounding whitespace:

- Comments containing CJK characters never count.
- Comments starting with `TODO`, `FIXME`, `NOTE` or `HACK` do not count.
- GDScript: suspected if it starts with `func`, `var`, `const`, `if`, `elif`, `else`, `for`, `while`, `match`, `return`, `await`, `pass`, `extends`, `class_name`, `signal`, `@export` or `@onready`; or ends with `)` / `:`; or contains `" = "`.
- Shader: suspected if it starts with `uniform`, `void`, `float`, `int`, `vec2`, `vec3`, `vec4`, `if` or `return`; or ends with `;`.

## Agent skill

Coding agents (Claude Code, DSH and other harnesses that support skills) work better with gdloc when they know how to drive it: which option answers which question, that the scan root must be the directory holding `project.godot`, that Scene / Resource lines stay out of Total, and that `.cs` files are not counted. A skill that packages exactly that knowledge lives in [`skill/gdloc/`](skill/gdloc):

```
skill/gdloc/
├── SKILL.md                     # when to reach for gdloc, and how to read its numbers
└── scripts/gdloc_summary.py     # condenses `gdloc --json` output for very large projects
```

To install it, put the `skill/gdloc` directory in the skills folder your agent reads (for Claude Code and DSH that is `~/.agents/skills/`, i.e. `%USERPROFILE%\.agents\skills\` on Windows, where a symlink or junction to this directory keeps it in sync with the repository). [`skill/evals/`](skill/evals) holds the fixture generator and the test cases used to check that the skill behaves.

## Differences from scc

### GDScript

Sample: `D:\WindupWonderland\src` (115 `.gd` files):

| Metric | gdloc | scc |
|---|---|---|
| Files | 115 | 115 |
| Lines | 24,927 | 24,927 |
| Code | 19,672 | 19,676 |
| Comments | 2,263 | 2,263 |
| Blanks | 2,992 | 2,988 |

The only difference comes from lines 75, 78, 80 and 1120 of `addons/ww_water/baker/river_baker.gd`: whitespace-only lines inside a triple-quoted string (`"""..."""`). gdloc counts any whitespace-only line as blank, including inside multi-line strings, while scc counts every line inside a string as code. gdloc's behavior follows the general rule above.

GDScript results match exactly on two other projects:

| Project | Files | Lines | Code | Comments | Blanks |
|---|---|---|---|---|---|
| `D:\Godot\astral-mason` | 25 | 4,511 | 2,371 | 1,196 | 944 |
| `D:\Godot\flipped-sky` | 21 | 2,308 | 1,002 | 842 | 464 |

### Shader

Sample: `D:\WindupWonderland\src` (17 `.gdshader` / `.gdshaderinc` files, no `.glsl`):

| Project | Files | Lines | Code | Comments | Blanks |
|---|---|---|---|---|---|
| `D:\WindupWonderland\src` | 17 | 1,358 | 891 | 287 | 180 |
| `D:\Godot\astral-mason` | 3 | 247 | 130 | 67 | 50 |
| `D:\Godot\flipped-sky` | 0 | 0 | 0 | 0 | 0 |

gdloc's Shader group matches scc's GLSL group with zero per-file differences. scc's GLSL group contains exactly the `.gdshader` / `.gdshaderinc` files; the local scc 3.7.0 does not recognize these extensions, so for the comparison the samples were copied to a temporary directory and renamed to `.glsl`. None of the three projects contain native `.glsl` files.

### Scene / Resource

scc's "Godot Scene" **only counts `.tscn`** (Windup has 12 `.tres` files that it does not count). Files and Lines compared:

| Project | gdloc Scene Files | scc Godot Scene Files | gdloc Scene Lines | scc Lines |
|---|---|---|---|---|
| `D:\WindupWonderland\src` | 35 | 35 | 3,210 | 3,210 |
| `D:\Godot\astral-mason` | 19 | 19 | 4,657 | 4,657 |
| `D:\Godot\flipped-sky` | 15 | 15 | 3,254 | 3,254 |

`.tscn` Files and Lines match exactly. `.tres` has no scc counterpart (scc does not count it as Godot Scene); gdloc lists it separately as Resource: Windup `12 / 187`, 0 for the other two. None of the three projects contain embedded GDScript / Shader (`script/source` and `type="Shader"` have zero matches), so no `(embedded)` rows appear.

## Releasing

Pushing a `v*` tag (e.g. `v0.1.0`) makes GitHub Actions run GoReleaser, which builds amd64 and arm64 binaries for Windows / macOS / Linux, publishes them to Releases, and pushes the Scoop manifest to [scoop-bucket](https://github.com/qwqzhanqwq/scoop-bucket) (requires the repository secret `SCOOP_BUCKET_TOKEN`). See `.goreleaser.yaml` and `.github/workflows/release.yml`.

## License

MIT, see [LICENSE](LICENSE).
