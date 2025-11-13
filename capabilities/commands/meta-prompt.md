---
name: meta-prompt
description: Refine prompts using successful patterns from project history.
argument-hint: [prompt]
keywords: prompt, refinement, optimization, effectiveness, clarity
category: guidance
---

λ(prompt_raw) → prompt_refined | workflow:
  search_history(prompt_raw) →ᵉˣⁱᵗ reused_prompt
  ∨ (search_similar(prompt_raw) → analyze_quality(candidates) → extract_patterns(quality_data) → detect_gaps(prompt_raw, patterns) → generate_alternatives(prompt_raw, gaps, patterns) → output(alternatives))
  →ᵒᵖᵗ save_workflow(result) → saved_prompt

where:
  prompt_raw :: `$1`
  library_path :: ".meta-cc/prompts/library/"
  workflow :: search_library → search_history → analyze_quality → extract_patterns → detect_gaps → generate_alternatives → save
  early_exit :: reuse_from_library → skip(all_optimization_phases)
  normal_flow :: multi_stage_optimization → optional_save
  mcp_tools_required :: [query_user_messages, query_tool_errors, query_conversation_flow, query_token_usage]

---

## Phase 1: Search Library

search_history :: Prompt → Search_Result
search_history(P) = {
  keywords: extract_keywords(P),
  candidates: ∀file ∈ get_library_path(project_root()): {
    meta: parse_frontmatter(file),
    similarity: jaccard_similarity(keywords, meta.keywords ∪ extract_keywords(extract_original(file))),
    usage_score: log(meta.usage_count + 1) / 5.0,
    combined_score: (similarity * 0.7) + (usage_score * 0.3)
  },
  matches: filter(candidates, c → c.similarity > 0.2) |> sort_desc(combined_score) |> take(5),
  if (|matches| > 0): display_matches(matches) → user_selection → {"reuse": update_usage(selected) → {action: "exit_early", prompt: selected.optimized}, "skip": {action: "continue", prompt: null}},
  else: {action: "continue", prompt: null}
}

extract_keywords :: String → [String]
extract_keywords(S) = tokenize(S) |> filter(w → |w| > 2 ∧ w ∉ stopwords) |> lowercase |> unique

jaccard_similarity :: ([String], [String]) → Float
jaccard_similarity(A, B) = |A ∩ B| / |A ∪ B|

parse_frontmatter :: FilePath → Metadata
parse_frontmatter(F) = extract_yaml(F) |> validate(required_fields)

update_usage_count :: FilePath → Result
update_usage_count(F) = atomic_write(F, {usage_count: +1, updated: now()})

---

## Phase 2: Optimize Prompt

refine :: Raw_Prompt → Optimized_Prompts
refine(P) = search_similar(P) → analyze_quality(candidates) → extract_patterns(quality_analysis) → detect_gaps(P, patterns) → generate_alternatives(P, gaps, patterns)

---

### Step 2.1: Search Similar Historical Prompts

**CRITICAL**: This step MUST execute MCP tool `query_user_messages` to search project history.

search_similar :: Prompt → [Historical_Prompts]
search_similar(P) = {
  keywords: extract_keywords(P),
  regex_pattern: keywords |> join("|") |> escape_regex,

  // ⚠️ REQUIRED MCP TOOL CALL ⚠️
  // Use query_user_messages to search historical prompts
  mcp_call: query_user_messages({
    pattern: regex_pattern,
    scope: "project",
    limit: 10
  }),

  // Expected return format: [{turn, timestamp, content}]
  results: mcp_call |> sort_by(.timestamp) |> reverse,

  // Display candidates to user
  display: if (|results| > 0):
    "Found " + |results| + " similar historical prompts:\n" + format_table(results),
  else:
    "No similar historical prompts found. Proceeding with best practices analysis.",

  return: results
}

---

### Step 2.2: Analyze Quality Metrics

**CRITICAL**: This step MUST execute multiple MCP tools to assess prompt quality.

analyze_quality :: [Historical_Prompts] → [Quality_Analysis]
analyze_quality(candidates) = if (|candidates| == 0): [], else: candidates |> map(c → {
  prompt: c,
  turn: c.turn,
  timestamp: c.timestamp,

  // ⚠️ REQUIRED MCP TOOL CALL 1: Error Rate Analysis ⚠️
  // Query tool errors within 1 hour after this prompt
  errors_mcp: query_tool_errors({
    scope: "project",
    limit: 100
  }),
  errors: errors_mcp
    |> filter(e → e.timestamp > c.timestamp
                 ∧ timestamp_diff(e.timestamp, c.timestamp) < 3600),
  error_count: |errors|,

  // ⚠️ REQUIRED MCP TOOL CALL 2: Conversation Efficiency ⚠️
  // Query conversation flow to count turns to completion
  conversation_mcp: query_conversation_flow({
    scope: "project",
    limit: 100
  }),
  conversation_segment: conversation_mcp
    |> filter(msg → msg.turn >= c.turn)
    |> takeWhile(msg → !(msg.type == "user" ∧ msg.turn > c.turn)),
  turns_to_complete: |conversation_segment|,

  // ⚠️ REQUIRED MCP TOOL CALL 3: Token Efficiency ⚠️
  // Query token usage for this time window
  tokens_mcp: query_token_usage({
    scope: "project",
    limit: 100
  }),
  tokens: tokens_mcp
    |> filter(t → t.timestamp >= c.timestamp
                 ∧ timestamp_diff(t.timestamp, c.timestamp) < 3600)
    |> sum(.input_tokens + .output_tokens),

  // Calculate quality score
  quality_score: calculate_quality_score(error_count, turns_to_complete, tokens, |c.content|),

  return: {
    prompt: c.content,
    turn: c.turn,
    timestamp: c.timestamp,
    metrics: {
      error_count: error_count,
      turns_to_complete: turns_to_complete,
      total_tokens: tokens,
      prompt_length: |c.content|
    },
    quality_score: quality_score
  }
})

calculate_quality_score :: (Errors, Turns, Tokens, Length) → Float
calculate_quality_score(E, T, K, L) = {
  base: 1.0,
  error_penalty: E > 0 ? 0.5 : 1.0,
  efficiency_factor: T > 10 ? 0.7 : (T > 5 ? 0.85 : 1.0),
  token_factor: K > 10000 ? 0.8 : (K > 5000 ? 0.9 : 1.0),
  length_factor: L > 500 ? 0.9 : (L < 50 ? 0.85 : 1.0),

  final: base * error_penalty * efficiency_factor * token_factor * length_factor,
  return: min(final, 1.0)
}

---

### Step 2.3: Extract Success Patterns

**CRITICAL**: Analyze quality metrics to extract common patterns from high-quality prompts.

extract_patterns :: [Quality_Analysis] → Success_Patterns
extract_patterns(QA) = {
  // Filter high-quality prompts (score >= 0.6)
  high_quality: QA |> filter(q → q.quality_score >= 0.6),
  total_count: |QA|,

  // Extract structural features from high-quality prompts
  features: high_quality |> analyze_features,

  patterns: {
    // Pattern 1: Clear Goals
    has_clear_goal: {
      count: high_quality |> count(p → matches(p.prompt, /goal:|objective:|implement|create|fix|refactor/i)),
      percentage: (count / max(|high_quality|, 1)) * 100,
      examples: high_quality |> filter(p → matches(p.prompt, /goal:|objective:/i)) |> map(.prompt) |> take(2)
    },

    // Pattern 2: Explicit Constraints
    has_constraints: {
      count: high_quality |> count(p → matches(p.prompt, /must|should|constraint|requirement|limit/i)),
      percentage: (count / max(|high_quality|, 1)) * 100,
      examples: high_quality |> filter(p → matches(p.prompt, /must|should/i)) |> map(.prompt) |> take(2)
    },

    // Pattern 3: File References
    has_file_refs: {
      count: high_quality |> count(p → matches(p.prompt, /@file:|\.go|\.md|\.py|path:|file:/i)),
      percentage: (count / max(|high_quality|, 1)) * 100,
      examples: high_quality |> filter(p → matches(p.prompt, /@file:/i)) |> map(.prompt) |> take(2)
    },

    // Pattern 4: Agent References
    has_agent_refs: {
      count: high_quality |> count(p → matches(p.prompt, /@agent-|Task tool|subagent|Explore/i)),
      percentage: (count / max(|high_quality|, 1)) * 100,
      examples: high_quality |> filter(p → matches(p.prompt, /@agent-/i)) |> map(.prompt) |> take(2)
    },

    // Pattern 5: Specific Locations
    has_locations: {
      count: high_quality |> count(p → matches(p.prompt, /:\d+|line \d+|lines? \d+-\d+/i)),
      percentage: (count / max(|high_quality|, 1)) * 100
    },

    // Pattern 6: Acceptance Criteria
    has_acceptance: {
      count: high_quality |> count(p → matches(p.prompt, /acceptance|criteria|should pass|verify|test/i)),
      percentage: (count / max(|high_quality|, 1)) * 100
    },

    // Statistical measures
    avg_length: high_quality |> avg(p → |p.prompt|),
    avg_turns: high_quality |> avg(p → p.metrics.turns_to_complete),
    avg_tokens: high_quality |> avg(p → p.metrics.total_tokens),

    // Top keywords (frequency analysis)
    common_keywords: high_quality
      |> flatMap(p → extract_keywords(p.prompt))
      |> frequency
      |> sort_desc(by_value)
      |> take(15)
  },

  // Display insights
  display: format_pattern_insights(patterns, total_count, |high_quality|),

  return: patterns
}

---

### Step 2.4: Detect Gaps & Generate Alternatives

detect_gaps :: (Prompt, Patterns) → [Improvement_Areas]
detect_gaps(P, S) = {
  current_features: {
    has_goal: matches(P, /goal:|objective:|implement|create|fix|refactor/i),
    has_constraints: matches(P, /must|should|constraint|requirement|limit/i),
    has_file_refs: matches(P, /@file:|\.go|\.md|\.py|path:|file:/i),
    has_agent_refs: matches(P, /@agent-|Task tool|subagent|Explore/i),
    has_locations: matches(P, /:\d+|line \d+|lines? \d+-\d+/i),
    has_acceptance: matches(P, /acceptance|criteria|should pass|verify|test/i),
    length: |P|
  },

  gaps: {
    missing_goal: ¬current_features.has_goal ∧ S.has_clear_goal.percentage > 50,
    missing_constraints: ¬current_features.has_constraints ∧ S.has_constraints.percentage > 50,
    missing_file_refs: ¬current_features.has_file_refs ∧ S.has_file_refs.percentage > 40,
    missing_agent_refs: ¬current_features.has_agent_refs ∧ S.has_agent_refs.percentage > 30,
    missing_locations: ¬current_features.has_locations ∧ S.has_locations.percentage > 30,
    missing_acceptance: ¬current_features.has_acceptance ∧ S.has_acceptance.percentage > 40,

    // Length analysis
    too_long: current_features.length > S.avg_length * 1.5,
    too_short: current_features.length < S.avg_length * 0.5,

    // Keyword gap
    current_keywords: extract_keywords(P),
    keyword_gap: S.common_keywords |> filter(kw → kw ∉ current_keywords) |> take(5)
  },

  significant_gaps: gaps |> filter(g → g.value == true) |> keys,

  return: {gaps: gaps, significant: significant_gaps}
}

generate_alternatives :: (Prompt, Gap_Analysis, Patterns) → [Optimized_Prompts]
generate_alternatives(P, G, S) = {
  alternatives: [],

  // Alternative 1: Add missing structural elements
  alt1: if (G.gaps.missing_goal ∨ G.gaps.missing_constraints ∨ G.gaps.missing_acceptance):
    optimize_structure(P, S, {
      add_goal: G.gaps.missing_goal,
      add_constraints: G.gaps.missing_constraints,
      add_acceptance: G.gaps.missing_acceptance
    }),

  // Alternative 2: Add file/agent references
  alt2: if (G.gaps.missing_file_refs ∨ G.gaps.missing_agent_refs):
    optimize_references(P, S, {
      add_file_refs: G.gaps.missing_file_refs,
      add_agent_refs: G.gaps.missing_agent_refs,
      add_locations: G.gaps.missing_locations
    }),

  // Alternative 3: Adjust length and incorporate common keywords
  alt3: if (G.gaps.too_long ∨ G.gaps.too_short ∨ |G.gaps.keyword_gap| > 2):
    optimize_content(P, S, {
      target_length: S.avg_length,
      add_keywords: G.gaps.keyword_gap,
      current_length: |P|
    }),

  // Collect non-null alternatives
  candidates: [alt1, alt2, alt3] |> filter(not_null),

  // Rank by expected quality improvement
  ranked: candidates |> rank_by(expected_quality_improvement),

  // Take top 3
  final: ranked |> take(3),

  return: final
}

output :: (Prompt, Gap_Analysis, Patterns, Alternatives) → Report
output(P, G, S, A) = {
  display: format_report({
    original: P,

    analysis: {
      patterns_found: S.has_clear_goal.count + S.has_constraints.count + S.has_file_refs.count + S.has_agent_refs.count + " patterns from " + |S.high_quality| + " high-quality prompts",
      gaps_detected: G.significant,
      improvement_potential: estimate_improvement(G, S)
    },

    alternatives: A |> enumerate |> map((i, alt) → {
      number: i + 1,
      prompt: alt.optimized,
      improvements: alt.changes,
      expected_quality: alt.estimated_score
    }),

    recommendation: {
      best_option: argmax(A, a → a.estimated_score),
      rationale: explain_recommendation(best_option, G, S)
    }
  }),

  note: "⚠️ These are suggestions only. Review and modify before use.",

  return: {original: P, alternatives: A, patterns: S, gaps: G}
}

---

## Phase 3: Save to Library

save_workflow :: Optimized_Result → Optional[Saved_File]
save_workflow(R) = display: output(R), ask: "Save optimized prompt to library? (y/N): " → read_input() → {confirmed: call_save(R), skipped: {saved: false}}

call_save :: Result → Saved_File
call_save(R) = {storage: initialize(project_root()), metadata: collect_metadata(R), id: generate_id(storage, metadata), title: infer_title(R.optimized, metadata.description), variables: extract_variables(R.optimized), frontmatter: create_frontmatter(id, title, metadata, variables, now()), content: format_content(R.original, R.optimized), filepath: atomic_write(storage + "/" + id + ".md", frontmatter + "\n---\n\n" + content), display: "✓ Saved to: " + filepath + "\n   Browse: /meta prompts/meta-prompt-list", return: {saved: true, filepath: filepath}}

initialize :: Project_Root → Storage_Path
initialize(P) = {path: P + "/.meta-cc/prompts/library/", exists(path) ? path : mkdir(path) ∧ write_gitignore(path) → path}

generate_id :: (Storage_Path, Category, Description) → Unique_ID
generate_id(S, C, D) = {pattern: C + "-" + D + "-*.md", max_num: glob(S + "/" + pattern) |> extract_numbers |> max |> (+1), return: sprintf("%s-%s-%03d", C, D, max_num)}

collect_metadata :: Result → User_Input
collect_metadata(R) = {category: ask("Category (release/debug/refactor/test/docs/feature/other): ") |> validate, keywords: ask("Keywords (comma-separated): ") |> split(",") |> validate(≥2), description: ask("Short description (kebab-case): ") |> normalize |> validate(/^[a-z][a-z0-9-]*$/)}

extract_variables :: Prompt → [Variable_Names]
extract_variables(P) = find_all(P, /\{([A-Z_]+)\}/g) |> unique

create_frontmatter :: (ID, Title, Category, Keywords, Vars, Timestamp) → YAML
create_frontmatter(id, title, cat, kw, vars, ts) = format_yaml({
  id, title, category: cat, keywords: kw, variables: vars,
  created: ts, updated: ts, usage_count: 0, effectiveness: 1.0, status: "active"
})

format_content :: (Original, Optimized) → Markdown
format_content(O, P) = "## Original Prompts\n" + format_list(O) + "\n\n## Optimized Prompt\n\n" + P

---

## Library Management

list_prompts :: (Category?, Sort?, Detail?) → Display
list_prompts(C, S, D) = {library: get_library_path(project_root()), if (D ≠ null): show_detail(library, D), else: prompts: ∀file ∈ glob(library + "*.md"): parse_frontmatter(file) |> filter(p → C == null ∨ p.category == C) |> apply_sort(S ∨ "usage"), if (empty(prompts)): display_empty_message(), else: stats: calculate_stats(prompts), display: format_summary(stats) + "\n\n" + format_table(prompts)}

apply_sort :: ([Prompts], Sort_Method) → [Prompts]
apply_sort(P, M) = case M of {"usage": sort_desc(P, p → p.usage_count), "date": sort_desc(P, p → p.updated), "alpha": sort_asc(P, p → lowercase(p.title))}

calculate_stats :: [Prompts] → Statistics
calculate_stats(P) = {total: |P|, categories: |unique(map(P, p → p.category))|, total_usage: sum(map(P, p → p.usage_count)), most_used: argmax(P, p → p.usage_count)}

format_table :: [Prompts] → String
format_table(P) = header + separator + join(rows(P), "\n") where rows(p) = sprintf("%-40s %-15s %-10d %-10s", truncate(p.title, 40), p.category, p.usage_count, format_date(p.updated))

show_detail :: (Storage_Path, Prompt_ID) → Display
show_detail(S, ID) = read_file(glob(S + ID + ".md")[0]) |> display_with_header

---

## Constants & Configuration

config :: System_Config
config = {
  library_path: ".meta-cc/prompts/library/",
  similarity_threshold: 0.2,
  scoring_weights: {similarity: 0.7, usage: 0.3},
  usage_normalization: 5.0,
  max_matches: 5,
  max_alternatives: 3,

  // Phase 2: Quality Analysis Configuration
  quality_threshold: 0.6,                  // Minimum score for "high quality" prompts
  historical_search_limit: 10,             // Max historical prompts to analyze
  analysis_time_window: 3600,              // 1 hour in seconds
  pattern_percentage_threshold: {
    goal: 50,                               // 50%+ high-quality prompts have goals
    constraints: 50,
    file_refs: 40,
    agent_refs: 30,
    locations: 30,
    acceptance: 40
  },

  stopwords: ["the", "a", "an", "and", "or", "to", "in", "of", "for", "on", "with", "is", "are", "was", "were", "be", "been", "have", "has", "had", "do", "does", "did", "will", "would", "should", "could", "can", "may", "might", "this", "that", "these", "those", "i", "you", "he", "she", "it", "we", "they"],
  allowed_categories: ["release", "debug", "refactor", "test", "docs", "feature", "hotfix", "optimization", "security", "other"],
  required_fields: ["id", "title", "category", "keywords", "created", "updated", "usage_count"]
}

metadata_schema :: {id, title, category, keywords: [String], variables: [String], created, updated: ISO8601, usage_count: Int, effectiveness: Float, status: "active"|"archived"}
