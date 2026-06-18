# Decisions

Append-only log of non-obvious technical and design choices, newest first. One
entry per decision; never rewrite history — supersede an old entry with a new one.

Format:

```
## YYYY-MM-DD — Short title
**Choice:** What we picked.
**Why:** The reasoning, including alternatives considered.
**Implications:** What this commits us to or rules out.
```

What belongs here: anything a future reader would otherwise have to re-derive or
re-litigate — a language choice, a persistence model, a concurrency policy, a
deliberate non-feature, a rejected refactor. *Routine* decisions don't.

---

## 2026-06-18 — `@layout` split M3: implementation
**Choice:** Implement the 2026-06-17 "separate semantics from layout" ADR's first phase. Concretely:
- **Dropped the inline layout shorthand.** A node body is now purely semantic (`label`, `kind`, `anchors: [names]`, child nodes). `direction`/`size`/`margin`/`box`/anchor positions live *only* in `@layout` — bare `direction:` on a node is a syntax error pointing the author to `@layout`. Chosen over keeping shorthand so a reader can always tell semantics from presentation; an inline `@layout {…}` block still colocates a node's own layout.
- **Removed the `group` keyword entirely.** A borderless grouping is a `box: none` node, which (unlike the old group) keeps an id and therefore contributes a path segment — former group children move from `c1.x` to `c1.grp.x`. Geometry is unchanged, so SVG stays byte-identical; only reference paths shift.
- **One uniform layout list.** Inline `@layout` is desugared by the parser into a `LayoutRule` whose selector is the enclosing node's full path, so inline and standalone sheet rules are the same "direct" tier and share one conflict/merge path. The AST splits into a semantic `ContainerNode` (no layout fields, `Children []*ContainerNode`, no `Node`/`GroupNode`) and `Declarations`/`LayoutRule` (pointer fields so "unset" is distinguishable for the conflict check).
- **New pure `resolve` stage** merges rules onto paths into `map[path]*Declarations`; the pipeline is parse → validate → resolve → generate. The validator (not resolve) owns all diagnostics: dangling selector (at the selector position), duplicate **direct** property, relocated `size`/`margin`/coord ranges, anchor positions naming a declared name, and arrow anchor-name resolution. Resolve assumes validity and just overlays.
- **`@` tokenizes as its own token**; `@layout` is `@` + identifier, leaving room for `@block`/`@use`/`@group`. `;` became a cosmetic separator (skipped like whitespace). An unpositioned but declared anchor resolves to centre.
**Why:** The split is the whole point of the epic — semantic structure is stable, layout is fiddly — so M3 commits to it cleanly rather than half-keeping the old inline form. Desugaring inline-to-selector avoids two code paths for "direct" layout. Putting every diagnostic in the validator preserves the existing "validator is the single error stage; generator/resolve are pure" shape. Range checks moved off the parser because values are now layout declarations, not node fields.
**Implications:** A breaking syntax change. All golden fixtures, examples, and site samples were rewritten in the new syntax; every SVG renders byte-for-byte identically (verified by `TestGolden` and by regenerating the site SVGs to a no-op diff), confirming the split is structural, not visual. Range errors now report at line 1, column 1 (validator) instead of the old parse position — consistent with the other validator diagnostics, pending the tracked "AST source positions" work. Still open: per-side margins; the precedence tiers stay a single direct tier until M4 adds `kind`/`@use`.

## 2026-06-18 — Auto-cardinal routing M2: implementation
**Choice:** Implement the 2026-06-17 auto-cardinal ADR with: **4 sides** (N/E/S/W) only; each bare endpoint aims at the **other node's box centre** (never the other end's anchor), so routing is stable whatever the far end does; and **no parser or AST change** — the distinction between a bare `a` (auto-route) and an explicit `a#center` (force centre) is already carried by the arrow string, so it is resolved entirely in the generator. Layout exposes a new full-path→border-box map; an explicit anchor (named or `#center`) still resolves to its fixed position and wins.
**Why:** 4 sides keep the side choice unambiguous (a single dominant-axis test) and match the design; 8 would add corner cases for little gain. Aiming at the box centre (not the far anchor) means a named anchor — which can serve many arrows — never destabilises the auto-routed end. The roadmap had pencilled in a parser change to "distinguish no-anchor from `#center`", but inspection showed the raw `ast.Arrow` strings already differ (`a` vs `a#center`), so the only place that ambiguity was being lost was the generator's old `parseTarget`, which defaulted no-anchor to centre.
**Implications:** Cardinal routing depends on M1's margins for a gap to span (flush boxes would give a zero-length edge-to-edge arrow). Anchorless arrows in every diagram move from centre-to-centre to edge-to-edge (goldens and the `pipeline` site sample regenerated; explicitly-anchored arrows such as the `contexts` sample are unaffected). Still open: whether to ever offer 8 directions.

## 2026-06-18 — Box model M1: defaults and margin arithmetic
**Choice:** Implement the margin-based box model (the 2026-06-17 ADR) as inline `margin` / `box` node properties, resolving the open details as:
- **Default margin of 8** user units, non-zero and uniform on all four sides; `margin: 0` restores the old flush packing. Authored margins are absolute layout units (like coordinates), not a fraction of anything.
- **Adjacent sibling margins sum** — no CSS-style collapse-to-max. The gap between two siblings is the sum of their facing margins, which falls straight out of packing each node's margin box flush.
- **Uniform margin only** in v1 (no per-side `margin-top`/…).
- **`box` takes a bareword** (`box: none` / `box: default`), matching the design's surface syntax instead of the quoted-string form `direction` uses today.
- A **`box: none` container keeps its own margin** (it is a real node that merely draws no border); only the layout-only `group` contributes zero margin. Both, as invisible *parents*, collapse their children's perimeter margins.
**Why:** A visible default is what the box model is *for* — it gives auto-cardinal routing (M2) a gap to attach to and makes nesting legible; 8 is a clean, font-independent spacing unit that reads well at the common 12–16px sizes. Summing is the simpler, more predictable rule and the natural consequence of "each node owns its margin box"; collapse-to-max imports CSS's most surprising behaviour for no benefit here. Uniform-only keeps the grammar minimal and is forward-compatible with per-side later. Keeping a margin on `box: none` follows from "it's a node, not a wrapper" — its ID, label, and anchors already separate it from a `group`.
**Implications:** Every multi-node diagram changes; all goldens and the three site sample SVGs were regenerated. The "exact-fit canvas, no padding" rule now holds only at the invisible document root (top-level perimeter margins still collapse) — bordered parents inset their children. Still open and deferred: per-side margins, and whether `box: none` should instead be fully margin-transparent like a `group`.

## 2026-06-17 — Margin-based box model
**Choice:** Express inter-element spacing as a per-node **`margin`** (a layout property), implemented with a real box model that distinguishes a node's **border box** (the visible rectangle) from its **margin box** (border box + margins). Chosen over a parent-level `gap`. A child's margin counts toward a **bordered** parent's size (like padding inside the border) but **collapses** against an **invisible** (`box: none`) parent — including the document root — so it never inflates a box with no wall to sit against. Anchors and arrows attach to the border box.
**Why:** Margin is the more natural authoring model ("this node wants room around it") and composes per-node, unlike a parent-level gap. It is also the enabler for auto-cardinal arrow routing: without a gap, edge-attached arrows between touching boxes degenerate to zero length (Arkitecture packs siblings flush). The invisible-vs-bordered rule is what keeps an invisible grouping from gaining phantom padding while a real container still insets its children.
**Implications:** A substantial layout refactor — sizing, positioning, and canvas all become box-model-aware, tracking border vs margin boxes and margin collapse. It revises the "no padding / exact-fit canvas" stance in `design.md` once a non-zero margin is used. `margin`/`box` are layout properties, so they sit inline today and move into `@layout` when the layer split lands. Open: adjacent-sibling collapse (sum vs max), per-side margins, and the default margin value (0 preserves today's compact look).

## 2026-06-17 — Auto-cardinal arrow endpoints
**Choice:** An arrow that does not name an anchor (`a --> b`) auto-routes: each end attaches to the cardinal side (N/E/S/W) of its box facing the other box's centre, chosen by the dominant axis of the centre-to-centre vector (`|dx| ≥ |dy|` → E/W, else N/S; exact diagonals favour horizontal). Named anchors stay single fixed points (centre if unpositioned). An explicit anchor always overrides.
**Why:** Centre-to-centre lines cut through the boxes and read poorly; "nearest side facing the target" is the cheap heuristic that makes plain arrows look intentional. A named anchor can serve many arrows, so it has no single "other node" to aim at — hence the heuristic belongs to anchor-less endpoints, not to anchor positions.
**Why it doesn't violate "manual, deterministic layout":** it is auto-*routing*, not auto-*placement* — it never moves a box the author positioned, only picks where a line attaches, as a deterministic function of author-controlled positions, and is fully overridable. This is the one bounded "automatic" behaviour in the tool.
**Implications:** Lives in the generator's endpoint resolution and is **independent of the `@layout` epic** — it can ship on the current Go pipeline as a standalone improvement. Open: 4 vs 8 directions; whether an unpositioned *named* anchor should error instead of defaulting to centre.

## 2026-06-17 — Separate semantics from layout (CSS without the cascade)
**Choice:** Author diagrams in two layers — a **semantic** layer (the `.ark` structure: node `id`/`label`/`kind`, named anchors, containment, arrows) and a **layout** layer (`@layout`: `direction`, `size`, anchor positions, `box`, child arrangement) — using `@`-prefixed directives. Collapse node and group into **one node type** (`box: none` replaces the layout-only group). Add exact-path selectors, opt-in reusable blocks (`@block`/`@use`), a semantic `kind` that implicitly `@use`s the block of the same name, and anonymous presentational regrouping (`@group`). Deliberately omit CSS's cascade.
**Why:** Semantic structure is stable; layout is fiddly and frequently retuned. Separating them lets layout be edited, reused, and even swapped (multiple layouts over one model) without disturbing semantics. The separation is modelled on HTML/CSS, but CSS's *pain* — "why is this here?" answered three selectors away — comes from the cascade (specificity, inheritance, competing rules), which is the exact action-at-a-distance the core principle forbids. So we keep the separation and drop the cascade. `@` was chosen over `!` for the CSS at-rule feel and to avoid the `!important` connotation.
**The rules that preserve determinism:**
- Selectors are **exact dotted paths** only — no wildcards, specificity, or inheritance. One selector names one node.
- Resolution has two precedence tiers: **imported** (`kind` baseline, then `@use` — source order wins within) is overridden by **direct** declarations naming the node (inline or sheet). Direct-over-imported is not a conflict; two *direct* rules setting the same property on the same node is a **validation error** (relaxable later to import-order-wins for multi-sheet themes).
- `@block`/`@use` is opt-in and explicit, parameterless in v1; cycles are an error.
- `kind` expands to an implicit `@use <kind>` at **lowest precedence** (a baseline); explicit layout overrides it without error. A small set of built-in kinds ships (e.g. `invisible` → `box: none`) and any kind can be redeclared.
- An anonymous `@group` may regroup only **direct semantic children of the enclosing node**, and an arrangement must reference each child **exactly once** — so the layout tree is always a *refinement* of the semantic tree.
- Anchors split into a semantic **name** (arrows reference it) and a layout **position**.
**Implications:** A new **resolve** stage merges the layout layer onto the semantic tree by exact path (the analog of CSS computed style) before generation; the validator gains selector/conflict/`@group`/`@use`/`kind`/anchor-name checks; the generator reads resolved layout rather than properties on the semantic node. This is a breaking syntax change (today's inline `size`/`direction`/`anchors` and the `group` keyword go away or become layout shorthand), and former groups now carry an `id` and a path segment instead of being path-transparent — acceptable pre-1.0. Built in phases (see roadmap): split → reuse (+`kind`) → regrouping. Remaining details (unpositioned-anchor default, unknown-kind handling, inline-vs-sheet precedence, cross-file sheets, arrow-anchor routing) are tracked in `design.md`'s open questions.

## 2026-06-17 — Rewrite the implementation in Go
**Choice:** Port Arkitecture from TypeScript to Go as a single switchover PR, structured library-first: a root `arkitecture` package is the library, with `cmd/arkitecture` (CLI) and `wasm/` (a `js,wasm` build) as thin wrappers over it. Supersedes "Pure TypeScript library with a thin CLI" (2025-06-19).
**Why:** Go compiles a single portable static binary with no runtime to install, which suits a CLI tool better than an npm package. It also has a first-class `GOOS=js GOARCH=wasm` target, giving a clean path to a WASM library for future TypeScript/browser interop without maintaining a second implementation. The pipeline is pure string-in/string-out logic, so it ports almost mechanically.
**Why Go over the alternatives:** Rust was the other portable-binary candidate; Go was chosen for faster porting of this loosely-typed code, a simpler build, and a mature `syscall/js` WASM story. Staying on TypeScript was rejected because "single binary, no Node required" was the whole motivation.
**Implications:** The CLI and the WASM shim must depend on the library, never reimplement it. The error model stays "collected as data, never thrown across stages" (Go: `[]ast.Error`, no panics across boundaries; recover only at the top level). The `string-width` text-measurement decision (2025-06-19) will be re-decided during the generator port — likely a small Go rune-width function — since the npm package can't come along. The port lands stage by stage on one branch: tokenizer + parser first; validator, generator, and CLI watch before the PR merges.

## 2026-06-17 — Go text measurement: a built-in rune-width approximation
**Choice:** Estimate label dimensions with a small built-in display-width function (rune count, East-Asian-wide/emoji counted as 2, combining/format marks as 0) — the same shape as `string-width` — rather than adding a dependency or measuring real font metrics. Resolves the open item from the Go rewrite ADR (supersedes the 2025-06-19 `string-width` decision).
**Why:** Layout must run headless and in WASM without a canvas, and the output must be deterministic across platforms. A built-in width keeps the module dependency-free (the portable-binary/WASM goal) and reproduces the TypeScript output exactly for ASCII — confirmed by the byte-for-byte golden tests.
**Implications:** Measurement is an approximation, not pixel-accurate metrics — fine for box packing. `generator/text.go` is the single seam to replace if true metrics are ever needed.

## 2026-06-17 — Watch mode: stdlib modtime polling, not fsnotify
**Choice:** Implement `--watch` by polling the input file's modification time (200ms) in the CLI, rather than depending on `fsnotify`.
**Why:** Watch is a CLI-only dev convenience; polling a single file is simple, dependency-free, and cross-compiles cleanly — consistent with the minimal-dependency posture. Event-based watching would add a dependency for negligible benefit at this scale.
**Implications:** Up to ~200ms latency before a change is picked up (imperceptible in practice). Revisit with fsnotify only if multi-file or directory watching is ever needed.

## 2026-06-17 — Adopt the project-template docs structure
**Choice:** Replace the original `specs/` + `.specs/` specifications, `prompt-plan.md`, and `.claude/commands/dev.md` workflow with `CLAUDE.md` + `docs/{design,architecture,roadmap,decisions}.md`, matching [kurrik/project-template](https://github.com/kurrik/project-template).
**Why:** The original scaffolding was a one-shot architecture spec plus a 15-step prompt plan that drove the initial build and then went stale; it duplicated content (`specs/` and `.specs/` held near-identical copies) and the prompt plan no longer reflected the shipped code. The project-template layout separates durable concerns (vision / layout / status / decisions) and is the shared convention across the owner's projects.
**Implications:** `docs/` is now the source of truth and must be updated in the same commit as the change it describes. The `specs/` tree is gone; the annotated DSL reference survives as `examples/annotated.ark`. Future work is steered by `CLAUDE.md` + `docs/`, not slash-command playbooks.

## 2025-06-19 — Manual layout instead of automatic graph layout
**Choice:** The author controls layout via nesting, `direction`, and `size`; the tool only measures text and packs boxes deterministically.
**Why:** Auto-layout tools (Graphviz, Mermaid) produce output you can't precisely control and can't correct without fighting the algorithm. The goal is predictable, hand-tuned high-level diagrams (e.g. DDD bounded contexts) kept as text in version control.
**Implications:** No layout engine to fight — but also no auto-routing or auto-placement; arrangement is the author's responsibility. Rules out force-directed / hierarchical layout features and curved auto-routing.

## 2025-06-19 — Pure TypeScript library with a thin CLI
**Choice:** Ship a side-effect-free library (DSL string in → `Result` out) compiled to CommonJS, with all file I/O isolated in the CLI.
**Why:** Keeps the core runnable in both Node and the browser, trivially testable, and reusable as an SDK; the CLI is just an adapter over the same function.
**Implications:** No filesystem or DOM assumptions inside the library; only the CLI touches disk (plus `chokidar` for `--watch`). The public API and the CLI must stay in sync as the pipeline evolves.

## 2025-06-19 — Collected errors, never thrown across stages
**Choice:** Every failure is a `ValidationError` (`line`, `column`, `message`, `type`); stages return arrays and don't fail-fast. Only the top-level entry point wraps an unexpected throw.
**Why:** Authors want to see *all* problems from one run, with positions, instead of fix-one-rerun. A single uniform error shape serves both the CLI and the API.
**Implications:** Each stage must accumulate rather than abort, and a run reports syntax, reference, and constraint errors together. Throwing across a stage boundary is a bug.

## 2025-06-19 — `string-width` for text measurement
**Choice:** Estimate label dimensions with the `string-width` package (default Arial 12px, 1.2× line height) rather than DOM/canvas measurement.
**Why:** Layout needs box sizes *before* rendering, and the library must work headless in Node and the browser without a canvas. `string-width` gives consistent, dependency-light width estimates everywhere.
**Implications:** Measurement is a cell-width approximation, not true font metrics — fine for box packing, and the font is overridable via options. If pixel-accurate text fitting is ever required, this is the seam to revisit.

## 2025-06-19 — Golden-file tests for SVG output
**Choice:** Render `.ark` fixtures and diff against checked-in `.svg`/`.error` references, regenerated via `npm run golden:generate`.
**Why:** SVG output is large and positional; golden files catch unintended rendering changes that hand-written assertions would miss, while staying easy to review as diffs.
**Implications:** Intentional output changes require regenerating and reviewing the fixtures; the generator's exact output is part of the test contract.
