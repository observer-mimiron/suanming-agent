#!/usr/bin/env python3
"""Fix index.md titles by reading frontmatter title from each referenced .md file."""

import re
import os
import yaml

KNOWLEDGE_DIR = os.path.join(os.path.dirname(__file__), "..", "knowledge", "wiki")
INDEX_FILE = os.path.join(KNOWLEDGE_DIR, "index.md")

def extract_frontmatter_title(filepath):
    """Extract 'title' from YAML frontmatter of a markdown file."""
    try:
        with open(filepath, "r", encoding="utf-8") as f:
            content = f.read()
        # Match YAML frontmatter between --- delimiters
        m = re.match(r"^---\s*\n(.*?)\n---", content, re.DOTALL)
        if not m:
            return None
        fm = yaml.safe_load(m.group(1))
        return fm.get("title") if fm else None
    except Exception:
        return None


def fix_index():
    with open(INDEX_FILE, "r", encoding="utf-8") as f:
        lines = f.readlines()

    pattern = re.compile(r"^(\s*-\s*)\[(.*?)\]\(([^)]+\.md)\)(\s*—\s*.*)$")
    fixed_count = 0

    for i, line in enumerate(lines):
        m = pattern.match(line.rstrip("\n"))
        if not m:
            continue

        prefix, current_title, slug, suffix = m.groups()
        filepath = os.path.join(KNOWLEDGE_DIR, slug)

        if not os.path.exists(filepath):
            print(f"  SKIP (file not found): {slug}")
            continue

        fm_title = extract_frontmatter_title(filepath)
        if fm_title is None:
            print(f"  SKIP (no frontmatter title): {slug}")
            continue

        if fm_title == current_title:
            continue  # Already correct

        # Replace the title in the link text
        old_line = line.rstrip("\n")
        new_line = f"{prefix}[{fm_title}]({slug}){suffix}"
        lines[i] = new_line + "\n"
        fixed_count += 1
        print(f"  FIX: [{current_title}] → [{fm_title}]  ({slug})")

    # Write back
    with open(INDEX_FILE, "w", encoding="utf-8") as f:
        f.writelines(lines)

    print(f"\nDone. Fixed {fixed_count} entries in {INDEX_FILE}")


if __name__ == "__main__":
    # Check for pyyaml
    try:
        import yaml
    except ImportError:
        print("ERROR: PyYAML is required. Install with: pip install pyyaml")
        exit(1)

    fix_index()
