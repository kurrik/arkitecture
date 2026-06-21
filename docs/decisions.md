# Decisions

Append-only log of non-obvious technical and design choices, newest first. One
entry per decision; never rewrite history — supersede an old entry with a new one.

Format:

```
## YYYY-MM-DD — Short title
**Choice:** What we picked.
**Why:** The reasoning, including alternatives considered.
**Implications:** What this commits us to or rules out.
```

What belongs here: anything a future reader would otherwise have to re-derive or
re-litigate — a language choice, a persistence model, a concurrency policy, a
deliberate non-feature, a rejected refactor. *Routine* decisions don't.

---

## 2026-06-21 — Auto edge routing: channel widening for cross-axis rails
**Choice:** Extend channel widening (the 2026-06-20 first cut, which handled
main-axis gaps) to **cross-axis perimeter rails** — the two container sides
*parallel* to its main axis, where an arrow travels *along* the perimeter (e.g.
the over-the-obstacle run in `orthogonal-around`/`detour`). A run parallel to a
container's main axis is now attributed to its low/high rail (`railSideAt`, by the
run's cross-axis position relative to the children band), demand populates
`widenDemand.rails`, and the run snaps to the rail centre (`railCenterAt`). The
`railExtra` layout plumbing was already in place from the first cut, so this slice
is attribution + snapping only. Gaps and rails were unified behind a
`channelRef{path, rail, index, base}` so both flow through one path.
**Why:** Reported: the first cut left lines running *along a perimeter* still
inside the margin (the only part of "any line parallel to a node edge acts as a
wall" not yet covered). Rails are the dual of gaps — a gap is a strip crossing the
main axis, a rail a strip along it — so they widen by the same `lanes × margin/2`
rule and reuse the same two-pass/clip-by-container machinery; the only genuinely
new logic is choosing a rail *side* from the run's cross-axis position and finding
the rail's centre (which must account for the label band, since for a *horizontal*
container the band sits on the cross axis). The document **root** has no perimeter,
so a run along it is deliberately not a channel (no canvas padding to widen).
**Implications:** Only `orthogonal-around` rebaselined (its over-the-wall run now
widens the row's top perimeter and centres in it); `orthogonal-arrows`/`-breakout`/
`-widening` are unchanged (their lanes are gaps). The `detour` site example
likewise widens. `findChannel` now returns a `channelRef`; `widen_test.go` covers
gap **and** rail attribution. Deterministic and still single-pass. **Remaining on
the widening track:** **multi-lane** distribution — two arrows sharing one channel
both snap to its centre (the channel widens by the count, but the lanes aren't
spread to distinct offsets); that is the last piece.

## 2026-06-20 — Auto edge routing: channel widening (first cut — main-axis gaps)
**Choice:** Make an orthogonal arrow that runs *along* a channel reserve its own
**lane** there, so the channel widens and the boxes spread, instead of the line
sitting inside a node's margin (the design's channel-vs-margin rule). Realised as
a **two-pass** layout rather than the full topological slot router:
1. Lay out un-widened and route every arrow (the existing geometric router).
2. **Attribute** each routed run to the container gap it runs along
   (`findChannel`): descend by the run's midpoint into the deepest box, and take
   the gap in the container that holds the midpoint in free space — clipping by
   container is what stops a single run from smearing across channels, so lane
   counts come out **exact**. A run *along* a container's main axis (a cross-axis
   perimeter rail) is deferred.
3. **Widen**: each gap reserves `lanes × margin/2` extra (`laneSpacing` = half a
   margin, the author's pick), threaded into `calcDimensions`/`positionNodes`
   (`gapExtra`/`railExtra`) and the root gap loop. Default zero ⇒ a document with
   no orthogonal lanes is byte-identical.
4. Lay out **again** with the widened channels, and **snap** each interior run to
   its gap centre (`snapToLanes`) so the line sits in its lane (tip tails are left
   anchored to the box).
**Why:** The author reported that routed lines travel *inside* the margin gap
(e.g. the contexts arrow ran 4px under Orders, in its 8px padding), contradicting
"channels reserve their own width and push boxes apart; margins stay breathing
room." Widening is the design's answer. The design specified a from-scratch
**topological slot router** to keep lane counts pure-topology and avoid a
position↔width fixpoint — but the fixpoint only bites *channel widths*, and
**clipping runs to containers** recovers exact per-gap lane counts from the
*geometric* router at far less risk, so this ships the channel-vs-margin outcome
without the full rewrite. Two passes are deterministic (no map order reaches the
SVG) and single-shot (no iteration): widening preserves the arrangement, so the
re-routed paths are topologically the same, only spread. `margin/2` per lane is
the author's "half a margin per lane" (a tighter look than a full margin each
side).
**Implications:** `computeLayout` gained a `*widenDemand` (nil = pass 1);
`GenerateSVG` runs the two passes in orthogonal mode; `generator/widen.go` holds
attribution + demand + snapping. `orthogonal-arrows` and `orthogonal-breakout`
rebaselined (gaps widened, runs centred); an `orthogonal-widening` golden locks
the showcase; `orthogonal-anchors`/`orthogonal-around` are unchanged (their runs
are tip tails or rails). **Deferred (next slices):** **cross-axis rails** — a run
along a container's main-axis perimeter (e.g. `orthogonal-around` over the wall)
isn't widened yet, so it still travels in the perimeter margin; **multi-lane
distribution** — two arrows sharing one gap both snap to its centre and overlap
(the gap widens by the count, but the lanes aren't spread to distinct offsets);
and the **midpoint** attribution is an approximation of true per-container
clipping (fine when a run's midpoint lands in the channel it mostly occupies).

## 2026-06-20 — Auto edge routing: edge-normal exits for pinned anchors
**Choice:** An explicit anchor pinned to one of its box's **own edges** now leaves
(or is entered) **perpendicular to that edge**, instead of running along the edge
toward the side facing the other node. Resolution splits explicit anchors three
ways (`resolveOrthoEndpoint`):
- **Edge anchor not on the facing side** (e.g. a bottom-centre anchor with the
  target to the east): the exit side is the anchor's own edge normal, and the
  routing start is a stub one inset outside the **leaf** edge — the line departs
  perpendicular into the leaf's channel and the channel router follows corridors
  out. The endpoint is flagged `edgeExit`, and `arrowPath` adds that leaf box to
  the arrow's obstacle set so the route **detours around its own box** instead of
  turning straight back through it.
- **Edge anchor on the facing side**: met one inset **past** the breakout-box
  border (out in the channel), so the router approaches it along the facing normal
  rather than sliding up the border when the route arrives askew. At the
  breakout-box border this is the same crossing the break-out slice already made,
  so `orthogonal-breakout` is byte-identical.
- **Interior / corner anchor, or a bare reference**: unchanged — met at the
  breakout-box border on the facing side (entering the node for an interior
  anchor).
**Why:** Reported on the `contexts` diagram: `ordering.orders#db` is pinned to
Orders' **bottom** edge (`[0.5, 1.0]`), but the arrow ran east *along* that bottom
edge and up the container's outside, ignoring that the author put the anchor on
the bottom. The author's expectation — leave perpendicular to the pinned edge,
drop into the channel below Orders, then route around — is the "edge-normal exits"
slice. Choosing the exit side from **the edge the anchor sits on** (not the facing
side) is the whole change; the existing channel router then supplies the
corridor-following and break-out for free. Walling off the leaf box is what stops
the perpendicular exit from immediately turning back through the source (a clear,
unobstructed bottom-anchor-to-eastern-target otherwise elbowed straight up through
its own box — caught by a render before this shipped). Keeping the **facing-side**
anchor at the breakout-box border (plus an inset so the approach is normal) is
what holds `orthogonal-breakout` stable while still fixing the *target*-side
approach (the `contexts` arrow now enters `inventory#api`, a west-edge anchor,
horizontally from the west rather than running up its border).
**Implications:** Only `orthogonal-anchors` moved among goldens — `source#out`
(a bottom anchor) now drops straight down before turning into Service, which the
fixture comment and a regenerated golden capture; `orthogonal-breakout`,
`orthogonal-arrows`, and `orthogonal-around` are byte-identical. `endpoint` gained
`edgeExit`; `anchorEdge` classifies an anchor's single edge (corner/interior →
none); `TestAnchorEdge` and `TestArrowPathEdgeAnchorExitsPerpendicular` lock the
classification and the down-and-around behaviour. **Residual:** a **bare-reference
target** can still be approached askew (it gets no facing-side stub — adding one
would move the bare-ref goldens), so a bottom-anchor→bare arrow enters the target
from whichever side the detour arrives; pinning the target to an edge gives the
clean normal approach. This supersedes the positioned-anchors ADR's "an edge
anchor not on the facing side routes along its own border" aesthetic note — that
case is now the perpendicular exit. Still deferred: channel **widening**, and a
fully topological (pre-layout) route assignment.

## 2026-06-20 — Auto edge routing: the channel-graph router (route around obstacles)
**Choice:** When an arrow's direct elbow would cut through a box between its
endpoints, route the arrow *around* the obstacles on a **channel graph** with
**few-bend A\***, instead of falling back to the straight line. The router
(`generator/channel.go`, `routeAround`) builds, per arrow, a sparse grid whose
travel lanes sit one **inset** (`defMargin/2`) outside every obstacle edge — so a
line runs in the gap *beside* a box, and in the common uniform-margin layout the
two lanes flanking a channel coincide on its centreline. Grid vertices are lane
crossings (plus the lines through the two endpoints); a grid edge joins adjacent
crossings whose connecting segment grazes no obstacle interior (reusing
`segIntersectsRect`). A\* search state is `(vertex, incoming-direction)` so a
**bend penalty** (12) is charged on each turn; cost is segment length + bend
penalty, the heuristic is Manhattan distance (admissible), and the frontier has a
**total order** (f, then g, then a state id encoding vertex-then-direction) so
ties resolve to the smaller-coordinate (north/west) route — fully deterministic.
`arrowPath` now tries, in order: the clear-case elbow, then `routeAround`, then
the straight fallback. The elbow stays the fast path, so a clear arrow is
byte-identical to before.
**Why:** This is the heart of the sized-channel design (the 2026-06-20 design
ADR): the substrate is the **gaps between boxes**, not a pixel grid. Realising
those gaps as obstacle-edge-offset lanes makes the grid sparse (a handful of
lines per arrow — "fine at this tool's scale, dozens of boxes") and puts lines on
channel centrelines in the uniform case for free. **Routing on the
already-computed layout** (rather than before sizing) is correct *for this slice*
because the deferred piece — channel **widening** — is the only thing that would
move boxes; with no widening yet, no box moves, so there is no routing↔layout
feedback loop to resolve, and the design's "route before pixels" requirement only
bites once widening lands. Keeping the **elbow as a fast path** is what holds
every existing orthogonal golden byte-for-byte stable: the router fires *only*
when the elbow is blocked, i.e. exactly the arrows that previously fell back to a
straight line through a box. The few-bend A\* with a deterministic total order
satisfies the design's "A\* tie-breaking must be totally deterministic or the
goldens churn."
**Implications:** Orthogonal arrows blocked by an intervening box now detour
cleanly (an `orthogonal-around` golden — a `Left`/`Wall`/`Right` row — locks the
up-and-over route through the row's perimeter padding; `channel_test.go` covers
the detour, the no-path→nil case, determinism, the fewer-bends preference, the
grid construction, and the end-to-end `arrowPath`). No existing golden moved.
**Deferred, unchanged from before:** (1) **Edge-normal exits & channel-following**
— A\* connects the breakout-box border points directly, so a route may approach an
endpoint from a side other than its facing edge (the `orthogonal-around`
arrowhead enters the target's west edge from the north, since the 2-bend route
beats the 3-bend edge-normal one under the bend penalty); forcing the first/last
segment along the side normal is its own slice. (2) **Channel widening** — the
lane *demand* per channel still isn't fed back into `calcDimensions`/
`positionNodes` to push boxes apart; this slice assigns routes but reserves no
width for them, so two arrows sharing a lane overlap. (3) A per-arrow `route:`
override. The router is geometric (post-layout) rather than a topological
slot-assignment ahead of sizing; widening will be the point at which route
assignment must hoist before layout, deriving lane counts as topology — this ADR
records that the current realisation routes on geometry and why that is sound
until then.

## 2026-06-20 — Auto edge routing: break out to the container channel
**Choice:** When an arrow endpoint is nested inside a container, route the
orthogonal gap-crossing between the endpoints' **breakout boxes** — the outermost
container that holds one endpoint but not the other — rather than between the leaf
boxes. The exit side and border point are taken on the breakout box (facing the
other's breakout box), with a tail connecting the tip out to that border;
`breakoutBox(self, other)` is the self-side child of the two paths' lowest common
container (the leaf itself when both share a parent).
**Why:** Reported: a nested anchor's line ran *along the container's border*
instead of through the gap (the Orders→Inventory arrow's vertical fell exactly on
Ordering's right edge). The elbow placed its bend at the midpoint of the two
**leaf** edges, and for a nested source that midpoint coincided with the container
border. Crossing between the **container** edges instead puts the bend at the
midpoint of the real gap — the channel between the containers. It reduces to the
prior behaviour for same-parent arrows (breakout box = leaf), so **no existing
golden moved**, and M2 straight mode is untouched: the tip still uses the
leaf-to-leaf cardinal selection, and only the orthogonal edge/side consult the
breakout box.
**Implications:** This is the channel-placement half of the designed "break-out"
slice — an arrow between different containers now crosses the inter-container
channel. Routing *around* obstacle boxes (the channel graph + few-bend A*) and
channel widening still remain; a blocked route still falls back to the straight
line. A new `orthogonal-breakout` golden locks the in-channel run, and
`TestBreakoutBox` covers the lowest-common-container child computation. Residual
aesthetic: an anchor on a box's far edge still traces that *leaf's* edge to exit
toward the target — distinct from the container-border issue fixed here, and
governed by where the author places the anchor.

## 2026-06-20 — Auto edge routing: orthogonal routing for positioned anchors
**Choice:** Extend orthogonal mode to route arrows that attach to an explicit
anchor, **superseding slice 1's "explicit-anchor arrows stay straight."** Every
positioned anchor is handled by one rule: the **exit side faces the other node's
box centre** (the same choice a bare reference makes), the orthogonal path meets
the box **border on that side aligned with the anchor**, and a final **tail
segment** runs between that border point and the anchor. The tail is zero-length
when the anchor already lies on the facing side (an edge anchor), and **crosses
the interior — entering the node — for an interior anchor** (e.g. `#center`). A
bare reference is the special case where the anchor *is* the border point (tip ==
edge), so the anchorless output is unchanged. The obstacle test and the
straight-line fallback are unchanged.
**Why:** The author asked for `route: orthogonal` to work for *all* positioned
anchors, not just edge ones, "entering the node for non-edge positions." Choosing
the exit side by facing the other box (rather than by which border the anchor sits
on) is what makes edge and interior anchors fall out of a single rule — an edge
anchor's tail collapses to nothing, an interior anchor's tail is the entry segment
— and makes the bare-reference case the same code with tip == edge. It reuses M2's
cardinal side selection and leaves the `elbow` primitive and the obstacle/fallback
machinery untouched, so the change is localized to endpoint resolution and path
assembly.
**Implications:** `endpoint` now carries `{tip, edge, side}`; `arrowPath`
assembles tip → border → elbow across the gap → border → tip and simplifies
(collinear tails merge; bare-to-bare reduces exactly to the M2 elbow, so the
`orthogonal-arrows` golden is byte-identical — confirmed). In orthogonal mode an
explicit `#anchor` (including `#center`) now routes orthogonally instead of
drawing straight. A new `orthogonal-anchors` golden locks the edge-anchor and
interior-anchor (enter-the-node) cases. **Aesthetic note:** an edge anchor that is
*not* on the side facing the target routes along its own border to exit toward the
target (a bottom-centre anchor with the target to the east runs along the bottom,
then turns up) — correct and orthogonal, though it traces the box edge; anchor
placement is the author's control. Still deferred on this track: routing *around*
obstacles (the channel graph + A*), channel widening, and break-out.

## 2026-06-20 — Single canonical example source (inject `.ark` into the site HTML)
**Choice:** Generate the docs site's shown example source from the canonical
`site/examples/*.ark` files at build time instead of hand-copying it into the
HTML. Each example's `<pre><code>` block is marked `data-ark="examples/NAME.ark"`;
a new `internal/sitegen` build tool (run by `scripts/build-site.sh`, right after
the SVG render) replaces that block's body with the file's contents — trailing
newline trimmed, only `&`/`<`/`>` escaped to match the page's hand-written style.
It is **idempotent** (no `.ark` change → no rewrite) and the injected HTML is
**committed**, exactly like the rendered example SVGs.
**Why:** The displayed source was a second, hand-maintained copy of each `.ark`,
and it drifted: after #19 set the pipeline example's `@layout { margin: 20 }` in
the `.ark` (and the render), the *shown* source still lacked it, and the
`contexts` example's `.ark` comment was likewise missing from the page. Reported
by the author ("I don't want to keep multiple source documents in sync"). Making
the `.ark` the one source removes both the manual-copy chore and the whole class
of drift. **Build-time injection over a client-side fetch** keeps the no-JS
contract: the committed HTML still carries the source for non-JS viewers and is
what Pages serves. **Committing the injected HTML over filling empty placeholders
at publish** matches how the example SVGs already work (generated, committed,
refreshed at publish) and avoids working-tree churn, since re-running is
idempotent. **A Go tool over a shell/Python one** keeps the publish path Go-only
(no new language in `build-site.sh` or CI) and is unit-tested (`inject` takes a
reader, so it tests without the filesystem).
**Implications:** To add or edit an example, change only its `.ark`; the same
build that refreshes the SVG now also refreshes the shown source (and
`dev-site`/`preview-site`/`pages` all run it). `internal/sitegen` is a
`package main` build helper under `internal/` — built by `go build ./...`, but
not part of the shipped CLI and not installable. The `data-ark` attribute is the
marker, so unmarked `<code>` blocks (the Quick-start shell/Go snippets) are left
alone. There is no separate CI drift-check — consistent with the committed SVGs,
where the publish always corrects — and the drift window is tiny since any local
preview rebuilds. The `panel-label` filename stays literal (cosmetic, low-drift);
generating it too is a possible later tidy-up.

## 2026-06-20 — Auto edge routing: `route` setting + orthogonal emission (slice 1)
**Choice:** Begin implementing the sized-channel design (the ADR below) in
reviewable slices. This first slice lands the surface and the emission path,
deferring the channel-graph router:
- **`route` is a document-level setting**, not a per-node property: a bare
  `route: straight | orthogonal` at an `@layout` sheet root, parsed into
  `Document.Route *ast.RouteMode` exactly the way `margin: N` becomes
  `DefaultMargin` (same `:`-after-name dispatch in `parseDocumentDefault`).
  Default (nil / `straight`) is today's M2 line; `orthogonal` opts the whole
  document in. A per-arrow override stays deferred (design ADR / roadmap).
- **Orthogonal mode emits a `<polyline>` only for genuinely bent paths; a
  two-point path still emits the existing `<line>`.** So straight mode — and any
  orthogonal arrow that stays straight (aligned or adjacent boxes) — is
  byte-for-byte unchanged, and only a real bend introduces the new element. The
  arrowhead `<marker>` is reused unchanged (it orients on the final segment).
- **This slice routes the clear case only.** Both ends keep their M2 cardinal
  point; the two points are joined by a deterministic `elbow()` (one corner for
  perpendicular exit sides, a Z through the gap mid-line for parallel ones,
  collapsing to a straight segment when collinear). The elbow is used **only when
  it is clear of the arrow's obstacles**; otherwise the arrow falls back to the
  straight M2 line — so orthogonal mode never renders a *worse* result than
  straight mode. Routing *around* obstacles (the channel graph + few-bend A*),
  channel widening, and cross-container break-out are the next slices.
- **The arrow-relative obstacle set is path-prefix lineage.** An arrow's
  obstacles are every box whose dotted path does **not** share a root-to-node
  lineage with the source or the target — i.e. excluding each endpoint's
  ancestors, descendants, and itself (`related(a,b)` = equal or one a `.`-prefix
  of the other). This is the design's "obstacles are every box except the
  ancestors of source and target" as cheap string matching, and it is what lets a
  route legitimately cross its *own* containers' borders (the seed of break-out)
  while still treating sibling/foreign boxes as walls.
- **Explicit-anchor arrows stay straight in orthogonal mode** (a pinned anchor has
  no cardinal side to extend orthogonally); orthogonalising them is deferred.
**Why:** Slicing keeps each change reviewable and golden-stable. Putting `route`
at the sheet root mirrors the only other document-wide knob (`margin`), needs no
new keyword, and reuses the existing duplicate/dispatch machinery. Emitting
`<line>` for two-point paths is what makes the slice land with **no existing
golden moved** — orthogonal output diverges from straight output only where a bend
actually exists. The clear-case-with-straight-fallback rule means the feature is
safe to ship incrementally: turning it on can only improve diagonal arrows (into
Z/elbows) and never degrades a blocked one below today's behaviour. Realising the
obstacle rule as prefix matching avoids threading the layout tree into routing and
falls straight out of the dotted paths the boxes are already keyed by.
**Implications:** Endpoint resolution moved from `svg.go` into a new
`generator/route.go` (`arrowPath`/`resolveOrthoEndpoint`/`cardinalEndpoint`),
which `renderArrows` now calls for both modes; the old `resolveEndpoint`/
`cardinalPoint` are gone (straight mode routes through the same code, two-point
result). `elbow()` is a complete connector for *any* two cardinal sides — its
perpendicular branch is not yet reachable from two-box cardinal routing (both ends
pick the same dominant axis, so sides are always parallel → a Z) but is unit-tested
and will be exercised once routes turn through perimeter rails. A new
`orthogonal-arrows` golden locks the Z and the straight cases in. Determinism holds:
elbow geometry is a fixed function of the endpoints, the obstacle test is an
order-independent boolean, and no map iteration order reaches the output. Still to
come on this track: the channel graph + A* (route around obstacles), channel
widening (lanes push boxes apart), and break-out across nesting levels.

## 2026-06-20 — Auto edge routing via sized channels (design)
**Choice:** Add an opt-in **auto-routing mode** that draws arrows as **orthogonal
paths routed around boxes** instead of straight lines, built on three commitments:
- **Channels are first-class, sized layout objects — not the margins.** A
  **channel** is a routing corridor between adjacent blocks that reserves its own
  width and **pushes the boxes apart** to make room for the lines it carries.
  **Margin** stays what it is — aesthetic breathing room that spaces a line nicely
  from the box it runs alongside. So routing does *not* thread lines through the
  margin gap; it inserts a sized lane-bundle, with margin still framing it on
  either side. A channel's width is `margin + lanes × laneSpacing`, where *lanes*
  is the number of arrows running **longitudinally** through that channel (a line
  merely crossing it perpendicularly needs only a point, not a lane).
- **Route on a channel graph derived from the arrangement tree, before pixel
  layout — so there is no layout↔routing feedback loop.** The graph's nodes are
  channel slots (between every pair of adjacent siblings, plus a **perimeter ring**
  inside each container) and its edges are topological adjacency, all known from
  the fixed arrangement tree *before* any coordinates exist. Pathfinding (A* with a
  bend penalty for few-turn routes) assigns each arrow a sequence of slots; channel
  **demand** (the lane count per slot) then falls out as pure topology. Only then
  does layout run once, consuming each channel's computed width exactly where
  `calcDimensions`/`positionNodes` today compute the collapsed inter-sibling gap.
  One pass, no fixpoint iteration.
- **Breakout/break-in is an arrow-relative obstacle rule.** For an arrow
  `a.b.c --> x.y`, obstacles are every node box **except** the ancestors of the
  source and of the target (and the endpoints themselves). A container border is
  passable only to an arrow it must legitimately leave or enter; the perimeter ring
  gives such an arrow a lane to travel before it punches through. Each arrow is thus
  its own pathfinding problem with its own obstacle set — fine at this tool's scale
  (dozens of boxes).

This **reverses the v1 out-of-scope line on orthogonal/auto routing** (`design.md`),
deliberately and with an ADR, while leaving **auto-*placement*** (moving the
author's arrangement) firmly out. It extends M2 cardinal routing: with no obstacles
between two boxes the route collapses to today's straight cardinal line.
**Why:** The author wants routing that *makes space* for lines rather than cutting
into space meant to stay empty — margins should keep spacing lines nicely, so the
corridor must be its own reserved width that displaces boxes. That "push boxes
apart" requirement is exactly what makes the **channel graph beat a pixel-level
visibility/Hanan grid**: routing on geometry would make channel width depend on
positions and positions depend on width — a fixpoint that iteration (and golden
byte-stability) can't tolerate. Deriving the graph from the arrangement tree
decouples demand from pixels and collapses the whole thing to a single
deterministic pass. The arrow-relative obstacle set is what turns "break out of a
parent container" from bespoke nesting logic into one prefix-matching rule. Routing
that pushes boxes apart but **preserves the arrangement** stays inside "manual,
deterministic layout": the displacement is a visible consequence of a local rule
("N arrows route here, so this gap is N lanes wide"), the same way a longer label
already grows a box and nudges its neighbours — arrows are content, and channel
width is content-driven sizing, not auto-placement.
**Implications:** This is a **layout-layer** feature, not a generator post-pass:
routing must run before sizing, and the layout tree gains channel slots as real
objects with width (a localized change to the inter-sibling gap math at
`layout.go` `calcDimensions`/`positionNodes`, plus the perimeter ring). The
generator emits a `<polyline>`/`<path>` per arrow instead of `<line>` (arrowhead
marker unchanged); A* tie-breaking must be totally deterministic (cost = length +
bend penalty, ties broken by a fixed geometric preference, never by map order) or
the golden tests churn. Cardinal endpoint selection (M2) is reused to pick each
arrow's start/end side, so the feature degrades gracefully to current output where
nothing is in the way. **Open knobs deferred to implementation:** the lane-spacing
formula (a fixed multiple of margin vs. font-scaled) and how the mode is expressed
(a document-wide `route: orthogonal` knob first — mirroring the root `margin: N`
default — with a per-arrow override as a follow-up, which finally gives arrows a
presence in the `@layout` layer, per the roadmap parking-lot item). Lane
*separation* for many parallel arrows sharing a slot beyond a simple count, and any
curved (non-orthogonal) routing, stay out. Not yet built — this ADR records the
model before code is written against it.

## 2026-06-19 — A document-wide default margin (root `@layout { margin: N }`)
**Choice:** Let an author override the built-in default margin (8) for a whole
diagram by writing a bare **`margin: N` at the root of an `@layout` sheet** —
i.e. directly inside `@layout { … }`, not in a selector block. It is parsed into
`Document.DefaultMargin *float64` (the parser distinguishes it from a selector by
the `:` after the name, the same `:`-vs-`{` test the declaration grammar already
uses), range-checked by the validator (`>= 0`), and consumed by the generator's
`marginOf`, which now falls back to `DefaultMargin` (else the constant 8) for any
node that sets no margin of its own. v1 accepts only `margin` at the root; any
other property there is a syntax error ("only 'margin' may be set at the @layout
root"). A duplicate is rejected like a duplicate declaration.
**Why:** Reported via the "simple pipeline" demo reading too crunched: every node
defaults to margin 8, so the only way to space a whole diagram out was to set
`margin:` on every node — tedious and noisy. A single global knob is the natural
fix, and it changes *nothing* about the model: the default margin was already a
hardcoded global constant (8); this just makes that constant authorable. Crucially
it is **not a cascade** — the thing `design.md` forbids. There is no parent→child
inheritance and no selector specificity: a node's margin is still either its own
explicit value or *the one* default, exactly as before. So it sits alongside the
other global defaults (font, `direction: vertical`) rather than introducing
action-at-a-distance. **Syntax:** a bare `margin: N` at the sheet root matches the
author's mental model ("a root-level property in `@layout`") with no new keyword,
and the position (root, not a selector) plus the `:` make it unambiguous even
against a node literally named `margin` (`margin { … }` is still that node's
selector). **Scope (margin only):** it is the one property that's both a sensible
document-wide default *and* was the request; `direction` is the only other
plausible one and is a trivial future addition, so the grammar rejects the rest
with a clear, extensible error rather than guessing.
**Implications:** Purely additive — no existing golden moved (no fixture set a
document default). New `default-margin` golden; the `pipeline` site sample now
sets `margin: 16` to fix the reported crunch. The default lives on `Document`
(parsed from the `.ark`), **not** in generator `Options` (`FontSize`/`FontFamily`),
because it is authored content, not a CLI/runtime knob — so there is no
Option-vs-sheet precedence question to answer (a `--margin` flag, if ever wanted,
is a separate decision). Range errors report at line 1, column 1 like the other
validator range checks (the value carries no AST position; the parser catches the
duplicate with position). Per-side margins remain open; this is orthogonal to that
and to the label-band work landed the same day.

## 2026-06-19 — Labelled parents reserve a band for their label
**Choice:** A **parent with a label** (bordered *or* `box: none`) now reserves a
strip — a **label band** — for that label instead of centring it over the
children (which obscured them). Concretely:
- **A new `@layout` property `label: top | bottom`** (default `top`) places the
  band at the top or bottom of the box. It lives on `Declarations` as
  `LabelPos *LabelPosition`, parses/validates/merges/conflicts like any other
  scalar layout property, and is a harmless no-op on a node with no label or no
  children.
- **The band is sized like a leaf box holding that label** (`max(textHeight +
  2·border, fontSize·2)`), so a group title reads as a consistent row, and the box
  **widens** to at least the label's width so the label never clips (and, for a
  borderless group, so the label can't overflow onto a sibling).
- **The band reuses each box type's own packing rule.** It is added to the
  parent's height as a full-width strip; the children lay out in the remaining
  area. In a **bordered** parent the band's inner edge is a wall — the children's
  facing margin collapses against it exactly as it would against the border. In a
  **`box: none`** parent the children pack **flush** below the band, because that
  is how a transparent group packs them everywhere (it reserves no perimeter; its
  effective margin still pushes out to the nearest bordered ancestor). A top band
  shifts the child-layout origin down; a bottom band leaves children at the top and
  sits under them. The change is `band`/`labelBand` plumbing in `calcDimensions`/
  `positionNodes` plus a `labelDim` helper that centres the label in the band in
  `svg.go` — gated only on "has a label and children," not on the box type.
**Why:** The reported bug was that a labelled group centred its label over its
children (`simple-container`, `nesting`), making it unreadable. Reserving a strip
sized like a leaf box (rather than tight to the text) makes a title read as a row
consistent with sibling leaves and is trivial to document ("as tall as the label
would be on its own"). **Extending it to `box: none` groups** (revising this ADR's
first draft, which scoped the band to bordered nodes) is the author's call: a
borderless group's centred label is *also* obscured by its children — "if no label
is desired, don't add one; otherwise reserve space for it." The scoping worry was
that a band contradicts box:none transparency, but it doesn't: making the band
*flush-packed reserved space* rather than a wall keeps the transparent group
adding no perimeter of its own (its children's margins still push through on every
non-band side), so the band is consistent with — not a violation of — the
box:none model. Bordered parents keep the wall semantics they already had. **Top
default** matches the conventional title-bar position; **bottom** is offered for
captions. Reusing the keyword `label` (text in a node body, position in `@layout`)
mirrors the existing anchor name/position split — same word, two layers.
**Implications:** Only **labelled parents** change; leaves and unlabelled parents
are byte-identical, and so is any `box: none` group **without** a label (the
common case — `@group`s and `kind: invisible` groupings carry none), which is why
no existing golden moved for the box:none extension. Four goldens
(`simple-container`, `kind-and-use`, `arrangement`, `complex-layout`) and the
`nesting`/`contexts` site samples were regenerated for the bordered band (taller
boxes, labels in their bands; `complex-layout` also widened `backend` to fit
"Backend Services" and re-routed its arrows); the `group-label` golden carries a
bordered top, a bordered bottom, and a `box: none` labelled group to lock all
three in. A `box: none` group whose label is wider than its children left-aligns
the children under the centred, full-width band (it doesn't stretch children) —
acceptable, and the width reservation is what stops the label overflowing a
sibling. The band is added before the `size` override, so on a *horizontal* parent
`size` scales the band along with the height (the same way `size` already scales a
parent below its content) — untouched here, and no fixture combines them. Per-side
margins and visual styling remain open; `label` position is the first `@layout`
property that is neither geometry nor reuse, a small precedent for the styling
layer `kind` is meant to hook.

## 2026-06-18 — box:none uses an effective margin (no perimeter padding)
**Choice:** Correct the previous implementation of "box:none is transparent to margins." A `box: none` node now reserves **no perimeter padding** of its own — its border box is the tight bounding box of its children (with collapsed inter-sibling gaps) — and instead exposes an **effective margin** = `max(own margin, max child margin)` that its parent reserves around it. Implemented by computing `l.margin` (the effective margin) in `calcDimensions` and dropping the `wall` flag entirely: a node reserves perimeter iff it is itself bordered.
**Why:** The earlier fix made a walled box:none group reserve perimeter padding *like a bordered box*. That padding is a real gap that does **not** collapse with the group's siblings, so a box:none row stacked above a normal node showed a doubled vertical channel (the row's bottom padding *plus* the row→sibling gap = 16, while every other channel was 8) — reported on the bounded-contexts example. Modelling the group's margin as an effective margin that *collapses* like any other margin makes every channel uniform again, including the one across the transparent group's edge. This supersedes the prior ADR's "a child under a group with a non-zero margin is inset by both" — it is now inset by the *larger* of the two, never the sum.
**Implications:** `box: none` groups no longer stretch their children to the cross axis (a transparent container imposes no sizing), so a vertical group's children keep their natural widths — reverting the `arrangement`/`kind-and-use` goldens to tighter, natural-width layouts (regenerated; `contexts` site sample too). Only diagrams with a `box: none` group inside a bordered parent change; bordered-only diagrams are byte-identical. A group's own margin still matters when it is *larger* than its children's (it can request more space) but never stacks. The `wall`-flag plumbing is gone, simplifying `calcDimensions`/`positionNodes`.

## 2026-06-18 — Margins collapse, and box:none is transparent to them
**Choice:** Revise two specifics of the M1 box model (2026-06-18) after they produced wrong-looking spacing:
- **Adjacent margins collapse to the larger, not the sum.** The channel between two siblings is `max(a.margin, b.margin)`, and a child's gap to its parent's wall is its own margin — so every channel (sibling↔sibling and child↔wall) is one uniform margin wide. Supersedes "adjacent sibling margins sum."
- **A `box: none` node is transparent to margins.** Its children's perimeter margins are no longer collapsed unconditionally; they push *through* the invisible node to the nearest **bordered** ancestor and become padding there. Perimeter margins collapse to zero only when there is no bordered ancestor (an invisible chain up to the document root — e.g. a top-level group), preserving "no phantom canvas padding." Implemented by threading a `wall` flag through `calcDimensions`/`positionNodes`: a bordered node sets it, an invisible node passes it through, and perimeter is reserved iff a wall encloses the node.
**Why:** Two author reports. (1) A `box: none` row nested in a bordered container packed its children flush against the container's border (overlapping its label), while a sibling normal node got its margin — because the M1 rule collapsed the row's children's perimeter margins even though there *was* a wall (the bordered grandparent) for them to sit against. The collapse is only correct at the document root, which has no wall. (2) Three siblings showed double-width channels between them (8+8) but single-width margins to the container (8) — non-uniform — because packing each node's full margin box flush sums adjacent margins. Collapsing to the max makes channels uniform, which is what the eye expects and what CSS does for the common case. The M1 ADR had explicitly chosen summing for simplicity and "no surprising CSS behaviour," but the surprise turned out to be the *summing*.
**Implications:** Every multi-child diagram gets tighter, uniform spacing; all five goldens and the three site samples were regenerated (smaller canvases; `box: none`/`@group` children now inset to the wall and stretch to the cross axis like any walled child). A `box: none` group still keeps its *own* margin, so a child under a group with a non-zero margin is inset by both (set the group's `margin: 0` for full transparency — what the reporting example did). The auto-cardinal arrow tests' coordinates shifted with the collapsed gaps. Still uniform-only margins; per-side margins remain open.

## 2026-06-18 — Top-level statements parse in any order (single-pass)
**Choice:** The parser reads node definitions, standalone `@layout` sheets, and arrow statements in a **single pass that accepts them in any order**, dispatching each by lookahead (an identifier that reaches `-->`, after an optional dotted path and `#anchor`, is an arrow; `@` starts a sheet; otherwise it's a node). This replaces the previous two-phase parse (all nodes/sheets, then a trailing block of arrows).
**Why:** The two-phase parser broke the moment an arrow appeared before an `@layout` block: the first arrow flipped it into arrow-mode and every later `@layout`/node was then mis-parsed as an arrow, producing a cascade of bogus "Expected '-->'" errors. Authors reasonably want to colocate an arrow with the nodes it connects (e.g. an arrow right after the two contexts it links, above the layout sheet), so order should not matter. Arrow endpoints are resolved by the validator against the fully-built node map, so forward references (an arrow naming a node defined later) were already fine — only the parser's phase split was forcing arrows to the end.
**Implications:** Strictly more permissive — every previously valid document still parses identically (trailing-arrow style is unchanged), and arrows/sheets/nodes may now interleave freely. The disambiguation rests entirely on the existing `isArrowStatement` lookahead (a node is `id {`, an arrow is `id … -->`), so there is no ambiguity. `parseArrows` (the old second phase) is gone, folded into the single top-level loop.

## 2026-06-18 — `@layout` regrouping M5: implementation
**Choice:** Implement the regrouping phase: an anonymous `@group { … }` inside a node's `@layout` block wraps sibling children into an invisible layout sub-container, and a node's children can be reordered by listing them. Concretely:
- **Grammar.** Inside an `@layout` block a bare identifier (no `:`) is an *arrangement child reference*; `@group { … }` is a wrapper whose body is the same grammar (declarations + nested arrangement). The trailing `:` is what disambiguates a property (`direction: …`) from a child reference (`payments`).
- **AST — node and group unified.** `Declarations` gains `Arrangement []ArrangementItem`; an `ArrangementItem` is either a `ChildID` or a `Group *Declarations`. A group *is* a `Declarations` whose own `Arrangement` holds its nested items — so node-level and group-level arrangements are the same shape and nest recursively. No separate Group type.
- **Direct-only resolution.** The arrangement never flows through the `@use`/`kind` imported tier: `mergeDecls` ignores `Arrangement`, and the resolver copies it solely from a node's direct rules (last wins). A `@block` carrying an arrangement is a validation error; `@use` inside a `@group` is a parse error. Groups are parameterless w.r.t. reuse in v1.
- **Groups are invisible and unaddressable.** A `@group` always renders as `box: none` (reusing M1's invisible-box layout — collapsed perimeter margins, no child stretch) and adds **no path segment**: a child inside a group keeps its real dotted path, so arrows and anchors are unaffected. Groups have no id, label, or anchors (any `box`/`anchor` written on a group is ignored). In the generator a group is a synthetic `layoutNode` with `node == nil`, skipped by box/anchor collection and SVG emission.
- **Validation — refinement invariant.** Two checks keep the layout tree a *refinement* of the semantic tree: **same-parent** (every reference, including those nested in groups, must be a direct child of the node) and **completeness** (once a node is arranged, each direct child is referenced exactly once — no foreigners, duplicates, or omissions). Two direct rules both arranging the same node is a conflict, like any duplicate direct property.
**Why:** Regrouping is purely presentational nesting (an HTML wrapper `<div>` for the layout layer), so it belongs in `@layout`, not the semantic tree — and the same-parent + completeness rules are exactly what guarantee a group can never make a node appear inside a box it isn't semantically part of. Modelling a group as a nested `Declarations` reuses the whole box model (direction/size/margin, invisible layout) and the existing merge/range machinery with almost no new types. Keeping the arrangement direct-only avoids the question of how an imported, node-specific child list would even make sense (child ids don't match across nodes) and keeps `@block` genuinely reusable.
**Implications:** Completes the near-term layered-authoring arc (M3 split → M4 reuse → M5 regrouping). The generator's `buildLayoutTree` now branches on a resolved arrangement; everything downstream (sizing, positioning, canvas) already handles invisible containers, so groups needed only the `node == nil`/`isGroup` guards. A new `arrangement` golden locks the canonical example in; existing fixtures are untouched (none use `@group`). Still scoped out: `@use`/reuse *of* an arrangement, bordered/labelled groups, and per-side margins.

## 2026-06-18 — Generated docs site + live WASM playground on the Examples page
**Choice:** Make the GitHub Pages publish a *generation step* (`scripts/build-site.sh`, run by `pages.yml` before upload) rather than serving the `site/` tree verbatim, and progressively enhance the Examples page into a live editor backed by the existing `wasm/` build. Concretely:
- **Publish-time generation.** The script re-renders every `site/examples/*.ark` to `.svg` through the CLI and builds `site/arkitecture.wasm` (+ copies Go's `wasm_exec.js`) before the artifact is uploaded, so a publish always reflects the current library. The same script runs locally for preview. Pages now also triggers on Go source changes, not just `site/**`.
- **Artifacts stay un-committed.** The `.wasm` is git-ignored (as ever) and `wasm_exec.js` is added to `.gitignore`; both are produced in CI, never checked in. The example `.svg`s remain committed (a working snapshot / no-JS fallback) but are refreshed at publish.
- **Progressive enhancement, not a rewrite.** `playground.js` loads the WASM and swaps each read-only `<pre>` source for a textarea that re-renders via `arkitectureToSVG` on input (debounced); a Reset control appears once the source diverges; compile errors show beneath the diagram. With JS or WASM unavailable the page is byte-for-byte what it was — static SVGs, read-only source.
- **Standard Go WASM, stripped.** Built with the normal toolchain and `-ldflags="-s -w"` (~2.8 MB), not TinyGo.
**Why:** The user wanted examples re-rendered automatically on publish and the WASM build shown off without changing the page's feel. A generation step removes the manual "regenerate the site SVGs" chore and guarantees the published diagrams match the library. Progressive enhancement keeps the no-JS contract and means the first live render is identical to the static fallback (verified: in-browser output matches the committed CLI SVGs byte-for-byte), so the enhancement is invisible until you start typing. The `wasm/` bridge (`arkitectureToSVG` → `{success, svg, errors}`) already existed, so the page is the only new surface. Building in CI rather than committing the binary keeps the repo free of a multi-MB artifact and honours the "never commit `*.wasm`" rule. TinyGo was rejected: it would shrink the download but risks stdlib/`syscall/js` gaps and a second toolchain, for a docs-page nicety — standard Go with `-s -w` is good enough.
**Implications:** GitHub Pages now needs the Go toolchain at publish (added to `pages.yml`) and publishes more often (Go changes trigger it). The committed `site/examples/*.svg` can drift from the library between publishes — they're a snapshot, and there is no CI check yet that they're current (the golden test covers `generator/testdata`, not `site/`); the publish always corrects them. Page weight grows by a lazily-fetched ~2.8 MB on the Examples page only. This realises the parking-lot "web playground" idea in miniature (per-example live editing) without a separate playground page; a fuller standalone playground and a published JS/TS package remain open.

## 2026-06-18 — `@layout` reuse + `kind` M4: implementation
**Choice:** Implement the reuse phase of the layered `@layout` model: `@block <name> { decls }` definitions and `@use <name>` imports, plus `kind` hooking a layout block. Concretely:
- **Grammar & AST.** `@block` is a top-level item inside an `@layout` sheet (a sibling of selector blocks); `@use` appears inside any declarations block (a selector, an inline `@layout`, or a `@block` body — the last giving composition). The AST gains `ast.Use{Block, Line, Column}`, `ast.Block{Name, Decls, Uses, …}`, `Document.Blocks`, and a `Uses []Use` field on `LayoutRule`. A `@use` records the **block-name token's** position for diagnostics.
- **Two-tier resolution.** The resolver computes each node's layout as *imported* then *direct*: the `kind` baseline first (lowest), then each `@use` in source order (later wins within the tier), then the node's direct declarations (which override imports with no conflict). A block expands recursively — its own `@use`s, then its own decls — so composition is itself import-then-direct. Expansion carries a visiting set so a cycle stops instead of looping.
- **Built-in kinds.** Shipped as `ast.BuiltinBlocks()` (a fresh map per call): just `invisible → box: none` for v1. The block table seeds built-ins, then layers user `@block`s on top, so a user `@block invisible` **overrides** the built-in (last definition wins; duplicate user blocks likewise last-win — not an error).
- **Validation.** An explicit `@use` of a name with no `@block` and no built-in is a **reference error** (reported at the `@use`); `@use` composition **cycles** are a reference error (coloured DFS over the block graph, each distinct cycle reported once at the back-edge target). An **unknown `kind` is *not* an error** — see below.
**Why:** Reuse is the payoff of the split (one bundle pulled into many nodes, and `kind` as a semantic hook for shared style). Keeping resolution to exactly two tiers — and making *direct beats imported* a non-conflict while two *direct* values still conflict — preserves the "no cascade, explicit override" rule from the epic's ADR. Recursive import-then-direct expansion makes a block behave like a mini-selector, so composition needs no special precedence rules. Seeding built-ins into the same table that user blocks write to makes "redeclarable" fall out for free.
**Unknown-kind decision (resolves a `design.md` open question):** an explicit `@use foo` with no such block is an **error**, but an unknown `kind: foo` is a **no-op** (the node keeps the semantic tag; no layout baseline applies). The split is explicit-vs-implicit: `@use` is a layout-layer request the author made — a missing target is a real mistake — whereas `kind` is a *semantic* tag (and the intended hook for a future styling layer) that should be authorable without forcing a matching layout block to exist, or the layers would re-couple. This is also the reversible direction (a no-op can tighten to an error later without breaking documents). It overrides the roadmap's earlier lean toward erroring on both.
**Implications:** Generation is unchanged — `kind`/`@use` resolve to ordinary `Declarations`, and the generator already renders `box: none`, so a `kind: invisible` node becomes an invisible grouping (it collapses perimeter margins, changing layout, as intended). The resolver still runs only on validated documents, but its cycle guard keeps `GenerateSVG` (which skips validation) loop-safe. v1 stays parameterless (no mixin args) and single-kind-per-node. A new golden (`kind-and-use`) locks in the end-to-end behaviour; existing goldens are untouched (no fixture used `kind`/`@use`). Still open: per-side margins; multiple kinds per node; cross-file layout sheets.

## 2026-06-18 — `@layout` split M3: implementation
**Choice:** Implement the 2026-06-17 "separate semantics from layout" ADR's first phase. Concretely:
- **Dropped the inline layout shorthand.** A node body is now purely semantic (`label`, `kind`, `anchors: [names]`, child nodes). `direction`/`size`/`margin`/`box`/anchor positions live *only* in `@layout` — bare `direction:` on a node is a syntax error pointing the author to `@layout`. Chosen over keeping shorthand so a reader can always tell semantics from presentation; an inline `@layout {…}` block still colocates a node's own layout.
- **Removed the `group` keyword entirely.** A borderless grouping is a `box: none` node, which (unlike the old group) keeps an id and therefore contributes a path segment — former group children move from `c1.x` to `c1.grp.x`. Geometry is unchanged, so SVG stays byte-identical; only reference paths shift.
- **One uniform layout list.** Inline `@layout` is desugared by the parser into a `LayoutRule` whose selector is the enclosing node's full path, so inline and standalone sheet rules are the same "direct" tier and share one conflict/merge path. The AST splits into a semantic `ContainerNode` (no layout fields, `Children []*ContainerNode`, no `Node`/`GroupNode`) and `Declarations`/`LayoutRule` (pointer fields so "unset" is distinguishable for the conflict check).
- **New pure `resolve` stage** merges rules onto paths into `map[path]*Declarations`; the pipeline is parse → validate → resolve → generate. The validator (not resolve) owns all diagnostics: dangling selector (at the selector position), duplicate **direct** property, relocated `size`/`margin`/coord ranges, anchor positions naming a declared name, and arrow anchor-name resolution. Resolve assumes validity and just overlays.
- **`@` tokenizes as its own token**; `@layout` is `@` + identifier, leaving room for `@block`/`@use`/`@group`. `;` became a cosmetic separator (skipped like whitespace). An unpositioned but declared anchor resolves to centre.
**Why:** The split is the whole point of the epic — semantic structure is stable, layout is fiddly — so M3 commits to it cleanly rather than half-keeping the old inline form. Desugaring inline-to-selector avoids two code paths for "direct" layout. Putting every diagnostic in the validator preserves the existing "validator is the single error stage; generator/resolve are pure" shape. Range checks moved off the parser because values are now layout declarations, not node fields.
**Implications:** A breaking syntax change. All golden fixtures, examples, and site samples were rewritten in the new syntax; every SVG renders byte-for-byte identically (verified by `TestGolden` and by regenerating the site SVGs to a no-op diff), confirming the split is structural, not visual. Range errors now report at line 1, column 1 (validator) instead of the old parse position — consistent with the other validator diagnostics, pending the tracked "AST source positions" work. Still open: per-side margins; the precedence tiers stay a single direct tier until M4 adds `kind`/`@use`.

## 2026-06-18 — Auto-cardinal routing M2: implementation
**Choice:** Implement the 2026-06-17 auto-cardinal ADR with: **4 sides** (N/E/S/W) only; each bare endpoint aims at the **other node's box centre** (never the other end's anchor), so routing is stable whatever the far end does; and **no parser or AST change** — the distinction between a bare `a` (auto-route) and an explicit `a#center` (force centre) is already carried by the arrow string, so it is resolved entirely in the generator. Layout exposes a new full-path→border-box map; an explicit anchor (named or `#center`) still resolves to its fixed position and wins.
**Why:** 4 sides keep the side choice unambiguous (a single dominant-axis test) and match the design; 8 would add corner cases for little gain. Aiming at the box centre (not the far anchor) means a named anchor — which can serve many arrows — never destabilises the auto-routed end. The roadmap had pencilled in a parser change to "distinguish no-anchor from `#center`", but inspection showed the raw `ast.Arrow` strings already differ (`a` vs `a#center`), so the only place that ambiguity was being lost was the generator's old `parseTarget`, which defaulted no-anchor to centre.
**Implications:** Cardinal routing depends on M1's margins for a gap to span (flush boxes would give a zero-length edge-to-edge arrow). Anchorless arrows in every diagram move from centre-to-centre to edge-to-edge (goldens and the `pipeline` site sample regenerated; explicitly-anchored arrows such as the `contexts` sample are unaffected). Still open: whether to ever offer 8 directions.

## 2026-06-18 — Box model M1: defaults and margin arithmetic
**Choice:** Implement the margin-based box model (the 2026-06-17 ADR) as inline `margin` / `box` node properties, resolving the open details as:
- **Default margin of 8** user units, non-zero and uniform on all four sides; `margin: 0` restores the old flush packing. Authored margins are absolute layout units (like coordinates), not a fraction of anything.
- **Adjacent sibling margins sum** — no CSS-style collapse-to-max. The gap between two siblings is the sum of their facing margins, which falls straight out of packing each node's margin box flush.
- **Uniform margin only** in v1 (no per-side `margin-top`/…).
- **`box` takes a bareword** (`box: none` / `box: default`), matching the design's surface syntax instead of the quoted-string form `direction` uses today.
- A **`box: none` container keeps its own margin** (it is a real node that merely draws no border); only the layout-only `group` contributes zero margin. Both, as invisible *parents*, collapse their children's perimeter margins.
**Why:** A visible default is what the box model is *for* — it gives auto-cardinal routing (M2) a gap to attach to and makes nesting legible; 8 is a clean, font-independent spacing unit that reads well at the common 12–16px sizes. Summing is the simpler, more predictable rule and the natural consequence of "each node owns its margin box"; collapse-to-max imports CSS's most surprising behaviour for no benefit here. Uniform-only keeps the grammar minimal and is forward-compatible with per-side later. Keeping a margin on `box: none` follows from "it's a node, not a wrapper" — its ID, label, and anchors already separate it from a `group`.
**Implications:** Every multi-node diagram changes; all goldens and the three site sample SVGs were regenerated. The "exact-fit canvas, no padding" rule now holds only at the invisible document root (top-level perimeter margins still collapse) — bordered parents inset their children. Still open and deferred: per-side margins, and whether `box: none` should instead be fully margin-transparent like a `group`.

## 2026-06-17 — Margin-based box model
**Choice:** Express inter-element spacing as a per-node **`margin`** (a layout property), implemented with a real box model that distinguishes a node's **border box** (the visible rectangle) from its **margin box** (border box + margins). Chosen over a parent-level `gap`. A child's margin counts toward a **bordered** parent's size (like padding inside the border) but **collapses** against an **invisible** (`box: none`) parent — including the document root — so it never inflates a box with no wall to sit against. Anchors and arrows attach to the border box.
**Why:** Margin is the more natural authoring model ("this node wants room around it") and composes per-node, unlike a parent-level gap. It is also the enabler for auto-cardinal arrow routing: without a gap, edge-attached arrows between touching boxes degenerate to zero length (Arkitecture packs siblings flush). The invisible-vs-bordered rule is what keeps an invisible grouping from gaining phantom padding while a real container still insets its children.
**Implications:** A substantial layout refactor — sizing, positioning, and canvas all become box-model-aware, tracking border vs margin boxes and margin collapse. It revises the "no padding / exact-fit canvas" stance in `design.md` once a non-zero margin is used. `margin`/`box` are layout properties, so they sit inline today and move into `@layout` when the layer split lands. Open: adjacent-sibling collapse (sum vs max), per-side margins, and the default margin value (0 preserves today's compact look).

## 2026-06-17 — Auto-cardinal arrow endpoints
**Choice:** An arrow that does not name an anchor (`a --> b`) auto-routes: each end attaches to the cardinal side (N/E/S/W) of its box facing the other box's centre, chosen by the dominant axis of the centre-to-centre vector (`|dx| ≥ |dy|` → E/W, else N/S; exact diagonals favour horizontal). Named anchors stay single fixed points (centre if unpositioned). An explicit anchor always overrides.
**Why:** Centre-to-centre lines cut through the boxes and read poorly; "nearest side facing the target" is the cheap heuristic that makes plain arrows look intentional. A named anchor can serve many arrows, so it has no single "other node" to aim at — hence the heuristic belongs to anchor-less endpoints, not to anchor positions.
**Why it doesn't violate "manual, deterministic layout":** it is auto-*routing*, not auto-*placement* — it never moves a box the author positioned, only picks where a line attaches, as a deterministic function of author-controlled positions, and is fully overridable. This is the one bounded "automatic" behaviour in the tool.
**Implications:** Lives in the generator's endpoint resolution and is **independent of the `@layout` epic** — it can ship on the current Go pipeline as a standalone improvement. Open: 4 vs 8 directions; whether an unpositioned *named* anchor should error instead of defaulting to centre.

## 2026-06-17 — Separate semantics from layout (CSS without the cascade)
**Choice:** Author diagrams in two layers — a **semantic** layer (the `.ark` structure: node `id`/`label`/`kind`, named anchors, containment, arrows) and a **layout** layer (`@layout`: `direction`, `size`, anchor positions, `box`, child arrangement) — using `@`-prefixed directives. Collapse node and group into **one node type** (`box: none` replaces the layout-only group). Add exact-path selectors, opt-in reusable blocks (`@block`/`@use`), a semantic `kind` that implicitly `@use`s the block of the same name, and anonymous presentational regrouping (`@group`). Deliberately omit CSS's cascade.
**Why:** Semantic structure is stable; layout is fiddly and frequently retuned. Separating them lets layout be edited, reused, and even swapped (multiple layouts over one model) without disturbing semantics. The separation is modelled on HTML/CSS, but CSS's *pain* — "why is this here?" answered three selectors away — comes from the cascade (specificity, inheritance, competing rules), which is the exact action-at-a-distance the core principle forbids. So we keep the separation and drop the cascade. `@` was chosen over `!` for the CSS at-rule feel and to avoid the `!important` connotation.
**The rules that preserve determinism:**
- Selectors are **exact dotted paths** only — no wildcards, specificity, or inheritance. One selector names one node.
- Resolution has two precedence tiers: **imported** (`kind` baseline, then `@use` — source order wins within) is overridden by **direct** declarations naming the node (inline or sheet). Direct-over-imported is not a conflict; two *direct* rules setting the same property on the same node is a **validation error** (relaxable later to import-order-wins for multi-sheet themes).
- `@block`/`@use` is opt-in and explicit, parameterless in v1; cycles are an error.
- `kind` expands to an implicit `@use <kind>` at **lowest precedence** (a baseline); explicit layout overrides it without error. A small set of built-in kinds ships (e.g. `invisible` → `box: none`) and any kind can be redeclared.
- An anonymous `@group` may regroup only **direct semantic children of the enclosing node**, and an arrangement must reference each child **exactly once** — so the layout tree is always a *refinement* of the semantic tree.
- Anchors split into a semantic **name** (arrows reference it) and a layout **position**.
**Implications:** A new **resolve** stage merges the layout layer onto the semantic tree by exact path (the analog of CSS computed style) before generation; the validator gains selector/conflict/`@group`/`@use`/`kind`/anchor-name checks; the generator reads resolved layout rather than properties on the semantic node. This is a breaking syntax change (today's inline `size`/`direction`/`anchors` and the `group` keyword go away or become layout shorthand), and former groups now carry an `id` and a path segment instead of being path-transparent — acceptable pre-1.0. Built in phases (see roadmap): split → reuse (+`kind`) → regrouping. Remaining details (unpositioned-anchor default, unknown-kind handling, inline-vs-sheet precedence, cross-file sheets, arrow-anchor routing) are tracked in `design.md`'s open questions.

## 2026-06-17 — Rewrite the implementation in Go
**Choice:** Port Arkitecture from TypeScript to Go as a single switchover PR, structured library-first: a root `arkitecture` package is the library, with `cmd/arkitecture` (CLI) and `wasm/` (a `js,wasm` build) as thin wrappers over it. Supersedes "Pure TypeScript library with a thin CLI" (2025-06-19).
**Why:** Go compiles a single portable static binary with no runtime to install, which suits a CLI tool better than an npm package. It also has a first-class `GOOS=js GOARCH=wasm` target, giving a clean path to a WASM library for future TypeScript/browser interop without maintaining a second implementation. The pipeline is pure string-in/string-out logic, so it ports almost mechanically.
**Why Go over the alternatives:** Rust was the other portable-binary candidate; Go was chosen for faster porting of this loosely-typed code, a simpler build, and a mature `syscall/js` WASM story. Staying on TypeScript was rejected because "single binary, no Node required" was the whole motivation.
**Implications:** The CLI and the WASM shim must depend on the library, never reimplement it. The error model stays "collected as data, never thrown across stages" (Go: `[]ast.Error`, no panics across boundaries; recover only at the top level). The `string-width` text-measurement decision (2025-06-19) will be re-decided during the generator port — likely a small Go rune-width function — since the npm package can't come along. The port lands stage by stage on one branch: tokenizer + parser first; validator, generator, and CLI watch before the PR merges.

## 2026-06-17 — Go text measurement: a built-in rune-width approximation
**Choice:** Estimate label dimensions with a small built-in display-width function (rune count, East-Asian-wide/emoji counted as 2, combining/format marks as 0) — the same shape as `string-width` — rather than adding a dependency or measuring real font metrics. Resolves the open item from the Go rewrite ADR (supersedes the 2025-06-19 `string-width` decision).
**Why:** Layout must run headless and in WASM without a canvas, and the output must be deterministic across platforms. A built-in width keeps the module dependency-free (the portable-binary/WASM goal) and reproduces the TypeScript output exactly for ASCII — confirmed by the byte-for-byte golden tests.
**Implications:** Measurement is an approximation, not pixel-accurate metrics — fine for box packing. `generator/text.go` is the single seam to replace if true metrics are ever needed.

## 2026-06-17 — Watch mode: stdlib modtime polling, not fsnotify
**Choice:** Implement `--watch` by polling the input file's modification time (200ms) in the CLI, rather than depending on `fsnotify`.
**Why:** Watch is a CLI-only dev convenience; polling a single file is simple, dependency-free, and cross-compiles cleanly — consistent with the minimal-dependency posture. Event-based watching would add a dependency for negligible benefit at this scale.
**Implications:** Up to ~200ms latency before a change is picked up (imperceptible in practice). Revisit with fsnotify only if multi-file or directory watching is ever needed.

## 2026-06-17 — Adopt the project-template docs structure
**Choice:** Replace the original `specs/` + `.specs/` specifications, `prompt-plan.md`, and `.claude/commands/dev.md` workflow with `CLAUDE.md` + `docs/{design,architecture,roadmap,decisions}.md`, matching [kurrik/project-template](https://github.com/kurrik/project-template).
**Why:** The original scaffolding was a one-shot architecture spec plus a 15-step prompt plan that drove the initial build and then went stale; it duplicated content (`specs/` and `.specs/` held near-identical copies) and the prompt plan no longer reflected the shipped code. The project-template layout separates durable concerns (vision / layout / status / decisions) and is the shared convention across the owner's projects.
**Implications:** `docs/` is now the source of truth and must be updated in the same commit as the change it describes. The `specs/` tree is gone; the annotated DSL reference survives as `examples/annotated.ark`. Future work is steered by `CLAUDE.md` + `docs/`, not slash-command playbooks.

## 2025-06-19 — Manual layout instead of automatic graph layout
**Choice:** The author controls layout via nesting, `direction`, and `size`; the tool only measures text and packs boxes deterministically.
**Why:** Auto-layout tools (Graphviz, Mermaid) produce output you can't precisely control and can't correct without fighting the algorithm. The goal is predictable, hand-tuned high-level diagrams (e.g. DDD bounded contexts) kept as text in version control.
**Implications:** No layout engine to fight — but also no auto-routing or auto-placement; arrangement is the author's responsibility. Rules out force-directed / hierarchical layout features and curved auto-routing.

## 2025-06-19 — Pure TypeScript library with a thin CLI
**Choice:** Ship a side-effect-free library (DSL string in → `Result` out) compiled to CommonJS, with all file I/O isolated in the CLI.
**Why:** Keeps the core runnable in both Node and the browser, trivially testable, and reusable as an SDK; the CLI is just an adapter over the same function.
**Implications:** No filesystem or DOM assumptions inside the library; only the CLI touches disk (plus `chokidar` for `--watch`). The public API and the CLI must stay in sync as the pipeline evolves.

## 2025-06-19 — Collected errors, never thrown across stages
**Choice:** Every failure is a `ValidationError` (`line`, `column`, `message`, `type`); stages return arrays and don't fail-fast. Only the top-level entry point wraps an unexpected throw.
**Why:** Authors want to see *all* problems from one run, with positions, instead of fix-one-rerun. A single uniform error shape serves both the CLI and the API.
**Implications:** Each stage must accumulate rather than abort, and a run reports syntax, reference, and constraint errors together. Throwing across a stage boundary is a bug.

## 2025-06-19 — `string-width` for text measurement
**Choice:** Estimate label dimensions with the `string-width` package (default Arial 12px, 1.2× line height) rather than DOM/canvas measurement.
**Why:** Layout needs box sizes *before* rendering, and the library must work headless in Node and the browser without a canvas. `string-width` gives consistent, dependency-light width estimates everywhere.
**Implications:** Measurement is a cell-width approximation, not true font metrics — fine for box packing, and the font is overridable via options. If pixel-accurate text fitting is ever required, this is the seam to revisit.

## 2025-06-19 — Golden-file tests for SVG output
**Choice:** Render `.ark` fixtures and diff against checked-in `.svg`/`.error` references, regenerated via `npm run golden:generate`.
**Why:** SVG output is large and positional; golden files catch unintended rendering changes that hand-written assertions would miss, while staying easy to review as diffs.
**Implications:** Intentional output changes require regenerating and reviewing the fixtures; the generator's exact output is part of the test contract.
