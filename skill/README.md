# gdloc skill

给编码 agent（Claude Code、DSH 等）用的 gdloc skill。SKILL.md 用英文写，内容是"怎么用 gdloc 回答用户的问题、以及怎么不把数字读错"——统计口径的权威说明仍然在仓库根目录的 README 里，两边不要互相复制。frontmatter 的 `description` 是中英混排的：触发判断只看这一行，中文提问（"我这个项目多大了"）也必须能命中，所以中文触发词要留在里面。

```
gdloc/                  ← 安装到 agent 的 skills 目录的就是这个目录
├── SKILL.md              # 英文正文；frontmatter 的 description 含中文触发词
└── scripts/
    └── gdloc_summary.py
evals/
├── make_fixtures.py        # 生成测试用的假 Godot 工程（固定随机种子）
├── evals.json              # 测试用例与断言（中文，对应中文测试提问）
├── normalize_workspace.py  # 把评测产物整理成 skill-creator 需要的目录结构
├── patch_timing.py         # 把产出体积等可测数据写进 timing.json
└── annotate_benchmark.py   # 把分析结论写进 benchmark.json 的 notes
gdloc-workspace/            # 评测产物（.gitignore 掉，不入库）
```

## 安装

把 `gdloc` 目录放进 agent 读取 skill 的位置。Claude Code 与 DSH 用 `~/.agents/skills/`（Windows 上是 `%USERPROFILE%\.agents\skills\`）。用 junction / 符号链接指向本目录，可以保持与仓库同步：

```powershell
New-Item -ItemType Junction -Path "$env:USERPROFILE\.agents\skills\gdloc" -Target "D:\Code\gdloc\skill\gdloc"
```

## 重新跑评测

需要本机已装 `gdloc`（`gdloc --version` 能跑通）和 Python 3。所有脚本只用标准库。

```powershell
# 1. 造测试工程（放在中立路径下，避免 agent 从路径名猜到答案）
python skill\evals\make_fixtures.py "$env:TEMP\gdloc-eval-iter2"

# 2. 让被测 agent 在真实场景里跑（with_skill 读 SKILL.md，without_skill 只给 gdloc 二进制）
#    产物写到 skill\gdloc-workspace\iteration-2\eval-N-<名字>\<config>\outputs\

# 3. 整理目录结构 + 补可测的 timing 数据
python skill\evals\normalize_workspace.py skill\gdloc-workspace\iteration-2 --evals-json skill\evals\evals.json
python skill\evals\patch_timing.py skill\gdloc-workspace\iteration-2

# 4. 逐条断言打分，结果写成每个 run 的 grading.json

# 5. 汇总并打开评审页面
cd "$env:USERPROFILE\.agents\skills\skill-creator"
python -X utf8 -m scripts.aggregate_benchmark <iteration-2 路径> --skill-name gdloc
python -X utf8 eval-viewer\generate_review.py <iteration-2 路径> --skill-name gdloc --benchmark <iteration-2 路径>\benchmark.json
```

Windows 上的两个坑：`quick_validate.py` 之类的脚本用系统默认编码读文件，含非 ASCII 的 SKILL.md 要加 `-X utf8`；生成 fixture 时 python 的 `hash()` 每次进程都变，所以 `make_fixtures.py` 里用的是自算的固定 uid。

## 已经知道的评测结论（iteration-1）

- 断言区分度不足：3 个用例、16 条断言，with_skill 与 without_skill 都是 100%。原断言只查"数字对不对"，而只装了 gdloc 的 agent 也能把数字查对。iteration-2 起改用更严的断言（口径陷阱、`--stats` 作用域、该不该用 gdloc）。
- 真正区分出 skill 价值的地方在过程与产出体积：带 skill 的 3 次运行共产出 22 个文件 / 96 KB，基线 3 次产出 24 个文件 / 130 KB。基线为了得到同样的结论，自己写 Python 脚本重新实现了一遍逐行分类（GDScript 字符串、三引号、Shader 块注释、tscn 内嵌代码），而带 skill 的运行直接调用了 `scripts/gdloc_summary.py`。
- 带 skill 的运行仍然犯过两类错，都已经写进 SKILL.md：把 `--stats` 全项目口径的统计（func 151 / 注释 47）说成"你自己代码的"；把 `--stats` 最长函数榜里 `addons/` 下的函数说成用户自己的函数。
- 顺带发现 gdloc 自身的一处不一致：`--stats` 的注释分析把内嵌代码并进 `GDScript` / `Shader`（47 / 19），表格则单列为 `(embedded)` 行（46 / 17）。两视图差额恰好等于内嵌部分，已在 SKILL.md 中说明；未改动工具代码。

## 语言

SKILL.md 正文是英文的，`description` 采用中英混排：这一行是触发判断的唯一依据，中文提问（"我这个项目多大了"）也必须命中，所以正文英文化之后仍要保留中文触发词。iteration-1 的评测用的是中文提问（`evals.json` 里的 prompt 与断言也是中文），被测的是当时的中文版 SKILL.md；正文改成英文后若要重跑评测，可以沿用这批中文 prompt（正好顺带验证 description 里的中文触发词够不够用），也可以再补一组英文 prompt 作为对照。
