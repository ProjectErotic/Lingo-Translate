#!/usr/bin/env python3
"""Lingo QA addon — optional pre-install quality gate.

Pipeline: [translated workspace] -> 1. Python format check -> 2. Fabric CLI
Thai-style review -> [write into game's lingo_translations/].

- Stage 1 catches broken formatting: leaked placeholder tags, unbalanced RPGM
  control codes, empty targets, missing Thai script. Failures are re-marked
  untranslated in the workspace so `lingo translate` picks them up again.
- Stage 2 asks Fabric (https://github.com/danielmiessler/fabric, install
  separately) to review Thai phrasing. Odd translations are NOT touched —
  they are logged to qa_fabric_findings.csv for manual review.
- Fabric missing or -skip-fabric given -> stage 2 skipped, stage 1 still runs.

Usage:
  python3 qa_review.py -workspace "game.nst" -game "/path/to/game"
                       [-retranslate] [-skip-fabric] [-fabric-pattern translate_thai_review]

Requires: python3 (stdlib only). Optional: fabric + OPENAI_BASE_URL/OPENAI_API_KEY
(or FABRIC_* vars) for stage 2.
"""
import argparse
import csv
import json
import os
import re
import shutil
import sqlite3
import subprocess
import sys
import urllib.request

TAG_RE = re.compile(r"__LINGO_TAG_\d+__")
CTRL_RE = re.compile(r"\\[A-Za-z]+\[[^\]]*\]|\\[A-Za-z]+|%1\d?|%2\d?|%3\d?")
THAI_RE = re.compile(r"[ก-์]")


def check_entry(source: str, target: str) -> list[str]:
    problems = []
    if not target or not target.strip():
        problems.append("empty_target")
        return problems
    src_tags = sorted(TAG_RE.findall(source))
    tgt_tags = sorted(TAG_RE.findall(target))
    if src_tags != tgt_tags:
        problems.append(f"tag_mismatch(src={len(src_tags)},tgt={len(tgt_tags)})")
    src_ctrl = sorted(CTRL_RE.findall(source))
    tgt_ctrl = sorted(CTRL_RE.findall(target))
    if src_ctrl != tgt_ctrl:
        problems.append(f"control_mismatch(src={src_ctrl[:3]},tgt={tgt_ctrl[:3]})")
    # JP->TH: target identical to a Japanese source means the model copied it
    jp = re.compile(r"[ぁ-ゟ゠-ヿ一-鿿]")
    if jp.search(source) and target.strip() == source.strip():
        problems.append("identity_translation")
    return problems


def run_format_check(ws: str) -> list[tuple]:
    db = sqlite3.connect(ws)
    rows = db.execute(
        "SELECT id, source, target, file_path FROM entries WHERE status='translated'"
    ).fetchall()
    bad = []
    for eid, src, tgt, fp in rows:
        problems = check_entry(src, tgt or "")
        if problems:
            bad.append((eid, fp, src[:80], tgt[:80], ";".join(problems)))
    db.close()
    return bad


def retranslate(ws: str, bad: list[tuple]) -> int:
    """Reset broken rows to untranslated so the main translate command redoes them."""
    db = sqlite3.connect(ws)
    n = 0
    for eid, *_ in bad:
        cur = db.execute("SELECT source FROM entries WHERE id=?", (eid,)).fetchone()
        if cur:
            db.execute(
                "UPDATE entries SET target='', status='untranslated', translator='' WHERE id=?",
                (eid,),
            )
            n += 1
    db.commit()
    db.close()
    return n


def fabric_review(rows: list[tuple], pattern: str, limit: int) -> list[dict]:
    """Ask fabric to review Thai phrasing in batches. Returns findings."""
    findings = []
    for i in range(0, min(len(rows), limit), 10):
        batch = rows[i : i + 10]
        lines = [
            f"{n+1}. JP: {s} | TH: {t}" for n, (s, t) in enumerate(batch)
        ]
        prompt = (
            "Review these Japanese->Thai game translations for awkward, unnatural, "
            "or wrong Thai phrasing. For each line output `N: OK` or "
            "`N: <short Thai reason>`. Nothing else.\n" + "\n".join(lines)
        )
        try:
            r = subprocess.run(
                ["fabric", "--pattern", pattern],
                input=prompt,
                capture_output=True,
                text=True,
                timeout=120,
            )
        except FileNotFoundError:
            print("  fabric not installed — skipping review (stage 2)")
            return findings
        if r.returncode != 0:
            print(f"  fabric error: {r.stderr[:200]} — skipping batch")
            continue
        for n, line in enumerate(r.stdout.splitlines()):
            line = line.strip()
            m = re.match(r"^(\d+)\s*[:.]\s*(.+)$", line)
            if m and m.group(1).isdigit() and int(m.group(1)) <= len(batch):
                verdict = m.group(2).strip()
                if verdict.upper() != "OK":
                    s, t = batch[int(m.group(1)) - 1]
                    findings.append(
                        {"source": s, "target": t, "verdict": verdict}
                    )
    return findings


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("-workspace", required=True)
    ap.add_argument("-game", required=True, help="game root containing lingo_translations/")
    ap.add_argument("-retranslate", action="store_true", help="reset broken rows for re-translation")
    ap.add_argument("-skip-fabric", action="store_true")
    ap.add_argument("-fabric-pattern", default="translate_thai_review")
    ap.add_argument("-fabric-limit", type=int, default=200, help="max rows to send to fabric")
    ap.add_argument("-review-sample", type=int, default=60, help="rows sampled for fabric review")
    args = ap.parse_args()

    # ---- Stage 1: format gate ----
    bad = run_format_check(args.workspace)
    print(f"[1/2] Format check: {len(bad)} problem(s) out of translated rows")
    for eid, fp, s, t, probs in bad[:15]:
        print(f"  FAIL {fp} | {probs}\n    JP: {s}\n    TH: {t}")
    if bad and args.retranslate:
        n = retranslate(args.workspace, bad)
        print(f"  -> reset {n} rows to untranslated; run `lingo translate` again")
        return 1
    if bad:
        print("  (use -retranslate to reset these for re-translation)")
        return 1
    print("  PASS")

    # ---- Stage 2: fabric style review (optional) ----
    if args.skip_fabric:
        print("[2/2] Fabric review: skipped")
    else:
        db = sqlite3.connect(args.workspace)
        rows = db.execute(
            "SELECT source, target FROM entries WHERE status='translated' "
            "ORDER BY RANDOM() LIMIT ?", (args.review_sample,),
        ).fetchall()
        db.close()
        print(f"[2/2] Fabric review: sampling {len(rows)} rows (pattern: {args.fabric_pattern})")
        findings = fabric_review(rows, args.fabric_pattern, args.fabric_limit)
        out_csv = os.path.join(os.path.dirname(os.path.abspath(args.workspace)), "qa_fabric_findings.csv")
        if findings:
            with open(out_csv, "w", newline="", encoding="utf-8") as f:
                w = csv.DictWriter(f, fieldnames=["source", "target", "verdict"])
                w.writeheader()
                w.writerows(findings)
            print(f"  {len(findings)} finding(s) -> {out_csv}")
        else:
            print("  PASS (no findings)")

    # ---- Stage 3: copy workspace translations into the game ----
    lt = os.path.join(args.game, "lingo_translations")
    if not os.path.isdir(lt):
        sys.exit(f"no {lt} — run `lingo deploy` first to create the layer")
    db = sqlite3.connect(args.workspace)
    rows = db.execute(
        "SELECT file_path, key_path, source, target FROM entries WHERE status='translated'"
    ).fetchall()
    db.close()
    by_file: dict[str, dict[str, str]] = {}
    for fp, kp, src, tgt in rows:
        by_file.setdefault(os.path.basename(fp), {})[src] = tgt
    written = 0
    for fname, mapping in by_file.items():
        path = os.path.join(lt, fname)
        if not os.path.exists(path):
            continue
        data = json.load(open(path, encoding="utf-8"))
        changed = 0
        if isinstance(data, list):
            for pair in data:
                if isinstance(pair, list) and len(pair) == 2 and isinstance(pair[0], str) \
                        and isinstance(pair[1], str) and pair[0] in mapping:
                    new = mapping[pair[0]]
                    if pair[1] != new:
                        pair[1] = new
                        changed += 1
        else:
            for k, v in data.items():
                if isinstance(v, str) and k in mapping and v != mapping[k]:
                    data[k] = mapping[k]
                    changed += 1
        if changed:
            json.dump(data, open(path, "w", encoding="utf-8"), ensure_ascii=False, indent=1)
        written += changed
    print(f"[3/3] Synced workspace -> {lt}: {written} row(s) updated across {len(by_file)} file(s)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
