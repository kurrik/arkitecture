#!/usr/bin/env bash
# Generate the publishable site.
#
# Three generation steps run before the static files in site/ are published:
#   1. Re-render every example diagram from its .ark source via the CLI, so the
#      committed SVGs (and the no-JS fallback they provide) always match the
#      current library.
#   2. Inject each example's .ark source into its <code data-ark="…"> block in
#      site/*.html, so the shown source has one canonical home (the .ark file)
#      and can never drift from the diagram it renders.
#   3. Build the WebAssembly module the Examples page loads for live, in-browser
#      editing, plus Go's wasm_exec.js support runtime.
#
# CI (.github/workflows/pages.yml) runs this same script, so a publish always
# reflects the current library. Run it locally from the repository root to
# preview the site the way it ships:
#
#   ./scripts/build-site.sh
#
# Outputs in site/: example *.svg + injected *.html (committed, refreshed here)
# and arkitecture.wasm + wasm_exec.js (build artifacts, git-ignored).
set -euo pipefail

cd "$(dirname "$0")/.."

echo "==> Rendering example diagrams from .ark sources"
for ark in site/examples/*.ark; do
  go run ./cmd/arkitecture "$ark" "${ark%.ark}.svg"
done

echo "==> Injecting example sources into the HTML (single canonical source)"
go run ./internal/sitegen site

echo "==> Building WebAssembly module (site/arkitecture.wasm)"
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o site/arkitecture.wasm ./wasm

echo "==> Copying Go wasm_exec.js support runtime"
goroot="$(go env GOROOT)"
if [ -f "$goroot/lib/wasm/wasm_exec.js" ]; then
  cp "$goroot/lib/wasm/wasm_exec.js" site/wasm_exec.js   # Go >= 1.24
elif [ -f "$goroot/misc/wasm/wasm_exec.js" ]; then
  cp "$goroot/misc/wasm/wasm_exec.js" site/wasm_exec.js  # Go <= 1.23
else
  echo "error: wasm_exec.js not found under $goroot" >&2
  exit 1
fi

echo "==> Site ready in site/"
