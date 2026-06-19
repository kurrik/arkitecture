# Arkitecture

[![CI](https://github.com/kurrik/arkitecture/actions/workflows/ci.yml/badge.svg)](https://github.com/kurrik/arkitecture/actions/workflows/ci.yml)

A domain-specific language (DSL) for generating SVG architecture diagrams with precise, manual positioning control. Arkitecture is built for high-level diagrams — Domain-Driven Design boundaries, bounded-context relationships, system overviews — where you want the layout to look exactly the way you arranged it.

Unlike tools that use automatic layout algorithms, Arkitecture gives you fine-grained control over element positioning and sizing: you describe the structure and the layout, and the tool only measures text and packs boxes — deterministically.

📖 **[Project site, syntax reference, and live examples →](https://kurrik.github.io/arkitecture/)**

## Documentation

- **[CLAUDE.md](CLAUDE.md)** — how to work in this repo (commands, conventions, workflow)
- **[docs/design.md](docs/design.md)** — what Arkitecture is and who it's for
- **[docs/architecture.md](docs/architecture.md)** — code layout and the processing pipeline
- **[docs/roadmap.md](docs/roadmap.md)** — done / in progress / planned
- **[docs/decisions.md](docs/decisions.md)** — why the key technical choices were made

## Requirements

- Go 1.23+

## Build / test

```bash
go build ./...     # build the library, CLI, and host wasm stub
go test ./...      # run all tests
go run ./cmd/arkitecture diagram.ark diagram.svg
```

Build the WebAssembly library with the standard JS/WASM target:

```bash
GOOS=js GOARCH=wasm go build -o arkitecture.wasm ./wasm
```

## Command-line usage

```bash
# Generate an SVG (output defaults to the input name with a .svg extension)
arkitecture diagram.ark diagram.svg
arkitecture diagram.ark

# Validate without generating
arkitecture diagram.ark --validate-only

# Override fonts; verbose output
arkitecture diagram.ark --font-size 16 --font-family Helvetica --verbose
```

Exit codes: `0` success · `1` validation/parse errors · `2` filesystem errors.

## Library usage (Go)

The CLI is a thin wrapper over the library — use the same API directly:

```go
import "github.com/kurrik/arkitecture"

res := arkitecture.ToSVG(dsl, nil)
if !res.Success {
    for _, e := range res.Errors {
        log.Printf("%s (line %d, col %d): %s", e.Type, e.Line, e.Column, e.Message)
    }
    return
}
fmt.Println(res.SVG)
```

Options and the individual stages are available too:

```go
res := arkitecture.ToSVG(dsl, &arkitecture.Options{
    ValidateOnly: true,
    FontSize:     14,
    FontFamily:   "Helvetica",
})

parsed := arkitecture.Parse(dsl)               // tokenize + parse -> AST
errs := arkitecture.Validate(parsed.Document)
svg, errs := arkitecture.GenerateSVG(parsed.Document, nil)
```

Every diagnostic is structured data, collected rather than thrown:

```go
type Error struct {
    Line    int
    Column  int
    Message string
    Type    ErrorType // "syntax" | "reference" | "constraint"
}
```

## DSL features

A diagram is authored in **two layers** kept deliberately separate — *semantics*
(what the components are) and *layout* (where and how they're drawn) — so
presentation can be retuned, reused, or swapped without touching structure. There
is no CSS-style cascade: layout targets nodes by exact path, and every position
traces to a local rule. See [examples/annotated.ark](examples/annotated.ark) for a
fully commented reference, or the
[live examples](https://kurrik.github.io/arkitecture/examples.html).

### Semantic layer — nodes

A node body holds only *what a component is*: an `id`, an optional `label`, an
optional `kind`, the anchor **names** it exposes, and nested children. Nesting
means "is part of".

```
api {
  label: "API Gateway"
  anchors: [south]          # anchor NAMES; their positions live in @layout

  auth    { label: "Authentication" }
  routing { label: "Request Routing" }
}
```

A borderless grouping is just a node with `box: none` (set in `@layout`) — the
replacement for the old `group` keyword. It keeps an id, so it still contributes
a path segment.

### Layout layer — `@layout`

Presentation — `direction`, `size`, `margin`, `box`, and anchor **positions** —
lives in `@layout` blocks that target nodes by **exact dotted path**. A block can
sit inline in a node body or stand alone as a sheet:

```
@layout {
  api { direction: vertical; anchor south: [0.5, 1.0] }

  api.auth { size: 0.5 }    # half the orthogonal dimension the parent would give
}
```

A bare `margin: N` at a sheet root sets the **document default margin** — the
fallback spacing for every node that sets none, replacing the built-in 8. It's
the one knob for spacing a whole diagram out (a node still overrides it directly):

```
@layout {
  margin: 16          # space every node out; not a cascade — just a new default
}
```

### Reuse — `@block` / `@use` / `kind`

Bundle shared layout into a named `@block` and pull it in with `@use`. A node's
`kind` implicitly `@use`s the block of the same name as a baseline. Imports are
explicit and overridable — a direct property always wins, with no cascade:

```
@layout {
  @block service { size: 0.75 }

  services.userService  { @use service }
  services.orderService { @use service; size: 0.5 }   # direct wins over the import
}
```

A small set of kinds is built in (`invisible` → `box: none`); any kind can be
(re)declared with `@block <name> { … }`. An explicit `@use` of an undefined block
is an error, but an unknown `kind` is a harmless semantic tag.

### Regrouping — `@group`

Inside a node's `@layout` block, list its children to reorder them and wrap a run
of them in an anonymous `@group` for purely visual nesting (a layout-layer `<div>`).
A group is invisible and has no id — a child inside one keeps its real path:

```
@layout {
  services {
    direction: horizontal
    @group { direction: vertical; userService; orderService }   # stacked, as one unit
    payments
  }
}
```

Once you arrange a node's children, reference each exactly once, and a group may
contain only that node's own children — so the layout stays a faithful regrouping
of the structure.

### Arrows

Connect nodes with `-->`, optionally naming an anchor with `#`. A bare endpoint
auto-routes to the cardinal edge (N/E/S/W) facing the other node; naming an anchor
pins a fixed point.

```
api --> database                            # auto-routed, edge to edge
api#south --> services#north                # explicit anchors
services.userService#db --> database#north  # nested paths
```

### Properties

Semantic (in the node body):

- **`label`** — display text for the node
- **`kind`** — semantic classification; imports the layout block of the same name
- **`anchors`** — declared anchor names, e.g. `[db, north]`

Layout (in `@layout`):

- **`direction`** — `vertical` (default) or `horizontal`
- **`size`** — override (0.0–1.0) for the orthogonal dimension
- **`margin`** — space around the node's border box (default 8; `0` packs flush)
- **`box`** — `default` (bordered) or `none` (borderless grouping)
- **`label`** — `top` (default) or `bottom`: which end of a parent reserves the
  strip for its label, so the label never overlaps the children (bordered and
  `box: none` parents alike)
- **`anchor <name>: [x, y]`** — position a declared anchor
- **`@use <block>`** — import a named `@block`

Document-wide (at an `@layout` sheet root, not in a selector):

- **`margin`** — the default margin for every node that sets none (overrides the
  built-in 8)

### Coordinate system

Anchors use relative coordinates within the node bounding box:

- `[0.0, 0.0]` — top-left corner
- `[0.5, 0.5]` — center (the implicit `center` anchor on every node)
- `[1.0, 1.0]` — bottom-right corner
- `[0.5, 0.0]` — top edge, horizontally centered

## Contributing

1. Branch off `main` (`feature/…`, `fix/…`, `chore/…`).
2. Add tests with each behavioural change.
3. Run `gofmt -l .`, `go vet ./...`, and `go test ./...` before opening a PR.

See [CLAUDE.md](CLAUDE.md) for the full working agreement.

## License

MIT
