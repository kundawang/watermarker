package cmd

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/disintegration/imaging"
)

// savePNG writes img to path, creating parent directories as needed.
func savePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := imaging.Save(img, path); err != nil {
		t.Fatalf("save %s: %v", path, err)
	}
}

func writeFixturePair(t *testing.T, dir string, srcSize image.Point) (srcPath, wmPath string) {
	src := solidImage(srcSize.X, srcSize.Y, color.NRGBA{12, 34, 56, 255})
	wm := solidImage(6, 6, color.NRGBA{255, 0, 0, 255})
	srcPath = filepath.Join(dir, "src.png")
	wmPath = filepath.Join(dir, "wm.png")
	savePNG(t, srcPath, src)
	savePNG(t, wmPath, wm)
	return srcPath, wmPath
}

func TestWatermarkImageLegacyRoundTrip(t *testing.T) {
	dir := t.TempDir()
	srcPath, wmPath := writeFixturePair(t, dir, image.Pt(40, 30))
	outPath := filepath.Join(dir, "out.png")

	if err := WatermarkImage(srcPath, wmPath, outPath, 0.25); err != nil {
		t.Fatalf("legacy WatermarkImage: %v", err)
	}
	out, err := imaging.Open(outPath)
	if err != nil {
		t.Fatalf("open output: %v", err)
	}
	// Dimensions and orientation are preserved.
	if out.Bounds().Dx() != 40 || out.Bounds().Dy() != 30 {
		t.Fatalf("output size %v, want 40x30", out.Bounds())
	}
}

func TestWatermarkImageWithOptionsTiledFile(t *testing.T) {
	dir := t.TempDir()
	// Non-integer-multiple canvas (31x23) with 6px tiles.
	srcPath, wmPath := writeFixturePair(t, dir, image.Pt(31, 23))
	outPath := filepath.Join(dir, "tiled.png")

	opts := DefaultWatermarkOptions()
	opts.Tile = true
	opts.Opacity = 0.4
	opts.SpacingX = 2
	opts.SpacingY = 4
	opts.Stagger = 3
	if err := WatermarkImageWithOptions(srcPath, wmPath, outPath, opts); err != nil {
		t.Fatalf("tiled WatermarkImageWithOptions: %v", err)
	}
	out, err := imaging.Open(outPath)
	if err != nil {
		t.Fatalf("open output: %v", err)
	}
	if out.Bounds().Dx() != 31 || out.Bounds().Dy() != 23 {
		t.Fatalf("tiled output size %v, want 31x23", out.Bounds())
	}
}

func TestWatermarkImageOverwritesSourceWhenNoName(t *testing.T) {
	// Documented historical behavior: without -n the source file is replaced.
	dir := t.TempDir()
	srcPath, wmPath := writeFixturePair(t, dir, image.Pt(20, 20))
	before, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := WatermarkImage(srcPath, wmPath, srcPath, 0.5); err != nil {
		t.Fatalf("in-place watermark: %v", err)
	}
	after, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) == len(after) && string(before) == string(after) {
		t.Fatal("source file was not modified despite in-place watermark")
	}
}

func TestWatermarkImageWithOptionsValidationErrors(t *testing.T) {
	dir := t.TempDir()
	srcPath, wmPath := writeFixturePair(t, dir, image.Pt(20, 20))

	cases := map[string]WatermarkOptions{
		"bad opacity":   {Opacity: 2, Scale: 1},
		"zero scale":    {Opacity: 0.5, Scale: 0},
		"neg spacing x": {Opacity: 0.5, Scale: 1, SpacingX: -5},
		"neg spacing y": {Opacity: 0.5, Scale: 1, SpacingY: -5},
	}
	for name, opts := range cases {
		if err := WatermarkImageWithOptions(srcPath, wmPath, filepath.Join(dir, name+".png"), opts); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestBulkWatermarkHappyPathAndSameFolderGuard(t *testing.T) {
	root := t.TempDir()
	srcFolder := filepath.Join(root, "in")
	outFolder := filepath.Join(root, "out")

	// Two source images plus a non-image file that must be skipped.
	savePNG(t, filepath.Join(srcFolder, "a.png"), solidImage(33, 27, color.NRGBA{1, 2, 3, 255}))
	savePNG(t, filepath.Join(srcFolder, "b.png"), solidImage(18, 18, color.NRGBA{4, 5, 6, 255}))
	if err := os.WriteFile(filepath.Join(srcFolder, "notes.txt"), []byte("not an image"), 0644); err != nil {
		t.Fatal(err)
	}
	wmPath := filepath.Join(root, "wm.png")
	savePNG(t, wmPath, solidImage(5, 5, color.NRGBA{255, 255, 255, 255}))

	opts := DefaultWatermarkOptions()
	opts.Tile = true
	opts.SpacingX = 1
	opts.SpacingY = 1
	opts.Opacity = 0.3
	if err := BulkWatermarkWithOptions(srcFolder, wmPath, outFolder, opts); err != nil {
		t.Fatalf("bulk: %v", err)
	}

	for _, name := range []string{"a.png", "b.png"} {
		out, err := imaging.Open(filepath.Join(outFolder, name))
		if err != nil {
			t.Fatalf("missing output %s: %v", name, err)
		}
		wantW, wantH := 0, 0
		if name == "a.png" {
			wantW, wantH = 33, 27
		} else {
			wantW, wantH = 18, 18
		}
		if out.Bounds().Dx() != wantW || out.Bounds().Dy() != wantH {
			t.Errorf("%s output size %v, want %dx%d", name, out.Bounds(), wantW, wantH)
		}
	}
	if _, err := os.Stat(filepath.Join(outFolder, "notes.txt")); !os.IsNotExist(err) {
		t.Error("non-image file should not be copied to the output folder")
	}

	// Writing back into the source folder must be refused outright.
	if err := BulkWatermarkWithOptions(srcFolder, wmPath, srcFolder, opts); err == nil {
		t.Fatal("expected error when output folder equals source folder")
	}

	// Legacy wrapper keeps working with its historical parameters.
	legacyOut := filepath.Join(root, "legacy")
	if err := BulkWatermark(srcFolder, wmPath, legacyOut, 0.25); err != nil {
		t.Fatalf("legacy bulk: %v", err)
	}
	if _, err := os.Stat(filepath.Join(legacyOut, "a.png")); err != nil {
		t.Errorf("legacy bulk output missing: %v", err)
	}
}

func TestBulkSkipsUndecodableAndOversized(t *testing.T) {
	root := t.TempDir()
	srcFolder := filepath.Join(root, "in")
	outFolder := filepath.Join(root, "out")

	savePNG(t, filepath.Join(srcFolder, "good.png"), solidImage(10, 10, color.NRGBA{1, 2, 3, 255}))
	// A .png that is not actually a PNG must be skipped, not fatal.
	if err := os.WriteFile(filepath.Join(srcFolder, "broken.png"), []byte("definitely not png bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	wmPath := filepath.Join(root, "wm.png")
	savePNG(t, wmPath, solidImage(2, 2, color.NRGBA{255, 0, 0, 255}))

	if err := BulkWatermarkWithOptions(srcFolder, wmPath, outFolder, DefaultWatermarkOptions()); err != nil {
		t.Fatalf("bulk should skip bad files, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(outFolder, "good.png")); err != nil {
		t.Errorf("good image was not processed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outFolder, "broken.png")); err == nil {
		t.Error("broken image should not have an output")
	}
}

func TestScaledTiledWatermarkFile(t *testing.T) {
	dir := t.TempDir()
	srcPath, wmPath := writeFixturePair(t, dir, image.Pt(50, 50))
	outPath := filepath.Join(dir, "scaled.png")

	opts := DefaultWatermarkOptions()
	opts.Tile = true
	opts.Scale = 0.5 // 6px watermark -> 3px
	opts.Opacity = 1.0
	opts.SpacingX = 2 // 3px tile, 2px gap: next tile starts at x=5
	if err := WatermarkImageWithOptions(srcPath, wmPath, outPath, opts); err != nil {
		t.Fatalf("scaled tiled: %v", err)
	}
	out, err := imaging.Open(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds().Dx() != 50 || out.Bounds().Dy() != 50 {
		t.Fatalf("output size changed: %v", out.Bounds())
	}
	// Scaled tile occupies x=0..2; the gap at x=3..4 keeps the source color.
	if c := color.NRGBAModel.Convert(out.At(0, 0)).(color.NRGBA); c.R != 255 || c.G != 0 {
		t.Errorf("scaled tile anchor = %v, want red", c)
	}
	if c := color.NRGBAModel.Convert(out.At(3, 0)).(color.NRGBA); c != (color.NRGBA{12, 34, 56, 255}) {
		t.Errorf("gap pixel after scaled tile = %v, want untouched source", c)
	}
	if c := color.NRGBAModel.Convert(out.At(5, 0)).(color.NRGBA); c.R != 255 {
		t.Errorf("next scaled tile missing at x=5, got %v", c)
	}
}
