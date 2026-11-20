package cmd

import (
	"math"

	"github.com/pkg/errors"
)

// Default watermark parameters used by both the "image" and "bulk" commands.
const (
	defaultOpacity  = 0.25
	defaultScale    = 1.0
	defaultSpacingX = 0
	defaultSpacingY = 0
	defaultStagger  = 0

	// maxCanvasDimension bounds the width or height of any image that is
	// watermarked, and maxCanvasPixels bounds its total pixel count. They keep
	// pathological inputs (for example a decoded image claiming an astronomic
	// size) from triggering unbounded memory allocations.
	maxCanvasDimension = 100000
	maxCanvasPixels    = 100000000
)

// WatermarkOptions holds every watermark parameter. Both the single-image and
// bulk commands build this exact same struct, so the two code paths always
// behave identically for a given set of flags.
type WatermarkOptions struct {
	// Opacity of the watermark layer, between 0.0 and 1.0.
	Opacity float64
	// Scale resizes the watermark relative to its native size. A value of 1.0
	// keeps the native size; 0.5 halves it, 2.0 doubles it. The watermark's
	// aspect ratio is preserved.
	Scale float64
	// Tile switches on repeated ("tiled") watermarking. When false the
	// watermark is overlaid once in the center of the image, matching the
	// historical behavior.
	Tile bool
	// SpacingX and SpacingY are the gaps in pixels between adjacent tiled
	// watermarks, horizontally and vertically. Must be non-negative.
	SpacingX int
	SpacingY int
	// Stagger is the horizontal offset (in pixels) applied to every other
	// watermark row, producing a brick-like pattern. Negative values shift
	// the alternating rows to the left.
	Stagger int
}

// DefaultWatermarkOptions returns the options used when no flags are provided.
// Tile is false, preserving the original single center overlay.
func DefaultWatermarkOptions() WatermarkOptions {
	return WatermarkOptions{
		Opacity:  defaultOpacity,
		Scale:    defaultScale,
		Tile:     false,
		SpacingX: defaultSpacingX,
		SpacingY: defaultSpacingY,
		Stagger:  defaultStagger,
	}
}

// Validate checks the option values that can be verified without decoding any
// image. Image-dependent checks (zero-size watermark after scaling, oversized
// canvas) are enforced while the images are being processed.
func (o WatermarkOptions) Validate() error {
	if math.IsNaN(o.Opacity) || math.IsInf(o.Opacity, 0) || o.Opacity < 0.0 || o.Opacity > 1.0 {
		return errors.Errorf("invalid opacity %v: opacity must be between 0.0 and 1.0", o.Opacity)
	}
	if math.IsNaN(o.Scale) || math.IsInf(o.Scale, 0) || o.Scale <= 0.0 {
		return errors.Errorf("invalid scale %v: scale must be greater than 0", o.Scale)
	}
	if o.SpacingX < 0 {
		return errors.Errorf("invalid horizontal spacing %d: spacing must not be negative", o.SpacingX)
	}
	if o.SpacingY < 0 {
		return errors.Errorf("invalid vertical spacing %d: spacing must not be negative", o.SpacingY)
	}
	return nil
}
