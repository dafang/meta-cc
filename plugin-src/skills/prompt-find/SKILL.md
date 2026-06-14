---
name: prompt-find
description: Search the meta-cc prompt library by keywords when the user wants to find saved prompts, reusable prompt templates, or prompt-library entries.
---

# Prompt Library Search

Search saved prompts under `.meta-cc/prompts/library/` in the current project.

## Input

Use the user's prompt as the keyword query. If the user explicitly provides
terms after `$prompt-find`, search those terms. If no keywords are provided,
show concise usage help.

## Workflow

1. Resolve the library path as `.meta-cc/prompts/library/` from the current project root.
2. If the directory does not exist, say no prompt library exists and suggest saving prompts before searching.
3. Read every `*.md` file in the library.
4. Parse YAML frontmatter when present. Prefer these fields when available:
   - `id`
   - `title`
   - `category`
   - `keywords`
   - `usage_count`
   - `updated`
5. Score matches using keyword overlap:
   - frontmatter `keywords` and `title`: strongest signal
   - body sections such as original prompts or optimized prompt: medium signal
   - `category`: weak signal
6. Sort matches descending by score.
7. Present a compact table with ID, category, score, title, and updated date when available.
8. If no entries match, say no matching prompts were found and suggest trying broader keywords.

## Output

Keep output concise. Do not print full prompt bodies in this skill; tell the
user to use `$prompt-show <id>` for details.
