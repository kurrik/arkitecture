package generator

import (
	"math"

	"github.com/kurrik/arkitecture/ast"
)

// gridInfo is the resolved geometry of a grid node, computed in calcGrid and
// consumed in positionGrid. colW/rowH are 1-based track sizes (index 0 unused);
// colGap[c]/rowGap[c] are the collapsed channel between track c and c+1 (1-based,
// the larger facing margin of the children adjacent across that boundary);
// leftPerim/topPerim are the bordered grid's low-edge perimeter insets; placed[i]
// is the resolved placement of children[i].
type gridInfo struct {
	cols, rows          int
	colW, rowH          []float64
	colGap, rowGap      []float64
	leftPerim, topPerim float64
	bordered            bool
	placed              []ast.PlacedCell
}

// calcGrid sizes a grid node and its tracks using the same margin-collapse box
// model as 1-D packing, so a single-track grid reproduces a `direction` stack.
// Children have already been sized (calcDimensions recurses first). Track sizing
// is joint on both axes: each single-track cell grows its column to its width and
// its row to its height; a spanning cell then distributes any shortfall across the
// tracks it covers. Each inter-track channel is the *collapsed* (larger) facing
// margin of the children across it, not a uniform gap; a bordered grid adds a
// perimeter sized from its edge children's margins; a transparent box:none grid
// adds none and carries its children's margins outward as its effective margin.
func calcGrid(l *layoutNode, fontSize, ownMargin float64, bordered bool, bw float64) {
	// Effective margin: a transparent box:none grid carries its children's margins
	// outward (the larger of its own and theirs); a bordered grid is the boundary,
	// so its effective margin is just its own.
	l.margin = ownMargin
	if !bordered {
		for _, c := range l.children {
			l.margin = math.Max(l.margin, c.margin)
		}
	}

	var band, labelW float64
	if label, ok := nodeLabel(l); ok {
		band = labelBandHeight(label, fontSize, bw)
		labelW = textWidth(label, fontSize) + 2*bw
	}
	l.labelBand = band

	cells := make([]ast.GridCell, len(l.children))
	for i, c := range l.children {
		id := ""
		if c.node != nil {
			id = c.node.ID
		}
		cells[i] = ast.GridCell{
			ChildID: id,
			Col:     derefInt(placementCol(c)),
			Row:     derefInt(placementRow(c)),
			ColSpan: derefInt(placementColSpan(c)),
			RowSpan: derefInt(placementRowSpan(c)),
		}
	}

	spec := ast.GridSpec{Cols: *l.decls.Cols, Rows: l.decls.Rows}
	placed, usedRows, _ := ast.PlaceGrid(spec, cells)
	cols := spec.Cols
	if cols < 1 {
		cols = 1
	}
	rows := usedRows
	if rows < 1 {
		rows = 1
	}

	colW := make([]float64, cols+1)
	rowH := make([]float64, rows+1)

	// Base pass: single-track cells set their column width / row height.
	for i, pc := range placed {
		c := l.children[i]
		if pc.ColSpan == 1 && pc.Col <= cols {
			colW[pc.Col] = math.Max(colW[pc.Col], c.dim.width)
		}
		if pc.RowSpan == 1 && pc.Row <= rows {
			rowH[pc.Row] = math.Max(rowH[pc.Row], c.dim.height)
		}
	}

	// Collapsed channels and perimeters: each child contributes its margin to the
	// boundaries it faces (the gap to its left/right column and top/bottom row, or
	// the perimeter when it sits on an edge); channels collapse to the larger
	// facing margin, mirroring 1-D packing.
	colGap := make([]float64, cols+1)
	rowGap := make([]float64, rows+1)
	var leftPerim, rightPerim, topPerim, botPerim float64
	for i, pc := range placed {
		m := l.children[i].margin
		cL, cR := pc.Col, pc.Col+pc.ColSpan-1
		rT, rB := pc.Row, pc.Row+pc.RowSpan-1
		if cL-1 >= 1 {
			colGap[cL-1] = math.Max(colGap[cL-1], m)
		}
		if cR <= cols-1 {
			colGap[cR] = math.Max(colGap[cR], m)
		}
		if rT-1 >= 1 {
			rowGap[rT-1] = math.Max(rowGap[rT-1], m)
		}
		if rB <= rows-1 {
			rowGap[rB] = math.Max(rowGap[rB], m)
		}
		if cL == 1 {
			leftPerim = math.Max(leftPerim, m)
		}
		if cR == cols {
			rightPerim = math.Max(rightPerim, m)
		}
		if rT == 1 {
			topPerim = math.Max(topPerim, m)
		}
		if rB == rows {
			botPerim = math.Max(botPerim, m)
		}
	}

	// Spanning pass: distribute any shortfall across the spanned tracks (counting
	// the collapsed channels the span swallows) so the cell fits its column/row span.
	for i, pc := range placed {
		c := l.children[i]
		if pc.ColSpan > 1 {
			distribute(colW, pc.Col, pc.ColSpan, cols, c.dim.width, sumTracks(colGap, pc.Col, pc.Col+pc.ColSpan-2))
		}
		if pc.RowSpan > 1 {
			distribute(rowH, pc.Row, pc.RowSpan, rows, c.dim.height, sumTracks(rowGap, pc.Row, pc.Row+pc.RowSpan-2))
		}
	}

	contentW := sumTracks(colW, 1, cols) + sumTracks(colGap, 1, cols-1)
	contentH := sumTracks(rowH, 1, rows) + sumTracks(rowGap, 1, rows-1)
	if bordered {
		contentW += leftPerim + rightPerim
		contentH += topPerim + botPerim
	}
	l.dim.width = math.Max(contentW, labelW)
	l.dim.height = contentH + band
	l.grid = &gridInfo{
		cols: cols, rows: rows, colW: colW, rowH: rowH, colGap: colGap, rowGap: rowGap,
		leftPerim: leftPerim, topPerim: topPerim, bordered: bordered, placed: placed,
	}
}

// positionGrid places each grid child at its track origin and aligns it within
// its (possibly spanning) cell per the child's justify/align (default stretch).
// childY is already past any top label band.
func positionGrid(l *layoutNode, x, childY float64) {
	g := l.grid
	x0, y0 := x, childY
	if g.bordered {
		x0 += g.leftPerim
		y0 += g.topPerim
	}

	colX := make([]float64, g.cols+2)
	colX[1] = x0
	for c := 2; c <= g.cols+1; c++ {
		colX[c] = colX[c-1] + g.colW[c-1] + sumTracks(g.colGap, c-1, c-1)
	}
	rowY := make([]float64, g.rows+2)
	rowY[1] = y0
	for r := 2; r <= g.rows+1; r++ {
		rowY[r] = rowY[r-1] + g.rowH[r-1] + sumTracks(g.rowGap, r-1, r-1)
	}

	for i, pc := range g.placed {
		ch := l.children[i]
		cellX := colX[pc.Col]
		cellW := sumTracks(g.colW, pc.Col, pc.Col+pc.ColSpan-1) + sumTracks(g.colGap, pc.Col, pc.Col+pc.ColSpan-2)
		cellY := rowY[pc.Row]
		cellH := sumTracks(g.rowH, pc.Row, pc.Row+pc.RowSpan-1) + sumTracks(g.rowGap, pc.Row, pc.Row+pc.RowSpan-2)

		px := alignWithin(cellX, cellW, &ch.dim.width, gridAlignOf(placementJustify(ch)))
		py := alignWithin(cellY, cellH, &ch.dim.height, gridAlignOf(placementAlign(ch)))
		positionNodes(ch, px, py)
	}
}

// alignWithin positions a child of size *size within a cell of size cellSize at
// origin cellOrigin, returning the child's coordinate on that axis. stretch
// grows the child to fill the cell; start/end leave its natural size at the
// near/far edge. (No centre in v1 — stretch already centres the box.)
func alignWithin(cellOrigin, cellSize float64, size *float64, a ast.GridAlign) float64 {
	switch a {
	case ast.GridEnd:
		return cellOrigin + cellSize - *size
	case ast.GridStart:
		return cellOrigin
	default: // stretch
		*size = cellSize
		return cellOrigin
	}
}

func distribute(track []float64, start, span, max int, need, gapSum float64) {
	end := start + span - 1
	if end > max {
		end = max
	}
	have := sumTracks(track, start, end) + gapSum
	if need <= have {
		return
	}
	add := (need - have) / float64(span)
	for t := start; t <= end; t++ {
		track[t] += add
	}
}

func sumTracks(track []float64, from, to int) float64 {
	s := 0.0
	for t := from; t <= to && t < len(track); t++ {
		s += track[t]
	}
	return s
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func gridAlignOf(p *ast.GridAlign) ast.GridAlign {
	if p == nil {
		return ast.GridStretch
	}
	return *p
}

func placementCol(l *layoutNode) *int {
	if l.decls == nil {
		return nil
	}
	return l.decls.Col
}
func placementRow(l *layoutNode) *int {
	if l.decls == nil {
		return nil
	}
	return l.decls.Row
}
func placementColSpan(l *layoutNode) *int {
	if l.decls == nil {
		return nil
	}
	return l.decls.ColSpan
}
func placementRowSpan(l *layoutNode) *int {
	if l.decls == nil {
		return nil
	}
	return l.decls.RowSpan
}
func placementJustify(l *layoutNode) *ast.GridAlign {
	if l.decls == nil {
		return nil
	}
	return l.decls.Justify
}
func placementAlign(l *layoutNode) *ast.GridAlign {
	if l.decls == nil {
		return nil
	}
	return l.decls.Align
}
