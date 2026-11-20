package cmd

import (
	"image"
	"image/draw"
)

// buildTileLayer creates a transparent image with the given bounds and draws
// watermark onto it at every anchor in positions. Tiles are assumed not to
// overlap, which is guaranteed by the non-negative spacing used to derive the
// positions. Tiles extending past an edge are clipped to the layer bounds.
func buildTileLayer(bounds image.Rectangle, watermark image.Image, positions []image.Point) *image.NRGBA {
	layer := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	origin := bounds.Min
	for _, p := range positions {
		// Translate the anchor from the source image's coordinate space (its
		// bounds may not start at 0,0) into the layer's 0-based space.
		targetRect := image.Rectangle{
			Min: p.Sub(origin),
			Max: p.Sub(origin).Add(watermark.Bounds().Size()),
		}
		draw.Draw(layer, targetRect, watermark, watermark.Bounds().Min, draw.Over)
	}
	return layer
}

// tilePositions returns the top-left anchor point of every watermark tile
// needed to cover a canvas of the given width and height.
//
// The tiles form a regular grid. The horizontal pitch is tileW+spacingX and
// the vertical pitch is tileH+spacingY. Every second row is shifted
// horizontally by stagger (a negative stagger shifts it to the left). Tiles
// that only partially intersect the canvas are still returned; the caller is
// expected to clip them at the canvas boundary, which is what makes edge
// cropping possible.
//
// Positions are returned in row-major order. All canvas dimensions and pitches
// are assumed positive and the arguments are assumed to have passed
// WatermarkOptions.Validate plus the canvas size guard.
func tilePositions(canvasW, canvasH, tileW, tileH, spacingX, spacingY, stagger int) []image.Point {
	stepX := tileW + spacingX
	stepY := tileH + spacingY

	var positions []image.Point
	for top := 0; top < canvasH; top += stepY {
		row := top / stepY
		offset := 0
		if row%2 == 1 {
			offset = stagger
		}

		// Smallest index whose tile reaches the left canvas edge.
		startK := floorDiv(-offset, stepX)
		// Largest index whose tile reaches the right canvas edge.
		endK := floorDiv(canvasW-1-offset, stepX)
		for k := startK; k <= endK; k++ {
			left := offset + k*stepX
			positions = append(positions, image.Pt(left, top))
		}
	}
	return positions
}

// floorDiv returns the integer floor of n/d. Go's built-in division truncates
// toward zero, so negative numerators need an explicit adjustment.
func floorDiv(n, d int) int {
	q := n / d
	r := n % d
	if r != 0 && ((n < 0) != (d < 0)) {
		q--
	}
	return q
}
