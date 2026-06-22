package generator

import (
	"math"

	"github.com/kurrik/arkitecture/ast"
)

// gridInfo is the resolved geometry of a grid node, computed in calcGrid and
// consumed in positionGrid. colW/rowH are 1-based track sizes (index 0 unused);
// gap is the uniform spacing between tracks (and the perimeter padding inside a
// bordered grid); placed[i] is the resolved placement of children[i].
type gridInfo struct {
	cols, rows int
	colW, rowH []float64
	gap        float64
	bordered   bool
	placed     []ast.PlacedCell
}

// calcGrid sizes a grid node and its tracks. Children have already been sized
// (calcDimensions recurses first). Track sizing is joint on both axes: each
// single-track cell grows its column to its width and its row to its height; a
// spanning cell then distributes any shortfall evenly across the tracks it
// covers. The node's box is the summed tracks plus inter-track gaps (plus a
// perimeter gap and label band where applicable).
func calcGrid(l *layoutNode, fontSize, gap float64, bordered bool, bw float64) {
	l.margin = gap

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
	// Spanning pass: distribute any shortfall across the spanned tracks so the
	// cell fits within its column/row span.
	for i, pc := range placed {
		c := l.children[i]
		if pc.ColSpan > 1 {
			distribute(colW, pc.Col, pc.ColSpan, cols, c.dim.width, gap)
		}
		if pc.RowSpan > 1 {
			distribute(rowH, pc.Row, pc.RowSpan, rows, c.dim.height, gap)
		}
	}

	contentW := sumTracks(colW, 1, cols) + gap*float64(cols-1)
	contentH := sumTracks(rowH, 1, rows) + gap*float64(rows-1)
	if bordered {
		contentW += 2 * gap
		contentH += 2 * gap
	}
	l.dim.width = math.Max(contentW, labelW)
	l.dim.height = contentH + band
	l.grid = &gridInfo{cols: cols, rows: rows, colW: colW, rowH: rowH, gap: gap, bordered: bordered, placed: placed}
}

// positionGrid places each grid child at its track origin and aligns it within
// its (possibly spanning) cell per the child's justify/align (default stretch).
// childY is already past any top label band.
func positionGrid(l *layoutNode, x, childY float64) {
	g := l.grid
	x0, y0 := x, childY
	if g.bordered {
		x0 += g.gap
		y0 += g.gap
	}

	colX := make([]float64, g.cols+2)
	colX[1] = x0
	for c := 2; c <= g.cols+1; c++ {
		colX[c] = colX[c-1] + g.colW[c-1] + g.gap
	}
	rowY := make([]float64, g.rows+2)
	rowY[1] = y0
	for r := 2; r <= g.rows+1; r++ {
		rowY[r] = rowY[r-1] + g.rowH[r-1] + g.gap
	}

	for i, pc := range g.placed {
		ch := l.children[i]
		cellX := colX[pc.Col]
		cellW := sumTracks(g.colW, pc.Col, pc.Col+pc.ColSpan-1) + g.gap*float64(pc.ColSpan-1)
		cellY := rowY[pc.Row]
		cellH := sumTracks(g.rowH, pc.Row, pc.Row+pc.RowSpan-1) + g.gap*float64(pc.RowSpan-1)

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

func distribute(track []float64, start, span, max int, need, gap float64) {
	end := start + span - 1
	if end > max {
		end = max
	}
	have := sumTracks(track, start, end) + gap*float64(span-1)
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
