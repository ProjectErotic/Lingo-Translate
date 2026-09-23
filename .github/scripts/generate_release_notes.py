#!/usr/bin/env python3
"""
Generate release notes from git commits between versions,
with optional AI summarization via Gemini or OpenAI API.
"""

import argparse
import json
import os
import re
import subprocess
import sys
import urllib.request
import urllib.error

def run_cmd(cmd):
    try:
        return subprocess.check_output(cmd, stderr=subprocess.PIPE).decode('utf-8', errors='replace').strip()
    except subprocess.CalledProcessError:
        return ""

def get_previous_tag(ref):
    # Try finding ancestor tag of ref
    prev = run_cmd(["git", "describe", "--tags", "--abbrev=0", f"{ref}^"])
    if prev and prev != ref:
        return prev

    # If ref is HEAD or a commit not yet tagged, git describe finds the latest existing tag
    latest = run_cmd(["git", "describe", "--tags", "--abbrev=0", ref])
    if latest and latest != ref:
        return latest

    # Fallback: find the most recent tag before ref
    all_tags = run_cmd(["git", "tag", "--sort=-v:refname"]).splitlines()
    for t in all_tags:
        t = t.strip()
        if t and t != ref:
            return t
    return ""

def get_commits(prev_tag, target_ref):
    if prev_tag:
        commit_range = f"{prev_tag}..{target_ref}"
    else:
        commit_range = target_ref

    output = run_cmd(["git", "log", commit_range, "--pretty=format:%h\t%s\t%an"])
    commits = []
    for line in output.splitlines():
        line = line.strip()
        if not line:
            continue
        parts = line.split("\t")
        if len(parts) >= 2:
            commit_hash = parts[0]
            subject = parts[1]
            author = parts[2] if len(parts) > 2 else ""
            commits.append({
                "hash": commit_hash,
                "subject": subject,
                "author": author
            })
    return commits

def categorize_commits(commits):
    categories = {
        "features": {
            "title": "🚀 Features",
            "patterns": [r"^feat(\([^)]+\))?:", r"^feature(\([^)]+\)):?"],
            "items": []
        },
        "fixes": {
            "title": "🐛 Bug Fixes",
            "patterns": [r"^fix(\([^)]+\))?:", r"^bug(\([^)]+\)):?"],
            "items": []
        },
        "perf": {
            "title": "⚡ Performance & Refactoring",
            "patterns": [r"^perf(\([^)]+\))?:", r"^refactor(\([^)]+\))?:"],
            "items": []
        },
        "maintenance": {
            "title": "🧰 Maintenance & CI/CD",
            "patterns": [r"^chore(\([^)]+\))?:", r"^ci(\([^)]+\))?:", r"^build(\([^)]+\))?:", r"^test(\([^)]+\))?:", r"^docs(\([^)]+\))?:", r"^bump:"],
            "items": []
        },
        "other": {
            "title": "📝 Other Changes",
            "patterns": [],
            "items": []
        }
    }

    for c in commits:
        subj = c["subject"]
        matched = False
        for cat_key, cat_data in categories.items():
            if cat_key == "other":
                continue
            for pattern in cat_data["patterns"]:
                if re.search(pattern, subj, re.IGNORECASE):
                    cat_data["items"].append(c)
                    matched = True
                    break
            if matched:
                break
        if not matched:
            categories["other"]["items"].append(c)

    return categories

def generate_ai_summary(commits, prev_tag, current_tag):
    gemini_key = os.environ.get("GEMINI_API_KEY", "").strip()
    openai_key = os.environ.get("OPENAI_API_KEY", "").strip()

    if not gemini_key and not openai_key:
        return None

    commit_list_text = "\n".join([f"- {c['subject']} (commit {c['hash']})" for c in commits])
    prompt = (
        f"You are a professional software release note generator. Summarize the changes between version "
        f"{prev_tag or 'initial'} and {current_tag} from the following git commits in clear, concise English markdown:\n\n"
        f"{commit_list_text}\n\n"
        f"Guidelines:\n"
        f"1. Provide a brief 1-2 sentence executive overview highlighting major additions and fixes.\n"
        f"2. Group key updates into sections (e.g. Highlights, Improvements, Fixes) if appropriate.\n"
        f"3. Use clean markdown formatting with bullet points and emojis.\n"
        f"4. Do not include conversational filler; output only the markdown release summary content."
    )

    # Try Gemini first
    if gemini_key:
        models = ["gemini-2.0-flash", "gemini-1.5-flash"]
        for model in models:
            url = f"https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent?key={gemini_key}"
            payload = {
                "contents": [{"parts": [{"text": prompt}]}],
                "generationConfig": {
                    "temperature": 0.2,
                    "maxOutputTokens": 1200
                }
            }
            try:
                req = urllib.request.Request(
                    url,
                    data=json.dumps(payload).encode("utf-8"),
                    headers={"Content-Type": "application/json"}
                )
                with urllib.request.urlopen(req, timeout=30) as resp:
                    data = json.loads(resp.read().decode("utf-8"))
                    text = data["candidates"][0]["content"]["parts"][0]["text"]
                    return text.strip()
            except Exception as e:
                print(f"[Warning] Gemini API call ({model}) failed: {e}", file=sys.stderr)

    # Try OpenAI if Gemini not set or failed
    if openai_key:
        url = "https://api.openai.com/v1/chat/completions"
        payload = {
            "model": "gpt-4o-mini",
            "messages": [
                {"role": "system", "content": "You are a professional software release note summarizer."},
                {"role": "user", "content": prompt}
            ],
            "temperature": 0.2,
            "max_tokens": 1000
        }
        try:
            req = urllib.request.Request(
                url,
                data=json.dumps(payload).encode("utf-8"),
                headers={
                    "Content-Type": "application/json",
                    "Authorization": f"Bearer {openai_key}"
                }
            )
            with urllib.request.urlopen(req, timeout=30) as resp:
                data = json.loads(resp.read().decode("utf-8"))
                text = data["choices"][0]["message"]["content"]
                return text.strip()
        except Exception as e:
            print(f"[Warning] OpenAI API call failed: {e}", file=sys.stderr)

    return None

def build_markdown(repo, prev_tag, current_tag, commits, categories, ai_summary):
    md = []
    
    # Title / Header
    if prev_tag:
        diff_title = f"Changes in {current_tag} (compared to {prev_tag})"
    else:
        diff_title = f"Release {current_tag}"
    
    # AI Summary Section
    if ai_summary:
        md.append("### 🤖 Overview & Highlights (AI Summary)")
        md.append(ai_summary)
        md.append("")
        md.append("---")
        md.append("")

    # Categorized Commits
    has_categorized = False
    for cat_key, cat_data in categories.items():
        items = cat_data["items"]
        if not items:
            continue
        has_categorized = True
        md.append(f"### {cat_data['title']}")
        for item in items:
            h = item["hash"]
            subj = item["subject"]
            author = item["author"]
            if repo:
                commit_link = f"[{h}](https://github.com/{repo}/commit/{h})"
            else:
                commit_link = f"`{h}`"
            author_str = f" by @{author}" if author else ""
            md.append(f"- {subj} ({commit_link}){author_str}")
        md.append("")

    if not has_categorized:
        md.append("### 📝 Commits")
        for item in commits:
            h = item["hash"]
            subj = item["subject"]
            if repo:
                commit_link = f"[{h}](https://github.com/{repo}/commit/{h})"
            else:
                commit_link = f"`{h}`"
            md.append(f"- {subj} ({commit_link})")
        md.append("")

    # Compare Link
    if repo and prev_tag and current_tag:
        md.append("---")
        md.append(f"🔍 **Full Changelog**: "
                  f"[https://github.com/{repo}/compare/{prev_tag}...{current_tag}]"
                  f"(https://github.com/{repo}/compare/{prev_tag}...{current_tag})")
        md.append("")

    return "\n".join(md)

def main():
    parser = argparse.ArgumentParser(description="Generate release notes from git commits.")
    parser.add_argument("--current-tag", default="", help="Current tag or commit (defaults to latest tag/HEAD)")
    parser.add_argument("--prev-tag", default="", help="Previous tag (auto-detected if omitted)")
    parser.add_argument("--repo", default=os.environ.get("GITHUB_REPOSITORY", ""), help="GitHub repo owner/name (e.g. ProjectErotic/Lingo-Translate)")
    parser.add_argument("--output", default="release_notes.md", help="Output file path")
    args = parser.parse_args()

    current_tag = args.current_tag
    if not current_tag:
        current_tag = run_cmd(["git", "describe", "--tags", "--exact-match", "HEAD"])
        if not current_tag:
            current_tag = "HEAD"

    # Resolve target git revision: if current_tag is not yet a git tag in the local repo, use HEAD
    target_ref = current_tag
    if not run_cmd(["git", "rev-parse", "--verify", f"{current_tag}^{{commit}}"]):
        target_ref = "HEAD"

    prev_tag = args.prev_tag
    if not prev_tag:
        prev_tag = get_previous_tag(target_ref)

    print(f"[Release Notes] Target: {current_tag} (ref: {target_ref}) | Previous: {prev_tag or 'None'} | Repo: {args.repo}")

    commits = get_commits(prev_tag, target_ref)
    print(f"[Release Notes] Found {len(commits)} commits")

    categories = categorize_commits(commits)
    ai_summary = None
    if commits:
        ai_summary = generate_ai_summary(commits, prev_tag, current_tag)
        if ai_summary:
            print("[Release Notes] AI summary generated successfully!")
        else:
            print("[Release Notes] No AI summary generated (API key not set or request skipped).")

    notes = build_markdown(args.repo, prev_tag, current_tag, commits, categories, ai_summary)

    with open(args.output, "w", encoding="utf-8") as f:
        f.write(notes)

    print(f"[Release Notes] Saved to {args.output}")

if __name__ == "__main__":
    main()
