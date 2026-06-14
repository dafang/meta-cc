---
name: prompt-list
description: List saved prompts in the meta-cc prompt library, optionally filtering by category or sorting by usage, date, or title.
---

# Prompt Library Listing

List saved prompts under `.meta-cc/prompts/library/` in the current project.

## Arguments

Support these optional key-value arguments when the user provides them:

- `category=<name>` filters by category.
- `sort=usage` sorts by `usage_count` descending.
- `sort=date` sorts by `updated` descending.
- `sort=alpha` sorts by title or ID ascending.

Default sort is `usage`.

## Workflow

1. Resolve the library path as `.meta-cc/prompts/library/` from the current project root.
2. If the directory does not exist, say no prompt library exists.
3. Read every `*.md` file in the library.
4. Parse YAML frontmatter when present. Prefer these fields when available:
   - `id`
   - `title`
   - `category`
   - `keywords`
   - `usage_count`
   - `updated`
   - `status`
5. Apply the category filter if provided.
6. Sort according to the requested sort mode.
7. Show a summary with total prompt count, category count, and total usage count when available.
8. Present a compact table with title, category, uses, updated date, and ID.

## Output

Keep output scannable. Tell the user to use `$prompt-show <id>` for full prompt
details.
