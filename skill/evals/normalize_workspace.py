#!/usr/bin/env python3
"""Normalize a skill-creator iteration directory into the layout its tools expect.

Background: the eval subagents were told to write into
`<iteration>/eval-N-name/<config>/outputs/`, which is the natural layout for a
human. skill-creator's aggregate_benchmark.py and generate_review.py, however,
expect each run to live in a `run-N/` directory (each with its own `outputs/`)
and to have `grading.json` next to that `outputs/`. This script moves things
into place:

    eval-N-name/
      eval_metadata.json        <- {"eval_id", "eval_name", "prompt", "assertions"}
      with_skill/run-1/{outputs/,grading.json,timing.json}
      without_skill/run-1/{outputs/,grading.json,timing.json}

Idempotent: already-normalized runs are left alone. Run it before grading.

Usage: python normalize_workspace.py <iteration-dir> [--evals-json evals.json]
"""

import argparse
import json
import os
import shutil
import sys

CONFIGS = ("with_skill", "without_skill")


def normalize(iteration, evals_json):
    # Pull prompts/assertions from evals.json so the viewer has something to
    # show even if the run directories were created before the evals existed.
    by_name = {}
    if evals_json and os.path.exists(evals_json):
        with open(evals_json, encoding="utf-8") as fh:
            data = json.load(fh)
        for e in data.get("evals", []):
            by_name[e.get("name", "")] = e

    moved = []
    for entry in sorted(os.listdir(iteration)):
        eval_dir = os.path.join(iteration, entry)
        if not os.path.isdir(eval_dir) or not entry.startswith("eval-"):
            continue

        meta_path = os.path.join(eval_dir, "eval_metadata.json")
        meta = {}
        if os.path.exists(meta_path):
            with open(meta_path, encoding="utf-8") as fh:
                meta = json.load(fh)

        name = entry.split("-", 2)[2] if entry.count("-") >= 2 else entry
        try:
            eval_id = int(entry.split("-")[1])
        except (ValueError, IndexError):
            eval_id = 0
        source = by_name.get(name, {})

        meta.setdefault("eval_id", eval_id)
        meta.setdefault("eval_name", name)
        meta.setdefault("prompt", source.get("prompt", ""))
        meta.setdefault("assertions", source.get("assertions", []))
        meta.setdefault("expected_output", source.get("expected_output", ""))
        with open(meta_path, "w", encoding="utf-8") as fh:
            json.dump(meta, fh, indent=2, ensure_ascii=False)

        for config in CONFIGS:
            config_dir = os.path.join(eval_dir, config)
            if not os.path.isdir(config_dir):
                continue
            run_dir = os.path.join(config_dir, "run-1")
            outputs = os.path.join(config_dir, "outputs")
            if not os.path.isdir(outputs):
                continue
            if os.path.isdir(os.path.join(run_dir, "outputs")):
                moved.append("%s/%s already normalized" % (entry, config))
                continue
            os.makedirs(run_dir, exist_ok=True)
            shutil.move(outputs, os.path.join(run_dir, "outputs"))
            for extra in ("grading.json", "timing.json"):
                src = os.path.join(config_dir, extra)
                if os.path.exists(src):
                    shutil.move(src, os.path.join(run_dir, extra))
            moved.append("%s/%s -> run-1" % (entry, config))

    for line in moved:
        print(line)
    return 0


def main():
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("iteration", help="iteration directory, e.g. .../iteration-1")
    ap.add_argument("--evals-json", default=None, help="path to evals.json for prompts/assertions")
    args = ap.parse_args()

    if not os.path.isdir(args.iteration):
        print("error: %s is not a directory" % args.iteration, file=sys.stderr)
        return 2
    return normalize(os.path.abspath(args.iteration), args.evals_json)


if __name__ == "__main__":
    sys.exit(main())
