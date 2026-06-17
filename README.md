# Arkitecture

[![CI](https://github.com/kurrik/arkitecture/actions/workflows/ci.yml/badge.svg)](https://github.com/kurrik/arkitecture/actions/workflows/ci.yml)

A domain-specific language (DSL) for generating SVG architecture diagrams with precise, manual positioning control. Arkitecture is built for high-level diagrams — Domain-Driven Design boundaries, bounded-context relationships, system overviews — where you want the layout to look exactly the way you arranged it.

Unlike tools that use automatic layout algorithms, Arkitecture gives you fine-grained control over element positioning and sizing: you describe the structure and the layout, and the tool only measures text and packs boxes — deterministically.

> 🚧 **Rewrite in progress.** Arkitecture is being ported from TypeScript to Go
> (a single portable binary, plus a WASM library for JS/TS interop). The
> tokenizer and parser are ported; the validator and SVG generator are still
> being ported, so `--validate-only` works today but SVG output reports "not yet
> ported". See [docs/roadmap.md](docs/roadmap.md).

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

The Arkitecture DSL has a clean, intuitive syntax for describing architecture diagrams. See [examples/annotated.ark](examples/annotated.ark) for a fully commented reference.

### Container nodes

Container nodes are the primary building blocks, with IDs, labels, and layout properties:

```
# Basic node with a label
api {
  label: "API Gateway"
  direction: "vertical"
}

# Node with a size override and custom anchors
userService {
  label: "User Service"
  size: 0.75
  anchors: {
    db: [0.5, 1.0],
    api: [0.5, 0.0]
  }
}
```

### Groups

Groups provide layout organization without any visual representation:

```
services {
  label: "Microservices"
  direction: "horizontal"

  group {
    direction: "vertical"

    userService {
      label: "User Service"
    }

    orderService {
      label: "Order Service"
    }
  }
}
```

### Arrows

Connect nodes with arrow syntax and optional anchor references:

```
# Simple arrow between nodes
api --> database

# Arrow with anchor points
api#south --> services#north

# Nested node references
services.userService#db --> database#north
```

### Properties

- **`label`**: display text for the node
- **`direction`**: layout direction (`"vertical"` or `"horizontal"`)
- **`size`**: size override (0.0–1.0) for the orthogonal dimension
- **`anchors`**: custom anchor points as `{ anchorId: [x, y] }`

### Coordinate system

Anchors use relative coordinates within the node bounding box:

- `[0.0, 0.0]` — top-left corner
- `[0.5, 0.5]` — center (the default anchor on every node)
- `[1.0, 1.0]` — bottom-right corner
- `[0.5, 0.0]` — top edge, horizontally centered

## Contributing

1. Branch off `main` (`feature/…`, `fix/…`, `chore/…`).
2. Add tests with each behavioural change.
3. Run `gofmt -l .`, `go vet ./...`, and `go test ./...` before opening a PR.

See [CLAUDE.md](CLAUDE.md) for the full working agreement.

## License

MIT
