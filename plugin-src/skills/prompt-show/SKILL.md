---
name: prompt-show
description: Show the full details of a saved meta-cc prompt by ID or partial ID from .meta-cc/prompts/library/.
---

# Prompt Details Viewer

Display a saved prompt from `.meta-cc/prompts/library/` in the current project.

## Input

Use the provided ID or partial ID. If no ID is provided, show concise usage
help and suggest `$prompt-list`.

## Workflow

1. Resolve the library path as `.meta-cc/prompts/library/` from the current project root.
2. If the directory does not exist, say no prompt library exists.
3. Find matching prompt files:
   - exact filename match: `<id>.md`
   - prefix match: `<id>*.md`
   - contains match: `*<id>*.md`
4. If multiple files match, list the candidates and ask the user to provide a more specific ID.
5. Read the selected Markdown file.
6. Parse YAML frontmatter when present. Prefer these fields when available:
   - `id`
   - `title`
   - `category`
   - `keywords`
   - `created`
   - `updated`
   - `usage_count`
   - `effectiveness`
   - `variables`
   - `status`
7. Display the metadata first, then the prompt body sections.
8. Preserve code fences and Markdown headings from the saved prompt.

## Output

Show enough context for the user to reuse the prompt directly. Avoid rewriting
the saved prompt unless the user explicitly asks for edits.
