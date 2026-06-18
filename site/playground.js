/* Progressive enhancement for the Examples page.
 *
 * Loads the Arkitecture WebAssembly build (the exact Go library the CLI uses)
 * and turns each example's read-only .ark source into an editable field that
 * re-renders live in the browser as you type. A Reset control appears once an
 * example diverges from its original source.
 *
 * This is purely additive: with JavaScript or WebAssembly unavailable the page
 * is unchanged — the static, pre-rendered SVGs stay in place and the source is
 * shown as before. No build step beyond scripts/build-site.sh, which emits the
 * arkitecture.wasm and wasm_exec.js this file expects alongside the page.
 */
(function () {
  "use strict";

  var examples = Array.prototype.slice.call(
    document.querySelectorAll(".example")
  );
  // Go is defined by wasm_exec.js; bail out (leaving the static page) if either
  // the support runtime or the examples aren't present.
  if (examples.length === 0 || typeof Go === "undefined") return;

  instantiate()
    .then(function () {
      examples.forEach(enhance);
      announce();
    })
    .catch(function (err) {
      // Leave the static fallback untouched; just note why live edit is off.
      console.error("Arkitecture playground failed to load:", err);
    });

  // Fetch and start the WASM module. main() registers the global
  // arkitectureToSVG and then blocks on select{}, so go.run() returns with the
  // function already installed (we deliberately don't await its exit).
  function instantiate() {
    var go = new Go();
    var load = WebAssembly.instantiateStreaming
      ? WebAssembly.instantiateStreaming(
          fetch("arkitecture.wasm"),
          go.importObject
        )
      : fetch("arkitecture.wasm")
          .then(function (resp) {
            return resp.arrayBuffer();
          })
          .then(function (bytes) {
            return WebAssembly.instantiate(bytes, go.importObject);
          });
    return load.then(function (result) {
      go.run(result.instance);
    });
  }

  function enhance(example) {
    var sourcePanel = example.querySelector(".panel.source");
    var renderArea = example.querySelector(".render-area");
    var pre = sourcePanel && sourcePanel.querySelector("pre");
    if (!sourcePanel || !renderArea || !pre) return;

    var original = pre.textContent;

    // Swap the read-only <pre> for an editable textarea seeded from it.
    var editor = document.createElement("textarea");
    editor.className = "source-editor";
    editor.spellcheck = false;
    editor.setAttribute("aria-label", "Editable diagram source");
    editor.value = original;
    editor.rows = original.split("\n").length + 1;
    pre.replaceWith(editor);

    // Reset lives in the panel label, hidden until the source is modified.
    var reset = document.createElement("button");
    reset.type = "button";
    reset.className = "reset-btn";
    reset.textContent = "Reset";
    reset.hidden = true;
    var label = sourcePanel.querySelector(".panel-label");
    if (label) label.appendChild(reset);

    // Error readout shown beneath the diagram when the source doesn't compile.
    var errorBox = document.createElement("pre");
    errorBox.className = "render-errors";
    errorBox.hidden = true;
    renderArea.after(errorBox);

    function render() {
      var dsl = editor.value;
      reset.hidden = dsl === original;
      var result = globalThis.arkitectureToSVG(dsl);
      if (result && result.success) {
        renderArea.innerHTML = result.svg;
        errorBox.hidden = true;
      } else {
        // Keep the last good diagram on screen; surface the errors below it.
        var errs = (result && result.errors) || [];
        errorBox.textContent = errs.length
          ? errs.map(formatError).join("\n")
          : "Could not render diagram.";
        errorBox.hidden = false;
      }
    }

    editor.addEventListener("input", debounce(render, 150));
    reset.addEventListener("click", function () {
      editor.value = original;
      render();
      editor.focus();
    });

    // Render once through WASM so the live pipeline, not the static SVG, owns
    // the output from here on.
    render();
  }

  // Reveal a one-line hint only when live editing is actually available.
  function announce() {
    var lede = document.querySelector("main .lede");
    if (!lede) return;
    var hint = document.createElement("p");
    hint.className = "playground-hint";
    hint.textContent =
      "Live in your browser — edit any example below and it re-renders as you type.";
    lede.after(hint);
  }

  function formatError(e) {
    var loc = "";
    if (e.line) loc = " (line " + e.line + (e.column ? ":" + e.column : "") + ")";
    return (e.message || "Error") + loc;
  }

  function debounce(fn, ms) {
    var t;
    return function () {
      clearTimeout(t);
      t = setTimeout(fn, ms);
    };
  }
})();
