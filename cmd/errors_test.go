package cmd

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

func TestWatermarkImageWithOptionsMissingFiles(t *testing.T) {
	dir := t.TempDir()
	srcPath, wmPath := writeFixturePair(t, dir, image.Pt(16, 16))

	if err := WatermarkImageWithOptions(filepath.Join(dir, "missing.png"), wmPath,
		filepath.Join(dir, "o.png"), DefaultWatermarkOptions()); err == nil {
		t.Error("expected error when the original image is missing")
	}
	if err := WatermarkImageWithOptions(srcPath, filepath.Join(dir, "no-watermark.png"),
		filepath.Join(dir, "o.png"), DefaultWatermarkOptions()); err == nil {
		t.Error("expected error when the watermark image is missing")
	}
}

func TestWatermarkImageWithOptionsUndecodableWatermark(t *testing.T) {
	dir := t.TempDir()
	srcPath, _ := writeFixturePair(t, dir, image.Pt(16, 16))
	bad := filepath.Join(dir, "bad.png")
	if err := os.WriteFile(bad, []byte("not png"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := WatermarkImageWithOptions(srcPath, bad, filepath.Join(dir, "o.png"),
		DefaultWatermarkOptions()); err == nil {
		t.Error("expected error for an undecodable watermark")
	}
}

func TestBulkWatermarkMissingInputs(t *testing.T) {
	dir := t.TempDir()
	_, wmPath := writeFixturePair(t, dir, image.Pt(8, 8))

	// Missing source folder.
	if err := BulkWatermarkWithOptions(filepath.Join(dir, "nope"), wmPath,
		filepath.Join(dir, "out"), DefaultWatermarkOptions()); err == nil {
		t.Error("expected error when the source folder does not exist")
	}
	// Missing watermark.
	srcFolder := filepath.Join(dir, "in")
	savePNG(t, filepath.Join(srcFolder, "a.png"), solidImage(8, 8, color.NRGBA{1, 2, 3, 255}))
	if err := BulkWatermarkWithOptions(srcFolder, filepath.Join(dir, "missing-wm.png"),
		filepath.Join(dir, "out"), DefaultWatermarkOptions()); err == nil {
		t.Error("expected error when the watermark does not exist")
	}
}

func TestBulkWatermarkEmptyFolder(t *testing.T) {
	dir := t.TempDir()
	srcFolder := filepath.Join(dir, "empty")
	if err := os.MkdirAll(srcFolder, 0755); err != nil {
		t.Fatal(err)
	}
	wmPath := filepath.Join(dir, "wm.png")
	savePNG(t, wmPath, solidImage(2, 2, color.NRGBA{255, 0, 0, 255}))

	// An empty folder is not an error: it returns nil after printing a hint.
	if err := BulkWatermarkWithOptions(srcFolder, wmPath,
		filepath.Join(dir, "out"), DefaultWatermarkOptions()); err != nil {
		t.Errorf("empty source folder should return nil, got %v", err)
	}
}

func TestBulkWatermarkZeroScaleRejectedBeforeReadingFolder(t *testing.T) {
	dir := t.TempDir()
	srcFolder := filepath.Join(dir, "in")
	savePNG(t, filepath.Join(srcFolder, "a.png"), solidImage(8, 8, color.NRGBA{1, 2, 3, 255}))
	wmPath := filepath.Join(dir, "wm.png")
	savePNG(t, wmPath, solidImage(2, 2, color.NRGBA{255, 0, 0, 255}))

	opts := DefaultWatermarkOptions()
	opts.Scale = 0.01 // 2px mark rounds to zero
	if err := BulkWatermarkWithOptions(srcFolder, wmPath,
		filepath.Join(dir, "out"), opts); err == nil {
		t.Error("expected zero-size scaled watermark to be rejected")
	}
}
