# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

**Arkitecture** is a Go DSL library and CLI that compiles a small `.ark` text file into an SVG architecture diagram, giving the author manual layout control instead of automatic graph layout.

See [docs/design.md](docs/design.md) for the product model and [docs/architecture.md](docs/architecture.md) for the code layout. When the user gives durable direction, write it into `docs/design.md` immediately.

Target platform: Go 1.23+ (CI covers 1.23 and 1.24). Ships as a single portable binary and, from the same library, as a `GOOS=js GOARCH=wasm` module for JS/TS interop. The library is pure and side-effect-free.

> 🚧 **TypeScript → Go rewrite on one branch toward a single switchover PR.** The full pipeline is ported and at parity with the old TypeScript; the PR awaits review and merge, which flips `main` to Go. The pre-rewrite TypeScript remains in git history as the reference. See [docs/roadmap.md](docs/roadmap.md).

## Edit discipline

This repo uses an ordinary feature-branch workflow — there is no `.claude/worktrees` setup.

- Branch off `main` (`feature/<desc>`, `fix/<desc>`, `chore/<desc>`); don't commit straight to `main`.
- The Go rewrite accumulates on a single branch and merges only when it reaches parity with the old TypeScript — keep porting on that branch rather than opening parallel PRs.
- Compiled binaries, `*.wasm`, and `coverage.out` are build artifacts (git-ignored) — never commit them.
- Keep `git status` free of incidental files; commit only what the change needs.

## Commands

- Build (all): `go build ./...`
- Build WASM: `GOOS=js GOARCH=wasm go build -o arkitecture.wasm ./wasm`
- Test (all): `go test ./...`
- Test (one package): `go test ./parser`
- Test (one test): `go test ./parser -run TestParseArrows`
- Test (race + coverage): `go test -race -coverprofile=coverage.out ./...`
- Update golden fixtures: `go test . -run TestGolden -update` (only after an *intentional* output change — review the diff)
- Run (CLI): `go run ./cmd/arkitecture input.ark output.svg`
- Format: `gofmt -w .` (check: `gofmt -l .`)
- Vet: `go vet ./...`

Run **gofmt + vet + test** before considering work done.

## Architecture

A one-way, side-effect-free pipeline: each stage is a pure function of its input, and failures are *collected* as `[]ast.Error` rather than thrown across stages. The CLI and WASM builds are thin wrappers over the library — never put compilation logic in them. See [docs/architecture.md](docs/architecture.md) and keep it in sync as structure changes.

- **`ast`** — the syntax tree + the shared `Error` type. No dependencies (this is what avoids an import cycle between the root package and the stages).
- **`parser`** — tokenizer + recursive-descent build of the `ast.Document`.
- **`validator`** — semantic checks (references, ID uniqueness, range constraints); returns *all* errors, non-fail-fast.
- **`generator`** — text measurement → bottom-up layout + anchor resolution → SVG string.
- **`arkitecture` (root)** — `ToSVG()` wires the pipeline; `Parse`/`Validate`/`GenerateSVG` expose the stages; the AST types are re-exported as aliases.
- **`cmd/arkitecture`** / **`wasm/`** — CLI and WASM entry points over the library.

### Persistence

None — Arkitecture is stateless. Input is a `.ark` string/file; output is an SVG string. Only the CLI touches the filesystem.

### Concurrency

Single-threaded and synchronous. The library has no goroutines. The only asynchrony is the CLI watch loop (a modtime poller re-running the same synchronous pipeline per change); runs never overlap.

## Language & framework conventions

- Standard Go style; `gofmt` is mandatory and CI-enforced. Run `go vet ./...`.
- The `ast` package is the shared contract between stages — change it deliberately. Use pointers/zero values so "unset" stays distinguishable from a real value.
- Errors are data: every failure is an `ast.Error` (`Line`, `Column`, `Message`, `Type`) collected into a slice. Don't `panic` across stage boundaries; only the top-level `ToSVG` recovers an unexpected panic and wraps it.
- Output must stay deterministic: sort before iterating maps (e.g. anchors), never let map order leak into SVG.
- Keep stages pure — a function of input with no global state. The CLI/WASM wrappers depend on the library, never the reverse.
- Keep packages flat until a stage genuinely needs sub-packages; prefer the standard library (the portable-binary and WASM goals reward minimal dependencies).

## Testing

- Standard `testing` package, table-driven where it helps. Tests live beside the code (`parser/parser_test.go`); external-API tests use `package arkitecture_test`.
- **Golden test** (`golden_test.go`) renders `generator/testdata/golden/*.ark` through the full pipeline and diffs against checked-in `.svg`/`.error` references. When output changes *intentionally*, run `go test . -run TestGolden -update` and review the diff before committing.
- Add tests with each behavioural change. Test the pipeline stages and pure logic; let golden fixtures cover the exact SVG output rather than asserting it byte-for-byte inline.

## Project state in `docs/`

[docs/](docs/) is the durable record of *why* the project looks the way it does. Code answers "what"; `docs/` answers "what are we building, what's next, and why did we make these calls".

- [docs/design.md](docs/design.md) — the project's vision, target user, and core workflow. Update when the high-level concept shifts.
- [docs/architecture.md](docs/architecture.md) — code layout, layer roles, and key patterns. Update when modules, types, or structure change.
- [docs/roadmap.md](docs/roadmap.md) — what's done, in progress, and planned. Update when scope changes or items move between sections.
- [docs/decisions.md](docs/decisions.md) — append-only log of non-obvious technical or design choices and the reasoning behind them. New entry per decision; never rewrite history.

### When to update `docs/`

Update these files **as part of the change that makes them out of date**, not as a separate task:

- **After landing a feature** → mark it done in [docs/roadmap.md](docs/roadmap.md); reflect any new capability in [docs/design.md](docs/design.md); update [docs/architecture.md](docs/architecture.md) if the change added a package, type, or pattern.
- **After a non-obvious call** → append an entry to [docs/decisions.md](docs/decisions.md) with date, the choice, and *why*.
- **When the user gives durable direction** about vision, scope, or constraints → write it into [docs/design.md](docs/design.md) or [docs/roadmap.md](docs/roadmap.md) immediately, before continuing to code.

**What not to write down**: routine refactors, bugfixes, anything obvious from the diff, or step-by-step task state for the current conversation. Git history and the code cover those — `docs/` is for state that persists past a session.

## Commits & PRs

Land work in **logical, reviewable chunks**. One commit = one coherent change; one PR = one feature or standalone piece of architecture (if you need the word "and" to describe it, consider splitting).

[Conventional Commits](https://www.conventionalcommits.org/):

    <type>(<optional scope>): <short imperative summary, lowercase, no period>

    <optional body explaining the why>

Types: `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, `style`, `chore`, `build`, `ci`. Breaking change → append `!`. Scope identifies the area (e.g. `feat(parser): support multi-line labels`).

- Commit after each logical change passes gofmt, vet, and the tests. Don't bundle unrelated changes.
- Update [docs/](docs/) **in the same commit** as the change it describes.
- Branches: `feature/<desc>`, `fix/<desc>`, `chore/<desc>` (kebab-case).
- Prefer smaller, frequent PRs over one giant branch (the Go rewrite is the deliberate exception — one switchover branch).

## Notes

- Output is intentionally plain (white fill, 1px black border, one font); styling and theming are out of scope for v1 — see [docs/design.md](docs/design.md).
- The defining constraint is *manual, deterministic layout*: never introduce automatic placement that moves elements the author didn't position.
- Favor clarity over premature optimization.
