#!/usr/bin/env python3
"""Patch timing.json for each run with the objective data we can still measure.

Token counts and wall-clock durations arrive only in the completion notification
of an eval subagent, and this session did not capture them. What survives on disk
is the size of each run's outputs, which is a usable (if crude) proxy for how much
work a configuration produced.

Writes timing.json next to each grading.json; existing keys are preserved, so a
later run that does capture total_tokens/duration_ms can be merged in.

Usage: python patch_timing.py <iteration-dir>
"""

import argparse
import json
import os
import sys

CONFIGS = ("with_skill", "without_skill")


def main():
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("iteration", help="iteration directory")
    args = ap.parse_args()

    iteration = os.path.abspath(args.iteration)
    if not os.path.isdir(iteration):
        print("error: %s is not a directory" % iteration, file=sys.stderr)
        return 2

    patched = 0
    for entry in sorted(os.listdir(iteration)):
        eval_dir = os.path.join(iteration, entry)
        if not os.path.isdir(eval_dir) or not entry.startswith("eval-"):
            continue
        try:
            eval_id = int(entry.split("-")[1])
        except (ValueError, IndexError):
            eval_id = 0

        for config in CONFIGS:
            run_dir = os.path.join(eval_dir, config, "run-1")
            outputs = os.path.join(run_dir, "outputs")
            if not os.path.isdir(outputs):
                continue

            total_chars = 0
            files = []
            for root, _dirs, names in os.walk(outputs):
                if "__pycache__" in root:
                    continue
                for name in sorted(names):
                    path = os.path.join(root, name)
                    try:
                        size = os.path.getsize(path)
                    except OSError:
                        continue
                    total_chars += size
                    files.append(os.path.relpath(path, outputs).replace("\\", "/"))

            timing_path = os.path.join(run_dir, "timing.json")
            data = {}
            if os.path.exists(timing_path):
                try:
                    with open(timing_path, encoding="utf-8") as fh:
                        data = json.load(fh)
                except (OSError, ValueError):
                    data = {}

            data.setdefault("eval_id", eval_id)
            data.setdefault("eval_name", entry.split("-", 2)[2] if entry.count("-") >= 2 else entry)
            data.setdefault("configuration", config)
            data["output_chars"] = total_chars
            data["output_files"] = files
            data["tokens"] = data.get("total_tokens", 0)

            with open(timing_path, "w", encoding="utf-8") as fh:
                json.dump(data, fh, indent=2, ensure_ascii=False)
            patched += 1
            print("%s/%s: %d files, %d output chars" % (entry, config, len(files), total_chars))

    print("patched %d run(s)" % patched)
    return 0


if __name__ == "__main__":
    sys.exit(main())
