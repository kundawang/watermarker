package cmd

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/disintegration/imaging"
)

func nan() float64 { return math.NaN() }

// solidImage returns an opaque NRGBA image of the given size filled with c.
func solidImage(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 0
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

// alphaImage returns an image whose RGB channels are rgb and whose alpha is
// alphaAt(x,y), so blend math can be checked pixel by pixel.
func alphaImage(w, h int, rgb color.NRGBA, alphaAt func(x, y int) uint8) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: rgb.R, G: rgb.G, B: rgb.B, A: alphaAt(x, y)})
		}
	}
	return img
}

func TestTilePositionsGridAndSpacing(t *testing.T) {
	// 10x10 tiles, no spacing, over a 25x25 canvas: a 3x3 grid, with the last
	// column and row extending past the canvas (to be clipped by the drawer).
	got := tilePositions(25, 25, 10, 10, 0, 0, 0)
	want := []image.Point{
		{0, 0}, {10, 0}, {20, 0},
		{0, 10}, {10, 10}, {20, 10},
		{0, 20}, {10, 20}, {20, 20},
	}
	assertPoints(t, got, want)
}

func TestTilePositionsSpacingAndStagger(t *testing.T) {
	// 4x4 tiles, spacing 2 -> pitch 6 on both axes, stagger 3 on odd rows.
	got := tilePositions(13, 13, 4, 4, 2, 2, 3)
	want := []image.Point{
		// row 0 at y=0
		{0, 0}, {6, 0}, {12, 0},
		// row 1 at y=6, shifted +3: tiles at 3, 9, and -3 (clipped left edge)
		{-3, 6}, {3, 6}, {9, 6},
		// row 2 at y=12, unshifted
		{0, 12}, {6, 12}, {12, 12},
	}
	assertPoints(t, got, want)
}

func TestTilePositionsNegativeStaggerCoversEdges(t *testing.T) {
	// A negative stagger must still produce tiles covering both edges.
	// Canvas width 20, tile 5, pitch 6, stagger -4: shifted rows run
	// ..., -10, -4, 2, 8, 14, with the tile at 14 reaching the right edge.
	got := tilePositions(20, 10, 5, 5, 1, 0, -4)
	oddRow := got[4:] // first row has 4 tiles at y=0; odd row starts at index 4
	if len(oddRow) == 0 || oddRow[0].X > 0 {
		t.Fatalf("negative stagger left the left edge uncovered: %v", oddRow)
	}
	last := oddRow[len(oddRow)-1]
	if last.X+5 < 19 {
		t.Fatalf("negative stagger left the right edge uncovered: last tile %v", last)
	}
}

func TestFloorDiv(t *testing.T) {
	cases := []struct {
		n, d, want int
	}{
		{7, 3, 2}, {-7, 3, -3}, {7, -3, -3}, {-7, -3, 2}, {0, 3, 0}, {6, 3, 2},
	}
	for _, c := range cases {
		if got := floorDiv(c.n, c.d); got != c.want {
			t.Errorf("floorDiv(%d,%d) = %d, want %d", c.n, c.d, got, c.want)
		}
	}
}

func TestValidateOptions(t *testing.T) {
	good := DefaultWatermarkOptions()
	if err := good.Validate(); err != nil {
		t.Fatalf("default options should be valid, got %v", err)
	}

	bad := []WatermarkOptions{
		{Opacity: -0.1, Scale: 1},
		{Opacity: 1.1, Scale: 1},
		{Opacity: nan(), Scale: 1},
		{Opacity: 0.5, Scale: 0},
		{Opacity: 0.5, Scale: -1},
		{Opacity: 0.5, Scale: nan()},
		{Opacity: 0.5, Scale: 1, SpacingX: -1},
		{Opacity: 0.5, Scale: 1, SpacingY: -1},
	}
	for i, opts := range bad {
		if err := opts.Validate(); err == nil {
			t.Errorf("case %d: expected validation error for %+v", i, opts)
		}
	}
}

func TestScaleWatermarkZeroSize(t *testing.T) {
	wm := solidImage(10, 10, color.NRGBA{255, 0, 0, 255})

	// A tiny scale rounds the 10px side down to zero: must error, never
	// produce a zero-size watermark.
	if _, err := scaleWatermark(wm, 0.01); err == nil {
		t.Fatal("expected error when watermark scales to zero size")
	}

	// An image with degenerate bounds is rejected directly.
	empty := image.NewNRGBA(image.Rect(0, 0, 0, 0))
	if _, err := scaleWatermark(empty, 1.0); err == nil {
		t.Fatal("expected error for zero-dimension watermark")
	}

	// Scale 1.0 keeps native dimensions.
	got, err := scaleWatermark(wm, 1.0)
	if err != nil || got.Bounds().Dx() != 10 || got.Bounds().Dy() != 10 {
		t.Fatalf("scale 1.0 should preserve size, got %v err %v", got.Bounds(), err)
	}
}

func TestScaleWatermarkNonInteger(t *testing.T) {
	// 7x5 scaled by 1.5 -> 11x8 (rounded), aspect ratio preserved.
	wm := solidImage(7, 5, color.NRGBA{0, 0, 255, 255})
	got, err := scaleWatermark(wm, 1.5)
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds().Dx() != 11 || got.Bounds().Dy() != 8 {
		t.Fatalf("expected 11x8 after 1.5x scale, got %v", got.Bounds())
	}
}

func TestCheckCanvasSize(t *testing.T) {
	if err := checkCanvasSize(image.Rect(0, 0, 100, 100)); err != nil {
		t.Errorf("normal canvas rejected: %v", err)
	}
	if err := checkCanvasSize(image.Rect(0, 0, maxCanvasDimension+1, 10)); err == nil {
		t.Error("expected error for oversized width")
	}
	if err := checkCanvasSize(image.Rect(0, 0, 20000, 20000)); err == nil {
		t.Error("expected error for excessive pixel count")
	}
	if err := checkCanvasSize(image.Rect(0, 0, 0, 10)); err == nil {
		t.Error("expected error for zero-width canvas")
	}
}

func TestTiledOverlayGridPointsAndGap(t *testing.T) {
	// Green 20x20 source, red 4x4 opaque watermark, spacing 2 (pitch 6).
	src := solidImage(20, 20, color.NRGBA{0, 255, 0, 255})
	wm := solidImage(4, 4, color.NRGBA{255, 0, 0, 255})
	opts := WatermarkOptions{Opacity: 1.0, Scale: 1.0, Tile: true, SpacingX: 2, SpacingY: 2}

	out := applyWatermark(src, wm, opts).(*image.NRGBA)
	if out.Bounds() != src.Bounds() {
		t.Fatalf("output bounds %v, want %v (size must be unchanged)", out.Bounds(), src.Bounds())
	}

	// Tile anchor (0,0): corner pixel is red.
	if c := out.NRGBAAt(0, 0); c.R != 255 || c.G != 0 {
		t.Errorf("tile anchor pixel = %v, want red", c)
	}
	// Gap pixel at x=4 (between first and second tile columns) stays green.
	if c := out.NRGBAAt(4, 0); c.G != 255 || c.R != 0 {
		t.Errorf("gap pixel = %v, want untouched green", c)
	}
	// Second tile anchor at x=6.
	if c := out.NRGBAAt(6, 0); c.R != 255 {
		t.Errorf("second tile anchor = %v, want red", c)
	}
}

func TestTiledOverlayEdgeClipping(t *testing.T) {
	// 10x10 source, 4x4 tiles, no spacing: last column/row anchors at 8, so
	// the tiles hang 2px off the edge and must be clipped, not dropped.
	src := solidImage(10, 10, color.NRGBA{0, 0, 0, 255})
	wm := solidImage(4, 4, color.NRGBA{255, 255, 255, 255})
	opts := WatermarkOptions{Opacity: 1.0, Tile: true}

	out := applyWatermark(src, wm, opts).(*image.NRGBA)
	// Bottom-right corner belongs to the clipped tile at (8,8).
	if c := out.NRGBAAt(9, 9); c.R != 255 {
		t.Errorf("corner pixel = %v, want white from clipped tile", c)
	}
	// The clipped tile must not write beyond the canvas.
	if out.Bounds().Dx() != 10 || out.Bounds().Dy() != 10 {
		t.Fatalf("canvas grew to %v, clipping must keep the original size", out.Bounds())
	}
}

func TestTiledOverlayAlphaBlending(t *testing.T) {
	// Opaque green source, semi-transparent red watermark (alpha 128),
	// opacity 0.5. Replicate imaging.Overlay's blend formula.
	src := solidImage(12, 12, color.NRGBA{0, 255, 0, 255})
	wm := alphaImage(4, 4, color.NRGBA{R: 255, G: 0, B: 0}, func(x, y int) uint8 { return 128 })
	opts := WatermarkOptions{Opacity: 0.5, Tile: true, SpacingX: 4, SpacingY: 4}
	out := applyWatermark(src, wm, opts).(*image.NRGBA)

	c := out.NRGBAAt(0, 0)
	coef2 := 0.5 * 128.0 / 255.0
	coef1 := (1 - coef2) // source is opaque: a1/255 == 1
	wantR := uint8(255 * coef2 / (coef1 + coef2))
	wantG := uint8(255 * coef1 / (coef1 + coef2))
	if abs(int(c.R)-int(wantR)) > 1 || abs(int(c.G)-int(wantG)) > 1 {
		t.Errorf("blended pixel = %v, want ~R%d G%d", c, wantR, wantG)
	}
	if c.A != 255 {
		t.Errorf("output alpha = %d, want 255 over opaque source", c.A)
	}

	// Gap pixel is untouched: exact source green.
	if g := out.NRGBAAt(4, 0); g != (color.NRGBA{0, 255, 0, 255}) {
		t.Errorf("gap pixel %v, want untouched source", g)
	}
}

func TestTiledOverlayNonIntegerCanvas(t *testing.T) {
	// Canvas 13x17 is not a whole multiple of the 4x4 tile pitch. The last
	// row/column of tiles must still cover the far edges.
	src := solidImage(13, 17, color.NRGBA{0, 0, 0, 255})
	wm := solidImage(4, 4, color.NRGBA{255, 255, 255, 255})
	opts := WatermarkOptions{Opacity: 1.0, Tile: true}
	out := applyWatermark(src, wm, opts).(*image.NRGBA)

	if c := out.NRGBAAt(12, 0); c.R != 255 {
		t.Errorf("right edge not covered: %v", c)
	}
	if c := out.NRGBAAt(0, 16); c.R != 255 {
		t.Errorf("bottom edge not covered: %v", c)
	}
	if c := out.NRGBAAt(12, 16); c.R != 255 {
		t.Errorf("bottom-right corner not covered: %v", c)
	}
}

func TestNonTiledKeepsHistoricalCenterOverlay(t *testing.T) {
	// With Tile off (the default), behavior must match imaging.OverlayCenter
	// and the output keeps the source dimensions and orientation.
	src := solidImage(21, 11, color.NRGBA{10, 20, 30, 255})
	wm := solidImage(5, 3, color.NRGBA{200, 100, 50, 255})
	opts := DefaultWatermarkOptions()
	opts.Opacity = 0.25

	got := applyWatermark(src, wm, opts).(*image.NRGBA)
	want := imaging.OverlayCenter(src, wm, 0.25)
	if got.Bounds() != want.Bounds() {
		t.Fatalf("bounds differ: %v vs %v", got.Bounds(), want.Bounds())
	}
	for y := 0; y < 11; y++ {
		for x := 0; x < 21; x++ {
			if got.NRGBAAt(x, y) != want.NRGBAAt(x, y) {
				t.Fatalf("pixel (%d,%d) = %v, want historical %v", x, y, got.NRGBAAt(x, y), want.NRGBAAt(x, y))
			}
		}
	}
}

func TestStaggerOffsetsAlternatingRows(t *testing.T) {
	src := solidImage(20, 20, color.NRGBA{0, 0, 0, 255})
	wm := solidImage(4, 4, color.NRGBA{255, 255, 255, 255})
	opts := WatermarkOptions{Opacity: 1.0, Tile: true, SpacingX: 4, SpacingY: 4, Stagger: 2}
	out := applyWatermark(src, wm, opts).(*image.NRGBA)

	// Row 0 tiles start at x=0; row 1 (y=8) is shifted +2, so x=0 there is
	// background and x=2 is watermark.
	if c := out.NRGBAAt(0, 8); c.R != 0 {
		t.Errorf("staggered row should not have a tile at x=0, got %v", c)
	}
	if c := out.NRGBAAt(2, 8); c.R != 255 {
		t.Errorf("staggered row tile missing at x=2, got %v", c)
	}
}

func TestTransparentSourcePreservesAlpha(t *testing.T) {
	// Over a fully transparent source, a half-opaque watermark keeps partial
	// alpha in the output instead of flattening to opaque.
	src := solidImage(8, 8, color.NRGBA{0, 0, 0, 0})
	wm := alphaImage(4, 4, color.NRGBA{R: 255}, func(x, y int) uint8 { return 255 })
	opts := WatermarkOptions{Opacity: 0.5, Tile: true}
	out := applyWatermark(src, wm, opts).(*image.NRGBA)
	if c := out.NRGBAAt(0, 0); c.A == 0 || c.A == 255 {
		t.Errorf("expected partial alpha over transparent source, got %v", c)
	}
}

func assertPoints(t *testing.T, got, want []image.Point) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d positions %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("position %d = %v, want %v (full got: %v)", i, got[i], want[i], got)
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
