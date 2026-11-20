package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// addWatermarkFlags registers the full set of watermark flags on a command and
// binds them to opts. Both the "image" and "bulk" commands call this so the
// two commands accept exactly the same parameters with identical defaults.
func addWatermarkFlags(cmd *cobra.Command, opts *WatermarkOptions) {
	cmd.Flags().Float64VarP(&opts.Opacity, "opacity", "o", defaultOpacity,
		fmt.Sprintf("Opacity of the watermark image, a float between 0.0 and 1.0 (default %v).", defaultOpacity))
	cmd.Flags().Float64Var(&opts.Scale, "scale", defaultScale,
		"Scale factor applied to the watermark before overlaying it, relative to its native size (e.g. 0.5 halves it, 2.0 doubles it). Must be greater than 0.")
	cmd.Flags().BoolVarP(&opts.Tile, "tile", "t", false,
		"Repeat the watermark in a tiled grid across the whole image instead of overlaying it once in the center.")
	cmd.Flags().IntVar(&opts.SpacingX, "spacing-x", defaultSpacingX,
		"Horizontal gap in pixels between adjacent tiled watermarks. Must not be negative.")
	cmd.Flags().IntVar(&opts.SpacingY, "spacing-y", defaultSpacingY,
		"Vertical gap in pixels between tiled watermark rows. Must not be negative.")
	cmd.Flags().IntVar(&opts.Stagger, "stagger", defaultStagger,
		"Horizontal offset in pixels applied to every other tile row for a brick-like pattern. Negative values shift the alternating rows left.")
}
