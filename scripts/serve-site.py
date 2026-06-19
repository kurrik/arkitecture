#!/usr/bin/env python3
"""Static file server for the docs site, with a correct .wasm MIME type.

Shared by scripts/dev-site.sh (interactive) and scripts/preview-site.sh
(screenshots). Python's http.server doesn't map .wasm by default, but the
Examples page's WebAssembly.instantiateStreaming needs application/wasm — so we
register it here in one place.

Usage: serve-site.py [PORT] [DIRECTORY]   (defaults: 8000, "site")
Serves until interrupted.
"""
import functools
import http.server
import socketserver
import sys

port = int(sys.argv[1]) if len(sys.argv) > 1 else 8000
directory = sys.argv[2] if len(sys.argv) > 2 else "site"

handler = http.server.SimpleHTTPRequestHandler
handler.extensions_map[".wasm"] = "application/wasm"

socketserver.TCPServer.allow_reuse_address = True
with socketserver.TCPServer(("", port), functools.partial(handler, directory=directory)) as httpd:
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        pass
