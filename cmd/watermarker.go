package cmd

import (
	"fmt"
	"image"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/disintegration/imaging"
)

// FileImage represents an image file
type FileImage struct {
	Name  string
	Image image.Image
}

// WatermarkImage overlays `watermarkFile` on top of `originalFile` with the specified `opacity` and saves
// the resulting image to `outputFile`.
//
// It is kept for backwards compatibility: it applies a single centered
// overlay at the native watermark size, exactly as before. New code should use
// WatermarkImageWithOptions which also supports tiled watermarks.
func WatermarkImage(originalFile, watermarkFile, outputFile string, opacity float64) error {
	opts := DefaultWatermarkOptions()
	opts.Opacity = opacity
	return WatermarkImageWithOptions(originalFile, watermarkFile, outputFile, opts)
}

// WatermarkImageWithOptions watermarks originalFile with watermarkFile using
// the full set of WatermarkOptions (tiled layout, scale, spacing, stagger) and
// saves the result to outputFile. The output image keeps the dimensions and
// orientation of the original.
func WatermarkImageWithOptions(originalFile, watermarkFile, outputFile string, opts WatermarkOptions) error {
	if err := opts.Validate(); err != nil {
		return err
	}

	// Open original image.
	src, err := imaging.Open(originalFile)
	if err != nil {
		return errors.Wrap(err, "failed to open original image")
	}
	if err := checkCanvasSize(src.Bounds()); err != nil {
		return err
	}

	// Open the image to use as the watermark
	watermark, err := imaging.Open(watermarkFile)
	if err != nil {
		return errors.Wrap(err, "failed to open watermark image")
	}
	watermark, err = scaleWatermark(watermark, opts.Scale)
	if err != nil {
		return err
	}

	marked := applyWatermark(src, watermark, opts)

	// Save the watermarked image
	if err := imaging.Save(marked, outputFile); err != nil {
		return errors.Wrap(err, "failed to save new watermarked image")
	}
	return nil
}

// BulkWatermark searches for image files in `srcFolder` and watermarks them all by overlaying each one with the image
// in `watermarkFile`, using the specified opacity. Watermarked images are then saved to `outputFolder`.
//
// It is kept for backwards compatibility and uses the historical parameters
// (single centered overlay, native watermark size).
func BulkWatermark(srcFolder, watermarkFile, outputFolder string, opacity float64) error {
	opts := DefaultWatermarkOptions()
	opts.Opacity = opacity
	return BulkWatermarkWithOptions(srcFolder, watermarkFile, outputFolder, opts)
}

// BulkWatermarkWithOptions applies the same WatermarkOptions to every image in
// srcFolder and writes the results to outputFolder. The watermark is opened and
// scaled once and reused for every image.
func BulkWatermarkWithOptions(srcFolder, watermarkFile, outputFolder string, opts WatermarkOptions) error {
	if err := opts.Validate(); err != nil {
		return err
	}

	// Refuse to write the watermarked images back into the source folder: each
	// output file uses the same name as its input, so this would silently
	// overwrite the originals.
	srcAbs, srcErr := filepath.Abs(filepath.Clean(srcFolder))
	outAbs, outErr := filepath.Abs(filepath.Clean(outputFolder))
	if srcErr == nil && outErr == nil && srcAbs == outAbs {
		return errors.Errorf("refusing to overwrite source images: output folder %q is the same as the source folder %q; choose a different --name", outputFolder, srcFolder)
	}

	// Open and scale the image to use as the watermark once, up front.
	watermark, err := imaging.Open(watermarkFile)
	if err != nil {
		return errors.Wrap(err, "failed to open watermark image")
	}
	watermark, err = scaleWatermark(watermark, opts.Scale)
	if err != nil {
		return err
	}

	// Get the image files from the srcFolder
	files, err := ioutil.ReadDir(srcFolder)
	if err != nil {
		return errors.Wrap(err, "failed to get files from source folder")
	}

	var images []FileImage
	for _, file := range files {
		img, err := imaging.Open(filepath.Join(srcFolder, file.Name()))
		if err != nil {
			fmt.Printf("Skipping file %s ...\n", file.Name())
			continue
		}
		if err := checkCanvasSize(img.Bounds()); err != nil {
			fmt.Printf("Skipping file %s: %s\n", file.Name(), err.Error())
			continue
		}
		fmt.Printf("Found image file %s\n", file.Name())
		images = append(images, FileImage{Name: file.Name(), Image: img})
	}

	if len(images) == 0 {
		fmt.Println("No images found in source folder. Did you forget to add a trailing '/' to the directory name?")
		return nil
	}

	// Convert the images
	fmt.Println("Watermarking images...")
	var watermarked []FileImage
	for _, img := range images {
		watermarked = append(watermarked, FileImage{Name: img.Name, Image: applyWatermark(img.Image, watermark, opts)})
	}

	// Save the images
	fmt.Println("Saving images...")
	if err := os.MkdirAll(outputFolder, 0755); err != nil {
		return errors.Wrap(err, "failed to create output folder")
	}
	for _, marked := range watermarked {
		if err := imaging.Save(marked.Image, filepath.Join(outputFolder, marked.Name)); err != nil {
			fmt.Printf("Failed to save watermarked image %s, received error %s\n", marked.Name, err.Error())
			continue
		}
	}

	return nil
}

// scaleWatermark returns a copy of watermark resized by scale relative to its
// native dimensions, preserving the aspect ratio. A scale of exactly 1.0 is a
// no-op so the default path stays pixel-identical to the historical behavior.
func scaleWatermark(watermark image.Image, scale float64) (*image.NRGBA, error) {
	b := watermark.Bounds()
	nativeW, nativeH := b.Dx(), b.Dy()
	if nativeW <= 0 || nativeH <= 0 {
		return nil, errors.New("watermark image has zero dimensions: nothing to overlay")
	}

	if scale == 1.0 {
		return imaging.Clone(watermark), nil
	}

	width := int(float64(nativeW)*scale + 0.5)
	height := int(float64(nativeH)*scale + 0.5)
	if width <= 0 || height <= 0 {
		return nil, errors.Errorf("watermark scaled to a zero-size image (scale %v applied to %dx%d)", scale, nativeW, nativeH)
	}
	return imaging.Resize(watermark, width, height, imaging.Lanczos), nil
}

// applyWatermark composites watermark onto src according to opts and returns
// the result. The returned image always has the same bounds as src. In tiled
// mode every watermark instance is first composited onto one full-canvas
// layer, which is then blended onto the source in a single pass.
func applyWatermark(src image.Image, watermark image.Image, opts WatermarkOptions) image.Image {
	if !opts.Tile {
		return imaging.OverlayCenter(src, watermark, opts.Opacity)
	}

	bounds := src.Bounds()
	positions := tilePositions(bounds.Dx(), bounds.Dy(),
		watermark.Bounds().Dx(), watermark.Bounds().Dy(),
		opts.SpacingX, opts.SpacingY, opts.Stagger)

	// Composite every tile onto one transparent full-canvas layer. draw.Draw
	// clips against the layer bounds, so tiles hanging off an edge are cropped
	// rather than dropped; the layer is then blended onto the source in a
	// single overlay pass.
	layer := buildTileLayer(bounds, watermark, positions)
	return imaging.Overlay(src, layer, bounds.Min, opts.Opacity)
}

// checkCanvasSize rejects images whose dimensions are too large to process
// safely.
func checkCanvasSize(bounds image.Rectangle) error {
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return errors.Errorf("image has zero or invalid dimensions %dx%d", w, h)
	}
	if w > maxCanvasDimension || h > maxCanvasDimension {
		return errors.Errorf("image dimensions %dx%d exceed the supported maximum of %d on either side", w, h, maxCanvasDimension)
	}
	if int64(w)*int64(h) > maxCanvasPixels {
		return errors.Errorf("image of %dx%d (%d pixels) exceeds the supported maximum of %d pixels", w, h, int64(w)*int64(h), maxCanvasPixels)
	}
	return nil
}
