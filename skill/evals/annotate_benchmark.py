#!/usr/bin/env python3
"""Add the analyst pass notes to iteration-1's benchmark.json.

aggregate_benchmark.py always writes an empty notes array; the observations below
come from reading the graded runs and the raw outputs, not from the aggregate
statistics (which were uninformative here because both configurations passed).

Usage: python annotate_benchmark.py <iteration-dir>
"""

import argparse
import json
import os
import sys

NOTES = [
    "DISCRIMINATION: both configurations scored 100% (16/16 assertions). The iteration-1 "
    "assertion set only checked whether the right numbers appeared somewhere, and an agent "
    "with just the gdloc binary on PATH (no skill) can get those numbers right. Treat the "
    "pass-rate delta as uninformative for this iteration.",
    "What the skill did change is the process: the three with_skill runs produced 22 files / "
    "96 KB of output, the three baselines 24 files / 130 KB, because two baselines "
    "reimplemented a line counter in Python (string and comment handling, triple-quoted "
    "GDScript, tscn embedded code) instead of reading gdloc's output. The with_skill runs "
    "called the bundled scripts/gdloc_summary.py instead.",
    "REAL ERRORS the assertions missed (with_skill): eval-2 presented the --stats "
    "longest-function ranking as the user's own code although the top entries are all under "
    "addons/; eval-3 misstated the excluded-line breakdown (vendor/ignored.gd is 489 "
    "lines, not 424). Both are scope-reading mistakes of the kind SKILL.md now covers.",
    "REAL ERRORS the assertions missed (baseline): eval-2 claimed --by-file --top 0 "
    "truncates the list to 15 rows (it does not; --top 0 means no truncation); eval-3 "
    "reported 1,405 code lines for Windup, 14 short, by leaving embedded GDScript/Shader "
    "out of Total, which the +/-2% tolerance let pass.",
    "WEAK ASSERTIONS for iteration 2: the +/-2% tolerance on Total (1,419) lets an answer "
    "that omits embedded code (1,405) pass; nothing asserted that Doc is a subset of "
    "Comments or that embedded code belongs in Total; nothing asserted the 'your own code' "
    "scope of --stats-derived numbers, which is exactly where the with_skill runs erred. "
    "The rewritten evals.json tightens all three.",
    "TIMING/TOKENS UNAVAILABLE: subagent completion notifications did not carry "
    "total_tokens/duration_ms into this session, so Time and Tokens are 0. The "
    "output_chars field in each timing.json is the measurable proxy.",
]


def main():
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("iteration")
    args = ap.parse_args()

    path = os.path.join(os.path.abspath(args.iteration), "benchmark.json")
    if not os.path.exists(path):
        print("error: %s not found; run aggregate_benchmark first" % path, file=sys.stderr)
        return 2

    with open(path, encoding="utf-8") as fh:
        data = json.load(fh)
    data["notes"] = NOTES
    with open(path, "w", encoding="utf-8") as fh:
        json.dump(data, fh, indent=2, ensure_ascii=False)
    print("wrote %d notes into %s" % (len(NOTES), path))
    return 0


if __name__ == "__main__":
    sys.exit(main())
