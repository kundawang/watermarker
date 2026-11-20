package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var bulkOpts = DefaultWatermarkOptions()
var outputFolder string

// bulkCmd represents the bulk command
var bulkCmd = &cobra.Command{
	Use:   "bulk",
	Short: "Bulk is used to watermark all images in a folder",
	Long: `The "bulk" command is used to watermark multiple images at once. It takes
	two arguments, first a folder containing images to be watermarked, and second
	an image to apply as the watermark. The level of opacity for the watermark image
	can be overwritten with the "--opacity" or "-o" flag. By default, the watermarked
	images will be saved in a new folder named "watermarked" within the current directory,
	to override the "watermarked" folder name use the "--name" or "-n" flag and
	provide a name for the new folder.

	Pass "--tile" to repeat the watermark across every image. In tiled mode the
	"--scale", "--spacing-x", "--spacing-y" and "--stagger" flags control the grid.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if bulkOpts.Opacity > 1.0 || bulkOpts.Opacity < 0.0 {
			cmd.Printf("Invalid opacity entered (opacity must be between 0.0 and 1.0), defaulting to %v ...\n", defaultOpacity)
			bulkOpts.Opacity = defaultOpacity
		}
		if err := BulkWatermarkWithOptions(args[0], args[1], outputFolder, bulkOpts); err != nil {
			log.Fatalf("Unable to bulk watermark images in folder, received the following error: %s", err.Error())
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(bulkCmd)
	bulkCmd.Flags().StringVarP(&outputFolder, "name", "n", "watermarked/", "Name can be used to customize the name of the generated folder containing the new watermarked images.")
	addWatermarkFlags(bulkCmd, &bulkOpts)
}
