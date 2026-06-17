# Architecture

How the code is laid out and how the runtime fits together. Update when the
structure changes — adding a module, introducing a stage, or establishing a new
pattern.

## App shape

Arkitecture is a TypeScript library with a thin CLI on top, compiled with `tsc` to
CommonJS in `dist/`. The library is pure and side-effect-free — given a DSL string
it returns a result object — so it runs unchanged in Node and the browser. The CLI
(`bin/arkitecture` → `src/cli`) adds file I/O, flags, and watch mode.

Processing is a one-way pipeline with no shared mutable state:

```
DSL text → Tokenizer → Parser → Validator → Layout → SVG Generator → SVG
            (Token[])   (Document) (errors?)  (boxes)                 (string)
```

Each stage is a pure function of its input. Errors are *collected*, never thrown
across stage boundaries (the top-level entry point wraps any unexpected throw into
a `ValidationError`), so a single run can report every problem it finds.

## Layout

```
src/
  types.ts              # AST + result interfaces — the shared vocabulary
  index.ts              # public surface: re-exports everything below
  arkitecture.ts        # arkitectureToSVG() — wires the pipeline together
  parser/
    tokenizer.ts        # DSL text → Token[]
    parser.ts           # Token[] → Document (AST)
    index.ts            # parseArkitecture(): tokenizer + parser
  validator/
    validator.ts        # Document → ValidationError[]
    index.ts            # validate()
  generator/
    text-measurement.ts # label text → {width, height} (via string-width)
    layout.ts           # Document → positioned/sized boxes + anchor coords
    svg-generator.ts    # Document + layout → SVG string
    index.ts            # generator barrel
  cli/
    index.ts            # argument parsing, file I/O, watch mode
bin/arkitecture           # executable shim that invokes the CLI
tests/                    # mirrors src/ (parser/, validator/, generator/, cli/) + golden/
examples/                 # sample .ark inputs (+ generated .svg)
scripts/generate-golden.ts  # regenerates golden .svg fixtures
```

Dependency direction is one-way: `cli → arkitecture → {parser, validator,
generator} → types`. The generator depends only on `types`; nothing depends on the
CLI. Keep folders flat until a stage genuinely needs sub-modules.

## Domain model

The AST (`src/types.ts`) is the contract every stage shares:

- **`Document`** — `{ nodes: ContainerNode[]; arrows: Arrow[] }`. Nodes and arrows
  are parsed in two phases, so arrows live in a flat list rather than on nodes.
- **`ContainerNode`** — `id`, optional `label` / `direction` / `size` / `anchors`,
  and `children` (nodes or groups).
- **`GroupNode`** — like a node but with no `id`/`label`: layout-only.
- **`Arrow`** — `source` and `target` strings, each a dotted node path with an
  optional `#anchor` suffix (e.g. `c1.n2 --> c1.n3#a1`). References stay as
  *strings* at parse time; the validator resolves them.
- **`ValidationError`** — `line`, `column`, `message`, and a `type` of `syntax` |
  `reference` | `constraint`. Every failure in any stage is reported as one of
  these.

Result shapes: `ParseResult` (parser), `ValidationError[]` (validator), and
`Result` (`{ success, svg?, errors }`) from the top-level API.

## Pipeline stages

- **Tokenizer** (`parser/tokenizer.ts`) — hand-written scanner producing `Token`s
  with line/column info: identifiers, strings, numbers, structural punctuation,
  the `-->` arrow, and `#` comments.
- **Parser** (`parser/parser.ts`) — recursive-descent build of the `Document`:
  container nodes, nested children, layout-only groups, `size`/`anchors`
  properties, then arrows in a second phase. Produces a `ParseResult`, collecting
  syntax errors with positions.
- **Validator** (`validator/validator.ts`) — semantic checks over a parsed
  `Document`: ID uniqueness within a scope, arrow source/target resolution, anchor
  existence, and range constraints (`size` and anchor coords in `[0, 1]`).
  Non-fail-fast: returns all errors.
- **Text measurement** (`generator/text-measurement.ts`) — wraps `string-width` to
  estimate label bounds for a given font (default Arial 12px, 1.2× line height for
  multi-line labels). The one place layout depends on font metrics.
- **Layout** (`generator/layout.ts`) — bottom-up sizing then top-down positioning,
  applying the vertical/horizontal rules and `size` overrides, and resolving each
  anchor's relative `[x, y]` to absolute canvas coordinates. Canvas = exact bounds
  of all top-level nodes.
- **SVG generator** (`generator/svg-generator.ts`) — emits `<rect>` + `<text>` per
  visible node (groups render nothing) and `<line>` + arrowhead `<marker>` per
  arrow, using layout coordinates.

## Public API

`src/index.ts` is the only supported surface:

- `arkitectureToSVG(dsl, options?) → Result` *(default export)* — the whole
  pipeline; honours `validateOnly`, `fontSize`, `fontFamily`.
- `parseArkitecture(dsl) → ParseResult` — tokenize + parse only.
- `validate(document) → ValidationError[]` — semantic checks on an AST.
- `generateSVG(document, options?) → string` — layout + SVG for an already-parsed
  document.
- All AST/result types are re-exported.

## Persistence

None. Arkitecture is stateless: input is a `.ark` file (or string), output is an
SVG string. The CLI is the only component that touches the filesystem — reading
the input, writing the output, and (in `--watch` mode) subscribing to changes via
`chokidar`. There is no database, config file, or cache.

## CLI

`src/cli` parses arguments with `commander`, reads the input file, runs
`arkitectureToSVG`, and writes the SVG (defaulting the output name to the input
with a `.svg` extension). Flags: `--validate-only`, `--verbose`, `--watch`,
`--font-size`, `--font-family`, `--help`, `--version`. Exit codes: `0` success,
`1` validation errors, `2` filesystem errors. `chalk` colourises diagnostics.

## Concurrency

Single-threaded. The library is synchronous and pure. The only asynchrony is the
CLI's watch loop (`chokidar` file events), which debounces and re-runs the same
synchronous pipeline per change; runs never overlap.

## Build & test

`tsc` compiles `src/` and `scripts/` to `dist/`. Tests run on Jest + ts-jest and
mirror the source tree, plus a **golden** suite: `.ark` fixtures in
`tests/golden/examples/` are rendered and diffed against checked-in `.svg`/
`.error` references. Regenerate them with `npm run golden:generate` after an
*intentional* output change, and review the diff. CI runs lint + build + coverage
on Node 20.x and 22.x.
