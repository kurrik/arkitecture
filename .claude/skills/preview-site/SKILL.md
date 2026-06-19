---
name: preview-site
description: Preview the Arkitecture documentation site (the static site in site/) as screenshots. Use whenever the user asks to preview, see, view, or screenshot the docs site, or to confirm a change under site/ renders correctly. Builds the site, captures a full-page screenshot of every page, and surfaces the images to the user.
allowed-tools: Bash(./scripts/preview-site.sh:*)
---

# Preview the docs site

Claude Code on the web has no port forwarding, so the running dev server can't be
opened in the user's browser. Preview the site as screenshots instead.

1. Render the pages:

   ```bash
   ./scripts/preview-site.sh
   ```

   Add `--build` after changing an example's `.ark` source or the `wasm/` Go code,
   so the example SVGs / WASM are regenerated first. The script builds the site if
   the artifacts are missing, serves it with the correct `application/wasm` MIME
   type, and writes one `<page>.png` per `site/*.html` into `.preview/`. The first
   run installs a headless browser (a few extra seconds); later runs reuse it.

2. Send every PNG it produced (`.preview/*.png`) to the user with the `SendUserFile`
   tool — in a single call, with a short caption naming the pages. These
   screenshots are the deliverable: the user cannot reach the container's
   localhost, so don't just report that a server is running.

For interactive local browsing instead (e.g. after teleporting the session to a
local terminal where localhost is reachable), point the user at
`./scripts/dev-site.sh`, which serves the site at http://localhost:8000/.
