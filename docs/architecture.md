# Architecture

How the code is laid out and how the runtime fits together. Update when the
structure changes — adding a package, introducing a stage, or establishing a new
pattern.

## App shape

Arkitecture is a Go library with two thin wrappers on top: a CLI
(`cmd/arkitecture`) and a WebAssembly shim (`wasm/`, built with
`GOOS=js GOARCH=wasm`). The library is pure and side-effect-free — given a DSL
string it returns a result — so it runs identically in a binary, in tests, and in
the browser via WASM. All file I/O lives in the CLI.

Processing is a one-way pipeline with no shared mutable state:

```
DSL text → Tokenizer → Parser → Validator → Generator → SVG
            ([]Token)   (Document) (errors?)  (layout+SVG)  (string)
```

Each stage is a pure function of its input. Errors are *collected* as
`[]ast.Error`, never thrown across stage boundaries; the top-level `ToSVG`
recovers from any unexpected panic and returns it as one error, so a single run
reports every problem it finds.

## Layout

```
github.com/kurrik/arkitecture        (module)
  arkitecture.go     package arkitecture — public API (ToSVG/Parse/Validate/
                     GenerateSVG); re-exports the AST types as aliases
  ast/               package ast — Document, ContainerNode, GroupNode, Arrow,
                     Error, ParseResult; no deps, so every stage can import it
  parser/            tokenizer + recursive-descent parser → ast.Document
  validator/         Document → []ast.Error (references, ID uniqueness, ranges)
  generator/         Document → SVG string (text measurement, layout, emit)
    testdata/golden/ .ark fixtures + .svg/.error references for the golden test
  cmd/arkitecture/   package main — the CLI (flags, file I/O, watch); imports the library
  wasm/              package main — js,wasm shim exposing ToSVG to JS (+ host stub)
examples/            sample .ark inputs
```

Dependency direction is one-way: `cmd`/`wasm` → `arkitecture` → `{parser,
validator, generator}` → `ast`. The `ast` package has no dependencies, which is
what lets the root package and the stage packages share types without an import
cycle. Nothing depends on the CLI. Keep packages flat until a stage genuinely
needs sub-packages.

## Domain model

The AST (`ast` package) is the contract every stage shares:

- **`Document`** — `{ Nodes []*ContainerNode; Arrows []Arrow }`. Nodes and arrows
  are parsed in two phases, so arrows are a flat list, not attached to nodes.
- **`Node`** — an interface implemented by `*ContainerNode` and `*GroupNode`, so a
  node's `Children` can hold either.
- **`ContainerNode`** — `ID`, optional `Label`/`Direction`/`Size`/`Anchors`, and
  `Children`. Optionals are pointers/zero values so "unset" stays distinguishable.
- **`GroupNode`** — a `Node` with only `Direction` and `Children`: layout-only.
- **`Arrow`** — `Source` and `Target` strings, each a dotted node path with an
  optional `#anchor` suffix (e.g. `c1.n2 --> c1.n3#a1`). Resolved by the validator.
- **`Error`** — `Line`, `Column`, `Message`, and `Type` (`syntax` | `reference` |
  `constraint`). Every failure is one of these; it also satisfies `error`.

## Pipeline stages

- **Tokenizer** (`parser/tokenizer.go`) — hand-written rune scanner producing
  `Token`s with line/column info: identifiers, strings (with escapes), numbers,
  structural punctuation, the `-->` arrow, `#` (anchor vs comment), and newlines.
  Returns a `*TokenizerError` on an unexpected character or unterminated string.
- **Parser** (`parser/parser.go`) — recursive-descent build of the `Document`:
  container nodes, nested children, layout-only groups, `size`/`anchors`, then
  arrows in a second phase. Collects syntax and range errors with positions and
  recovers to keep going. `parser.Parse` wires the tokenizer and parser together.
- **Validator** (`validator/validator.go`) — semantic checks over a parsed
  `Document`: ID uniqueness within a scope (groups don't introduce a scope), arrow
  source/target resolution against a flat node map, anchor existence (with the
  implicit `center`), and range constraints; non-fail-fast. Diagnostics report at
  line 1, column 1 — the AST carries no node positions.
- **Generator** (`generator/`) — `text.go` measures labels with a deterministic,
  dependency-free rune-width approximation (no font metrics); `layout.go` sizes
  bottom-up applying the vertical/horizontal rules and `size` overrides, positions
  top-down, sizes the canvas to fit, and resolves anchor coordinates; `svg.go`
  emits `<rect>` + `<text>` per visible node (groups render nothing) and `<line>`
  + arrowhead `<marker>` per arrow. Output is byte-for-byte stable.

## Public API

`arkitecture.go` (package `arkitecture`) is the supported surface:

- `ToSVG(dsl, *Options) Result` — the whole pipeline; honours `ValidateOnly`,
  `FontSize`, `FontFamily`. Never panics across stages.
- `Parse(dsl) ParseResult` — tokenize + parse only.
- `Validate(*Document) []Error` — semantic checks on an AST.
- `GenerateSVG(*Document, *Options) (string, []Error)` — layout + SVG.
- The AST/diagnostic types are re-exported as aliases (`arkitecture.Document`,
  `arkitecture.Error`, …), so callers need not import `ast` directly.

## Persistence

None. Arkitecture is stateless: input is a `.ark` string/file, output is an SVG
string. Only the CLI touches the filesystem (read input, write SVG, poll the
input in watch mode). There is no database, config file, or cache.

## CLI

`cmd/arkitecture` parses arguments with the standard-library `flag` package
(flags may appear before or after the input/output paths), reads the input, runs
`arkitecture.ToSVG`, and writes the SVG (defaulting the output to the input name
with a `.svg` extension). Flags: `--validate-only`, `--verbose`, `--watch`,
`--font-size`, `--font-family`, `--version`. Exit codes: `0` success, `1`
validation/parse errors, `2` filesystem errors. `--watch` re-renders on change
using a stdlib modtime poller and stops on SIGINT/SIGTERM.

## Concurrency

Single-threaded and synchronous. The library has no goroutines. The only
asynchrony is the CLI watch loop — a `time.Ticker` polling the input's modtime,
plus a signal channel for shutdown — which re-runs the same synchronous pipeline
per change; runs never overlap.

## Build & test

`go build ./...` builds the library, CLI, and the host wasm stub;
`GOOS=js GOARCH=wasm go build ./wasm` builds the real WASM module. Tests use the
standard `testing` package (table-driven). The **golden** test renders
`generator/testdata/golden/*.ark` through the full pipeline and diffs against the
checked-in `.svg`/`.error` references, regenerated with `-update`. CI runs gofmt +
vet + `go test -race` + the CLI and WASM builds on Go 1.23 and 1.24.
