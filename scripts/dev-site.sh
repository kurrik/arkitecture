#!/usr/bin/env bash
# Serve the documentation site locally for development.
#
# The site in site/ is static HTML/CSS/JS, but the Examples page progressively
# enhances itself by fetching the WebAssembly build (site/arkitecture.wasm) — so
# it must be served over HTTP with a correct application/wasm MIME type, not
# opened as a file:// URL. This script ensures the generated artifacts exist
# (example SVGs + WASM + wasm_exec.js, via build-site.sh) and then serves site/.
#
#   ./scripts/dev-site.sh                # build if needed, serve on :8000
#   ./scripts/dev-site.sh --port 9000    # pick a port (or PORT=9000 ...)
#   ./scripts/dev-site.sh --build        # force a rebuild first (after editing
#                                        #   an .ark source or the wasm/ Go code)
#
# Editing the HTML/CSS/JS needs no rebuild — just refresh the browser. Re-run
# with --build after changing an example's .ark or the library the WASM wraps.
set -euo pipefail

cd "$(dirname "$0")/.."

port="${PORT:-8000}"
force_build=0

while [ $# -gt 0 ]; do
  case "$1" in
    -p|--port) port="$2"; shift 2 ;;
    --port=*)  port="${1#*=}"; shift ;;
    -b|--build) force_build=1; shift ;;
    -h|--help)
      sed -n '2,16p' "$0" | sed 's/^# \{0,1\}//'
      exit 0 ;;
    *) echo "error: unknown argument '$1' (try --help)" >&2; exit 1 ;;
  esac
done

# Build when asked, or when the generated artifacts the Examples page needs are
# missing. Otherwise skip the (slow) WASM build so HTML/CSS/JS iteration is fast.
if [ "$force_build" -eq 1 ] || [ ! -f site/arkitecture.wasm ] || [ ! -f site/wasm_exec.js ]; then
  ./scripts/build-site.sh
else
  echo "==> Using existing build artifacts (pass --build to regenerate)"
fi

echo "==> Serving site/ at http://localhost:${port}/  (Ctrl-C to stop)"

# serve-site.py serves site/ with a correct application/wasm MIME type, which
# the playground's WebAssembly.instantiateStreaming needs.
exec python3 scripts/serve-site.py "$port" site
