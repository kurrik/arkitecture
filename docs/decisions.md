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
