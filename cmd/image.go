package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var imageOpts = DefaultWatermarkOptions()
var outFileName string

// imageCmd represents the image command
var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "image is used to watermark a single image",
	Long: `The "image" command is used to watermark a single image. It takes
	two image files as its arguments, and will overlay the second image over
	the first image with the default opacity. The level of opacity can be specified with the
	"--opacity" or "-o" flag. By default, the first image file will be overwritten with
	the new "watermarked" image, to output to a new file use the "--name" or "-n" flag and
	provide a name for the new file.

	Pass "--tile" to repeat the watermark across the whole image. In tiled mode the
	"--scale", "--spacing-x", "--spacing-y" and "--stagger" flags control the grid.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if outFileName == "" {
			outFileName = args[0]
		}
		// Historical behavior: an out-of-range but finite opacity is rejected
		// softly and falls back to the default. NaN and infinity cannot be
		// represented by the flag parser's normal range, so they surface as
		// hard errors through WatermarkOptions.Validate.
		if imageOpts.Opacity > 1.0 || imageOpts.Opacity < 0.0 {
			cmd.Printf("Invalid opacity entered (opacity must be between 0.0 and 1.0), defaulting to %v ...\n", defaultOpacity)
			imageOpts.Opacity = defaultOpacity
		}
		if err := WatermarkImageWithOptions(args[0], args[1], outFileName, imageOpts); err != nil {
			log.Fatalf("Unable to watermark image, received the following error: %s", err.Error())
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(imageCmd)
	imageCmd.Flags().StringVarP(&outFileName, "name", "n", "", "Name can be used to output the watermarked image to a new file with the provided name.")
	addWatermarkFlags(imageCmd, &imageOpts)
}
