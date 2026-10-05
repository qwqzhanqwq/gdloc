#!/usr/bin/env python3
"""Compress `gdloc --json` output into a compact, readable summary.

Why this exists: on a real project the JSON can be tens of thousands of lines,
and reading all of it into an agent's context is wasteful. This prints the parts
that usually matter (totals, ratios, biggest files/functions, addon split).

Usage:
    gdloc --json --by-file --top 0 > report.json
    gdloc --json --stats    --top 0 > stats.json
    python gdloc_summary.py report.json stats.json [--rows 15]

Notes:
 - `--stats` cannot be combined with `--by-file`, so run gdloc once per view.
 - Labels are intentionally ASCII: the script's stdout often travels through a
   Windows console whose code page cannot represent every character, and broken
   output is worse than English labels. The numbers and paths are what matter.
 - `--top` truncates arrays (total stays complete), so rankings here only cover
   what the JSON actually contains.
"""

import argparse
import json
import sys

# ---------------------------------------------------------------------------
# Output helpers
# ---------------------------------------------------------------------------


def emit(*parts):
    """Write a line that survives consoles with a non-UTF-8 code page."""
    text = " ".join(str(p) for p in parts)
    try:
        print(text)
    except UnicodeEncodeError:
        enc = getattr(sys.stdout, "encoding", None) or "ascii"
        print(text.encode(enc, "replace").decode(enc, "replace"))


def fmt(n):
    return format(int(n), ",")


def pct(part, whole):
    if not whole:
        return "0.0%"
    return "%.1f%%" % (100.0 * part / whole)


def section(title):
    emit("")
    emit(title)
    emit("-" * max(len(title), 8))


def table(headers, rows):
    """Print a left-aligned table sized to its content."""
    cells = [[str(c) for c in row] for row in rows]
    widths = [len(h) for h in headers]
    for row in cells:
        for i, cell in enumerate(row):
            if i < len(widths):
                widths[i] = max(widths[i], len(cell))
    emit("  ".join(h.ljust(widths[i]) for i, h in enumerate(headers)).rstrip())
    for row in cells:
        emit("  ".join(c.ljust(widths[i]) for i, c in enumerate(row)).rstrip())


# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------


def summarize(rep, rows):
    emit("Project: %s" % (rep.get("project_name") or "(no project.godot found)"))
    emit("Root:    %s" % rep.get("root", "?"))

    langs = rep.get("languages") or []
    total = rep.get("total") or {}
    total_code = total.get("code", 0)
    total_lines = total.get("lines", 0)

    section("By language")
    if langs:
        table(
            ["Language", "Files", "Lines", "Code", "Comments", "Doc", "Blanks", "Comment%"],
            [
                [
                    l["language"],
                    fmt(l.get("files", 0)),
                    fmt(l.get("lines", 0)),
                    fmt(l.get("code", 0)),
                    fmt(l.get("comments", 0)),
                    fmt(l.get("doc", 0)),
                    fmt(l.get("blanks", 0)),
                    pct(l.get("comments", 0), l.get("lines", 0)),
                ]
                for l in langs
            ],
        )
    else:
        emit("(no countable code files found)")

    emit("")
    emit(
        "Total: %s files, %s lines | code %s, comments %s, blanks %s | comment %s of lines, code %s"
        % (
            fmt(total.get("files", 0)),
            fmt(total_lines),
            fmt(total_code),
            fmt(total.get("comments", 0)),
            fmt(total.get("blanks", 0)),
            pct(total.get("comments", 0), total_lines),
            pct(total_code, total_lines),
        )
    )

    scenes = rep.get("scenes") or {}
    resources = rep.get("resources") or {}
    emit(
        "Not in Total: Scene %s files / %s lines, Resource %s files / %s lines"
        % (
            fmt(scenes.get("files", 0)),
            fmt(scenes.get("lines", 0)),
            fmt(resources.get("files", 0)),
            fmt(resources.get("lines", 0)),
        )
    )
    if rep.get("visual_shaders"):
        emit("Not counted: %s VisualShader resource(s)" % fmt(rep["visual_shaders"]))

    files = rep.get("files") or []
    if files:
        section("Biggest files by code (top %d)" % rows)
        ranked = sorted(files, key=lambda f: f.get("code", 0), reverse=True)
        table(
            ["Path", "Language", "Code", "Lines", "OfCode"],
            [
                [
                    f["path"],
                    f.get("language", ""),
                    fmt(f.get("code", 0)),
                    fmt(f.get("lines", 0)),
                    pct(f.get("code", 0), total_code),
                ]
                for f in ranked[:rows]
            ],
        )
        embedded = [f for f in files if "(embedded)" in f.get("language", "")]
        if embedded:
            emit("")
            emit(
                "  of which %d embedded block(s), %s code lines"
                % (len(embedded), fmt(sum(f.get("code", 0) for f in embedded)))
            )

    addons = rep.get("addons") or []
    if addons:
        section("By addon")
        table(
            ["Addon", "Version", "Files", "Code", "OfCode", "Note"],
            [
                [
                    a["name"],
                    a.get("version", ""),
                    fmt(a.get("files", 0)),
                    fmt(a.get("code", 0)),
                    pct(a.get("code", 0), total_code),
                    # "(project)" is the non-plugin bucket, so the plugin.cfg
                    # note would be misleading there.
                    "" if a["name"] == "(project)" or a.get("has_plugin_cfg") else "no plugin.cfg",
                ]
                for a in addons
            ],
        )
        own = [a for a in addons if a["name"] == "(project)"]
        if own and total_code:
            own_code = own[0].get("code", 0)
            emit("")
            emit(
                "  own code %s (%s), third-party addons %s (%s)"
                % (
                    fmt(own_code),
                    pct(own_code, total_code),
                    fmt(total_code - own_code),
                    pct(total_code - own_code, total_code),
                )
            )

    dirs = rep.get("dirs") or []
    if dirs:
        section("By top-level directory")
        table(
            ["Group", "Files", "Code", "OfCode"],
            [[d["name"], fmt(d.get("files", 0)), fmt(d.get("code", 0)), pct(d.get("code", 0), total_code)] for d in dirs],
        )

    stats = rep.get("stats") or {}
    if stats:
        struct = stats.get("structure") or {}
        if struct:
            section("GDScript structure")
            emit(
                "func %s | signal %s | class_name %s | @export %s"
                % (
                    fmt(struct.get("funcs", 0)),
                    fmt(struct.get("signals", 0)),
                    fmt(struct.get("class_names", 0)),
                    fmt(struct.get("exports", 0)),
                )
            )

        suspicious = [l for l in stats.get("languages") or [] if l.get("suspected")]
        if suspicious:
            section("Suspected commented-out code (heuristic estimate)")
            table(
                ["Language", "Comments", "Suspected", "OfComments"],
                [
                    [
                        l["language"],
                        fmt(l.get("comments", 0)),
                        fmt(l.get("suspected", 0)),
                        pct(l.get("suspected", 0), l.get("comments", 0)),
                    ]
                    for l in suspicious
                ],
            )

        funcs = stats.get("longest_functions") or []
        if funcs:
            section("Longest GDScript functions (top %d)" % rows)
            table(
                ["Location", "Function", "Code"],
                [["%s:%s" % (f["path"], f.get("line", 0)), f.get("name", ""), fmt(f.get("length", 0))] for f in funcs[:rows]],
            )

    section("Caveats")
    emit("- Scene/Resource lines are NOT part of Total.")
    emit("- Doc is a subset of Comments, do not add them together.")
    emit("- Arrays may be truncated by --top; Total is always complete.")
    emit("- Suspected commented-out code is an estimate, not a verdict.")


def main():
    ap = argparse.ArgumentParser(description="Compress gdloc --json output into a summary")
    ap.add_argument("reports", nargs="+", help="JSON files written by `gdloc --json`")
    ap.add_argument("--rows", type=int, default=10, help="ranking length (default 10)")
    args = ap.parse_args()

    try:
        sys.stdout.reconfigure(errors="replace")
    except (AttributeError, ValueError):
        pass

    for i, path in enumerate(args.reports):
        try:
            with open(path, "r", encoding="utf-8") as fh:
                rep = json.load(fh)
        except OSError as err:
            emit("error: cannot read %s: %s" % (path, err))
            return 2
        except ValueError as err:
            emit("error: %s is not valid JSON (%s)" % (path, err))
            emit("hint: gdloc writes plain UTF-8; on Windows PowerShell redirecting with `>`")
            emit("      produces UTF-16. Use cmd: gdloc --json > report.json")
            return 2
        if i:
            emit("")
            emit("=" * 72)
        summarize(rep, args.rows)

    sys.stdout.flush()
    return 0


if __name__ == "__main__":
    sys.exit(main())
