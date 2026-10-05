---
name: gdloc
description: Count and analyze the code size of a Godot 4 project with the gdloc CLI — lines of code by language, file, top-level directory or addon, plus doc-comment ratios, function/signal counts, longest files and functions, and suspected commented-out code. Use this skill whenever the user asks how large a Godot project is, how many lines of code it has, which scripts or functions are the largest, how much of the code sits in addons/third-party plugins, how many functions/signals/exports there are, or how much of the codebase is commented out — in English or Chinese (统计行数、项目多大、哪个文件最长、插件占多少、有多少函数、多少注释被注释掉了) — even if they never say "gdloc", and even if they only want two Godot projects compared. Do not use it for API questions (use godot-doc-lookup) or for design review (use godot-architecture-review).
---

# gdloc: counting lines in a Godot 4 project

gdloc is a command-line tool that measures how much code a Godot 4 project contains. It understands how Godot projects are laid out: it reads the project name from `project.godot`, recognizes plugins under `addons/`, pulls GDScript and Shader code embedded in `.tscn` / `.tres` files out into their own counts, and honors `.gdignore` and `.gitignore`.

It is read-only on the scanned directory and does not touch the network, so running it is always safe.

The authoritative statement of what gets counted is the gdloc repository README — `README.md` (English, default) and `README.zh-CN.md` (简体中文), which are kept in sync. The full counting rules, the priority order for classifying each line, and the known differences from scc all live there. Read it when you need to confirm an edge case instead of reasoning from first principles.

## When this skill applies

Questions gdloc is the right tool for:

- How big is this project? How many lines, how many files, what share is comments?
- Which scripts or functions are the longest, and where should refactoring start?
- How much of the code is third-party plugins (`addons/`) versus written by the user?
- How many `func` / `signal` / `class_name` / `@export` are there?
- How many comments are actually commented-out code?
- How do two Godot projects compare in size?

It is the wrong tool when the user is asking about C# or any other language (gdloc does not count `.cs`), when the question is about Godot API usage or code design (other skills cover that), or when they just want one file's line count — reading the file is faster.

## Workflow

### 1. Work out what is actually being asked

gdloc's output modes answer different questions, and picking the wrong one produces an answer to something the user did not ask:

| The user wants | Use |
|---|---|
| Overall project size, comment ratio | the default (no flags), grouped by language |
| How long one file or one function is | add `--by-file` or `--stats` |
| Which files/functions are longest | `--stats` (includes longest-file and longest-function rankings) |
| Counts of func / signal / @export | `--stats` |
| How much commented-out code exists | `--stats` |
| How code is distributed across directories | `--by-dir` |
| How much comes from third-party plugins | `--by-addon` (pair with `--exclude-addons` to see the user's own code) |
| Numbers that will be post-processed | `--json` |

When unsure, run the default view first and add a grouped view afterwards — gdloc is fast, and running it twice is cheaper than guessing at the right basis.

### 2. Confirm which directory to scan

This is the step that goes wrong most often. gdloc's scan root is exactly the directory you pass it; it does not pick out Godot files, it counts every file it recognizes under that root. So:

- The project root is the directory containing `project.godot`. Plugin detection (`addons/`) is anchored there even when the scan root is a subdirectory.
- If the repository keeps `project.godot` in a subdirectory (`src/`, `game/`), the scan root must be that subdirectory. Scanning one level off yields a different set of numbers that looks perfectly plausible.
- When the user names a project but gives no path, look in the usual development locations (on Windows that is often `D:\Godot\`, `C:\Users\<user>\Godot\`, or wherever they have mentioned before). Before reporting, confirm the target directory holds a `project.godot` or that the scan was not empty — reporting a confident "project size" for a directory of unrelated files is the worst possible outcome here.

### 3. Run it

```sh
gdloc [path] [options]
```

Options may come before or after the path. The combinations that cover most requests:

```sh
gdloc .                                  # per-language summary (default)
gdloc . --by-file --top 20 --sort code   # per file, 20 files with the most code
gdloc . --by-dir                         # per top-level directory
gdloc . --by-addon                       # per plugin; non-plugin code lands in (project)
gdloc . --stats                          # advanced statistics
gdloc . --stats --exclude-addons         # advanced statistics for the user's own code only
gdloc . --json --stats --top 20          # advanced statistics as JSON, rankings of 20
```

Options worth adding only when the question calls for them: `--exclude-addons` (ignore the whole `addons/` directory), `--exclude-dir a,b` (exclude directory names at any depth), `--no-ignore` (skip `.gitignore`), `--sort <code|comments|blanks|lines|files>` (default `code`, descending), `--top N` (show only the first N rows).

Check the tool exists first with `gdloc --version`. If it is missing, tell the user how to install it (`go install github.com/qwqzhanqwq/gdloc/cmd/gdloc@latest`, a download from GitHub Releases, or `scoop install gdloc` on Windows). Do not hand-roll a counting script to work around it — gdloc's line classification is far more careful than an improvised one.

On a large project, write `--json` to a file and read that, or cut the output down with `--top`, rather than pulling a whole table into context. The JSON field structure does not need to be guessed: the README's "JSON format" section has a complete example. When aggregating, parse the JSON with a real tool; do not scrape the table with regexes.

There is one Windows trap worth knowing: in PowerShell, `gdloc . --json > report.json` writes **UTF-16** because of how PowerShell redirects, and reading it back as UTF-8 fails outright. Use `cmd /c "gdloc . --json > report.json"` (or `gdloc . --json | Out-File -Encoding utf8 report.json`) to get UTF-8. Mojibake content, or an "invalid start byte" error while parsing, is almost always this.

### 4. Read the numbers correctly

These are the places where reports most often go wrong:

- **Total sums code languages only.** Scene (`.tscn`) and Resource (`.tres`) appear below the separator with Files and Lines only, and are **not part of Total**. Adding them into "how many lines in total" is simply wrong.
- **`GDScript (embedded)` / `Shader (embedded)` is code embedded in `.tscn` / `.tres` files.** It gets its own row but **does count toward Total**. For those rows, Files is the number of files containing embedded code, not the number of blocks; the block count is the `blocks` JSON field, and `--by-file` lists each one as `file::id` (`file::[resource]` for a main resource).
- **Doc is a subset of Comments**, not a sibling category. Use Comments as the denominator for comment ratios and never add Doc on top of it.
- **`.cs` files and VisualShader resources are not line-counted at all** — they are only reported as a count below the table. If the user asks for total project size and `.cs` files exist, say that this part was not counted.
- `--stats` rankings show 10 entries by default; `--top` controls that. For `longest_functions`, `length` is the number of code lines in the function body (blanks and comments excluded), not the span between start and end line numbers.
- `--stats` comment analysis **folds embedded code into the `GDScript` / `Shader` names**, while the table lists embedded code separately as `(embedded)` rows. In any project whose `.tscn` / `.tres` files contain embedded code, the same metric will differ between the two views — by exactly the embedded portion. That is not a mistake on your part. Compute consistently within one view rather than mixing them.
- **State the scope behind every number.** `--stats` (comment analysis, structure, longest files and functions) covers everything in the scan; `--exclude-addons` is what narrows it to the user's own code, and `--by-addon` gives group totals but no structure statistics. So questions like "how many functions do I have", "what is my comment ratio", or "what is my longest function" must be answered from a run that included `--exclude-addons`, or phrased explicitly as whole-project including plugins. Presenting plugin-inclusive statistics as the user's own code is the single easiest mistake to make here, and the user is unlikely to catch it — note in particular that the top of the `longest_functions` ranking is usually files under `addons/`, so do not read that ranking out as the user's code.

### 5. Report back

State the conclusion plainly; do not paste gdloc's table wholesale, since the user can run it themselves. Worth including:

- The exact command and the scan root (the user needs this to judge whether you measured the project they meant).
- The numbers that answer the question, with thousands separators, in tables for anything ranked.
- Observations that carry meaning, e.g. "`addons/` accounts for 62% of the code; your own code is only NNN lines", "comment ratio is 9%, on the low side", "the top 3 files hold 40% of all code, start there".
- Caveats only when they affect the conclusion — for example `.cs` files present, or a user who may assume Scene lines are included.

## Pitfalls

- `--by-file`, `--by-dir` and `--by-addon` are mutually exclusive; combining them exits with an error. `--exclude-addons` cannot be combined with `--by-addon`, and `--stats` cannot be combined with any of the three grouping flags. To see two views, run gdloc twice.
- `--top N` truncates the displayed rows only; **Total always covers everything**. So "the top 10 files summed" is not a share of Total — do not use it as a denominator.
- `.gitignore` is read **from the scan root downward only**; rules above the root do not apply. Running from a deep subdirectory silently loses the rules above it — ask the user before reaching for `--no-ignore`.
- Directories starting with `.` (`.godot/`, `.git/`) are skipped by default, as are directories containing `.gdignore` and their entire subtree.
- Non-zero exit codes: `1` invalid arguments, `2` path missing or unreadable. `--help` writes usage to stderr, which looks like a failure in PowerShell but is not one.
- Suspected "commented-out code" in `--stats` **is a heuristic estimate, not a verdict** (comments containing CJK characters, or starting with `TODO`/`FIXME`, never count). Always label it as an estimate when reporting it.
- gdloc counts GDScript and Godot Shader only. Asking it how large a Go, Python or C# project is has no meaning; use another tool (scc, tokei) for those.

## Differences from scc

When the user compares against scc and the numbers disagree, the causes are known and documented in the README's "Differences from scc" section: whitespace-only lines inside multi-line strings count as blank in gdloc but as code in scc; scc's "Godot Scene" covers `.tscn` but not `.tres`; and scc 3.7.0 does not recognize `.gdshader` / `.gdshaderinc`, so it either files them under GLSL or misses them. Quote the README rather than inventing an explanation on the spot.

## Bundled files

- `scripts/gdloc_summary.py` condenses `gdloc --json` output into a compact summary (per-language shares, biggest files, addon split, longest functions). Use it on large projects where reading the whole JSON into context is wasteful. `--stats` and `--by-file` are mutually exclusive, so export twice:

  ```sh
  cmd /c "gdloc . --json --by-file --top 0 > files.json"
  cmd /c "gdloc . --json --stats    --top 0 > stats.json"
  python scripts/gdloc_summary.py files.json stats.json --rows 15
  ```

  The script's labels are ASCII because Windows consoles frequently cannot represent every character; the numbers and paths are the point. On a small project, read `--json` or the table directly and skip this layer.
