# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

**Arkitecture** is a TypeScript DSL and CLI that compiles a small `.ark` text file into an SVG architecture diagram, giving the author manual layout control instead of automatic graph layout.

See [docs/design.md](docs/design.md) for the product model and [docs/architecture.md](docs/architecture.md) for the code layout. When the user gives durable direction, write it into `docs/design.md` immediately.

Target platform: Node.js 20+ (CI covers 20.x and 22.x); the library is pure and also runs in the browser. Language/toolchain: TypeScript 5 (`strict`), compiled with `tsc` to CommonJS in `dist/`.

## Edit discipline

This repo uses an ordinary feature-branch workflow — there is no `.claude/worktrees` setup.

- Branch off `main` (`feature/<desc>`, `fix/<desc>`, `chore/<desc>`); don't commit straight to `main`.
- Edit `src/` (and `scripts/`, `tests/`), then build. `dist/`, `coverage/`, and `node_modules/` are generated and git-ignored — never edit or commit them.
- Keep `git status` free of incidental files; commit only what the change needs.

## Commands

- Build: `npm run build` (watch: `npm run dev`)
- Test (all): `npm test`
- Test (one): `npm test -- tests/parser/parser.test.ts` (or `--testNamePattern="anchor"`)
- Test (coverage): `npm run test:coverage`
- Run (CLI): `./bin/arkitecture input.ark output.svg` (after `npm run build`)
- Lint: `npm run lint` (ESLint over `src/` + `scripts/`)
- Format: `npm run format` (Prettier over `src/`)
- Regenerate golden fixtures: `npm run golden:generate` (only after an *intentional* output change — review the diff)

Run **format + lint + test** before considering work done.

## Architecture

A one-way, side-effect-free pipeline: each stage is a pure function of its input, and failures are *collected* as errors rather than thrown across stages. See [docs/architecture.md](docs/architecture.md) and keep it in sync as structure changes.

- **Parser** (`src/parser`) — tokenizer + recursive-descent build of the `Document` AST.
- **Validator** (`src/validator`) — semantic checks (references, ID uniqueness, range constraints); returns *all* errors, non-fail-fast.
- **Generator** (`src/generator`) — text measurement → bottom-up layout + anchor resolution → SVG string.
- **API / CLI** (`src/arkitecture.ts`, `src/cli`) — `arkitectureToSVG()` wires the pipeline; the CLI adds file I/O, flags, and `--watch`.

### Persistence

None — Arkitecture is stateless. Input is a `.ark` string/file; output is an SVG string. Only the CLI touches the filesystem (read input, write SVG, watch via `chokidar`).

### Concurrency

Single-threaded and synchronous. The only asynchrony is the CLI watch loop (debounced `chokidar` events re-running the same synchronous pipeline); runs never overlap.

## Language & framework conventions

- TypeScript `strict`; prefer precise types over `any`. The AST in `src/types.ts` is the shared contract between stages — change it deliberately.
- Errors are data: every failure is a `ValidationError` (`line`, `column`, `message`, `type`) collected into an array. Don't `throw` across stage boundaries; the top-level entry point is the only place that wraps an unexpected throw.
- Keep stages pure — a stage is a function of its input with no global state.
- Keep folders flat until a stage genuinely needs sub-modules.
- Run Prettier + ESLint before declaring work done; don't silence a lint inline unless it's wrong here.

## Testing

- Jest + ts-jest. Tests live under `tests/`, mirroring `src/` (`parser/`, `validator/`, `generator/`, `cli/`).
- **Golden tests** (`tests/golden/`) render `.ark` fixtures and diff against checked-in `.svg`/`.error` references. When output changes *intentionally*, run `npm run golden:generate` and review the diff before committing the new fixtures.
- Add tests with each behavioural change. Test the pipeline stages and pure logic; let golden fixtures cover the exact SVG output rather than asserting byte-for-byte inline.

## Project state in `docs/`

[docs/](docs/) is the durable record of *why* the project looks the way it does. Code answers "what"; `docs/` answers "what are we building, what's next, and why did we make these calls".

- [docs/design.md](docs/design.md) — the project's vision, target user, and core workflow. Update when the high-level concept shifts.
- [docs/architecture.md](docs/architecture.md) — code layout, layer roles, and key patterns. Update when modules, types, or structure change.
- [docs/roadmap.md](docs/roadmap.md) — what's done, in progress, and planned. Update when scope changes or items move between sections.
- [docs/decisions.md](docs/decisions.md) — append-only log of non-obvious technical or design choices and the reasoning behind them. New entry per decision; never rewrite history.

### When to update `docs/`

Update these files **as part of the change that makes them out of date**, not as a separate task:

- **After landing a feature** → mark it done in [docs/roadmap.md](docs/roadmap.md); reflect any new capability in [docs/design.md](docs/design.md); update [docs/architecture.md](docs/architecture.md) if the change added a module, type, service, or pattern.
- **After a non-obvious call** → append an entry to [docs/decisions.md](docs/decisions.md) with date, the choice, and *why*.
- **When the user gives durable direction** about vision, scope, or constraints → write it into [docs/design.md](docs/design.md) or [docs/roadmap.md](docs/roadmap.md) immediately, before continuing to code.

**What not to write down**: routine refactors, bugfixes, anything obvious from the diff, or step-by-step task state for the current conversation. Git history and the code cover those — `docs/` is for state that persists past a session.

## Commits & PRs

Land work in **logical, reviewable chunks**. One commit = one coherent change; one PR = one feature or standalone piece of architecture (if you need the word "and" to describe it, consider splitting).

[Conventional Commits](https://www.conventionalcommits.org/):

    <type>(<optional scope>): <short imperative summary, lowercase, no period>

    <optional body explaining the why>

Types: `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, `style`, `chore`, `build`, `ci`. Breaking change → append `!`. Scope identifies the area (e.g. `feat(parser): support multi-line labels`).

- Commit after each logical change passes format, lint, and the tests. Don't bundle unrelated changes.
- Update [docs/](docs/) **in the same commit** as the change it describes.
- Branches: `feature/<desc>`, `fix/<desc>`, `chore/<desc>` (kebab-case).
- Prefer smaller, frequent PRs over one giant branch.

## Notes

- Output is intentionally plain (white fill, 1px black border, one font); styling and theming are out of scope for v1 — see [docs/design.md](docs/design.md).
- The defining constraint is *manual, deterministic layout*: never introduce automatic placement that moves elements the author didn't position.
- Favor clarity over premature optimization.
