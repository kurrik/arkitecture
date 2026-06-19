#!/usr/bin/env bash
# Preview the documentation site as images.
#
# Claude Code on the web has no port forwarding, so a running dev server can't
# be opened in a browser from a cloud session. This renders the site to PNGs
# instead: it builds the site, serves it locally just long enough to capture
# each page with a headless browser, and writes the images to .preview/ (which
# can then be surfaced to the user).
#
# Usage:
#   ./scripts/preview-site.sh              # build if needed, shoot every site/*.html
#   ./scripts/preview-site.sh --build      # force a rebuild first
#   ./scripts/preview-site.sh --port 9000  # serve on a specific port while shooting
#   ./scripts/preview-site.sh --out shots  # write PNGs to a different directory
#
# Output: one <page>.png per site/*.html in the output dir (default .preview/).
# The first run installs a headless browser (puppeteer) under .preview-tools/;
# both directories are git-ignored.
set -euo pipefail

cd "$(dirname "$0")/.."

port="${PORT:-8123}"
out=".preview"
tools=".preview-tools"
force_build=0

usage() {
  sed -n '2,18p' "$0" | sed 's/^# \{0,1\}//'
}

while [ $# -gt 0 ]; do
  case "$1" in
    -p|--port) port="$2"; shift 2 ;;
    --port=*)  port="${1#*=}"; shift ;;
    -o|--out)  out="$2"; shift 2 ;;
    --out=*)   out="${1#*=}"; shift ;;
    -b|--build) force_build=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "error: unknown argument '$1' (try --help)" >&2; exit 1 ;;
  esac
done

command -v node    >/dev/null || { echo "error: node is required for screenshots" >&2; exit 1; }
command -v python3 >/dev/null || { echo "error: python3 is required to serve the site" >&2; exit 1; }

# 1. Build the site if asked, or if the generated artifacts are missing.
if [ "$force_build" -eq 1 ] || [ ! -f site/arkitecture.wasm ] || [ ! -f site/wasm_exec.js ]; then
  ./scripts/build-site.sh
fi

# 2. The pages to capture: every top-level site/*.html.
pages=()
for f in site/*.html; do
  [ -e "$f" ] || continue
  pages+=("$(basename "$f")")
done
[ "${#pages[@]}" -gt 0 ] || { echo "error: no site/*.html pages found" >&2; exit 1; }

# 3. Ensure a headless browser (puppeteer + its Chromium). Keep the package and
#    the browser cache under .preview-tools/ so nothing leaks into the repo.
export PUPPETEER_CACHE_DIR="$PWD/$tools/.cache"
if [ ! -d "$tools/node_modules/puppeteer" ]; then
  echo "==> Installing headless browser (first run only)…"
  command -v npm >/dev/null || { echo "error: npm is required to install the headless browser" >&2; exit 1; }
  mkdir -p "$tools"
  # A minimal manifest with a valid package name (npm rejects the dotted dir name).
  printf '{\n  "name": "arkitecture-preview-tools",\n  "private": true\n}\n' > "$tools/package.json"
  if ! ( cd "$tools" && npm install puppeteer ) >"$tools/install.log" 2>&1; then
    echo "error: failed to install the headless browser; last lines of $tools/install.log:" >&2
    tail -20 "$tools/install.log" >&2
    exit 1
  fi
fi

mkdir -p "$out"

# 4. Serve site/ in the background; tear it down on any exit.
python3 scripts/serve-site.py "$port" site >/dev/null 2>&1 &
server=$!
trap 'kill "$server" 2>/dev/null || true' EXIT

# Wait for the server to accept connections (curl's own retry, no sleep loop).
curl --retry-connrefused --retry 50 --retry-delay 1 -sf -o /dev/null \
  "http://localhost:${port}/${pages[0]}" \
  || { echo "error: server did not come up on port ${port}" >&2; exit 1; }

echo "==> Capturing ${#pages[@]} page(s) to ${out}/"

# 5. Screenshot each page. puppeteer resolves from the .preview-tools install
#    via NODE_PATH; networkidle settles the Examples page's live WASM render.
PREVIEW_BASE="http://localhost:${port}" PREVIEW_OUT="$out" PREVIEW_PAGES="${pages[*]}" \
NODE_PATH="$PWD/$tools/node_modules" \
node <<'JS'
const puppeteer = require("puppeteer");
const base = process.env.PREVIEW_BASE;
const out = process.env.PREVIEW_OUT;
const pages = process.env.PREVIEW_PAGES.split(/\s+/).filter(Boolean);
(async () => {
  const browser = await puppeteer.launch({ args: ["--no-sandbox", "--disable-setuid-sandbox"] });
  const page = await browser.newPage();
  await page.setViewport({ width: 1100, height: 900, deviceScaleFactor: 2 });
  for (const name of pages) {
    await page.goto(`${base}/${name}`, { waitUntil: "networkidle0", timeout: 60000 });
    await new Promise((r) => setTimeout(r, 600)); // let live WASM renders settle
    const file = `${out}/${name.replace(/\.html$/, "")}.png`;
    await page.screenshot({ path: file, fullPage: true });
    console.log("  " + file);
  }
  await browser.close();
})().catch((e) => { console.error(e); process.exit(1); });
JS

echo "==> Done. PNGs are in ${out}/"
