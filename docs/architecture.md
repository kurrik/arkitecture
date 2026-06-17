# Architecture

How the code is laid out and how the runtime fits together. Update when the
structure changes — adding a package, introducing a stage, or establishing a new
pattern.

> 🚧 **Rewrite in progress (TypeScript → Go).** The module, AST, tokenizer, and
> parser are ported. The validator and generator are stubs pending their ports;
> this document describes the target Go structure and flags what is still a stub.

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
  validator/         Document → []ast.Error            (stub: port pending)
  generator/         Document → SVG string             (stub: port pending)
    testdata/golden/ .ark fixtures + .svg/.error references for the port
  cmd/arkitecture/   package main — the CLI (flags, file I/O); imports the library
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
- **Validator** (`validator/`) — *stub.* Will check ID uniqueness within a scope,
  arrow/anchor reference resolution, and range constraints; non-fail-fast.
- **Generator** (`generator/`) — *stub.* Will do text measurement → bottom-up
  layout + anchor resolution → SVG emission, reproducing the golden fixtures.

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
string. Only the CLI touches the filesystem (read input, write SVG). There is no
database, config file, or cache.

## CLI

`cmd/arkitecture` parses arguments with the standard-library `flag` package
(flags may appear before or after the input/output paths), reads the input, runs
`arkitecture.ToSVG`, and writes the SVG (defaulting the output to the input name
with a `.svg` extension). Flags: `--validate-only`, `--verbose`, `--watch`
(pending), `--font-size`, `--font-family`, `--version`. Exit codes: `0` success,
`1` validation/parse errors, `2` filesystem errors.

## Concurrency

Single-threaded and synchronous. The library has no goroutines. Watch mode
(`--watch`), once ported, will be the only asynchrony — a debounced file-event
loop re-running the same synchronous pipeline; runs never overlap.

## Build & test

`go build ./...` builds the library, CLI, and the host wasm stub;
`GOOS=js GOARCH=wasm go build ./wasm` builds the real WASM module. Tests use the
standard `testing` package (table-driven). The generator port will add a
**golden** test that renders `generator/testdata/golden/*.ark` and diffs against
the checked-in `.svg`/`.error` references, regenerated with a `-update` flag. CI
runs gofmt + vet + `go test -race` + the CLI and WASM builds on Go 1.23 and 1.24.
