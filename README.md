# watermarker
Simple CLI tool to watermark and overlay images. Built using Go.

## Special Thanks
Watermarker was built using the following wonderful packages:

[Cobra from @spf13](https://github.com/spf13/cobra)

[Imaging from @disintegration](https://github.com/disintegration/imaging)

## Install

```shell
go get -u github.com/YoungMetroTrusted/watermarker
```

## Usage
### Get help
```shell
watermarker help
```
```shell
watermarker image -h
```
```shell
watermarker bulk -h
```

### Watermark a single image
Note: This will overwrite the original image
```shell
watermarker image examples/example_image.jpg example_watermark.png
```

### Watermark a single image with custom opacity
```shell
watermarker image examples/example_image.jpg example_watermark.png -o 0.35
```

### Watermark a single image with custom opacity and write to a new file
```shell
watermarker image examples/example_image.jpg example_watermark.png -o 0.35 -n new_image.jpg
```

### Watermark all images in a folder
Note: New images will be saved to a folder named "watermarked" in the current directory
```shell
watermarker bulk imageFolder/ example_watermark.png
```

### Watermark all images in a folder with custom opacity
```shell
watermarker bulk imageFolder/ example_watermark.png -o 0.8
```

### Watermark all images in a folder with custom opacity and custom output folder
```shell
watermarker bulk imageFolder/ example_watermark.png -o 0.8 -n examples/output/bulk/
```

## Tiled watermarks
A single watermark in the center (or corner) of an image can be removed by
cropping that region. Pass `--tile` (`-t`) to repeat the watermark in a grid
that covers the whole image instead. Tiles hanging over an edge are cropped to
the image boundary, and the output always keeps the original dimensions and
orientation.

The `image` and `bulk` commands accept exactly the same flags:

| Flag | Default | Meaning |
| --- | --- | --- |
| `--tile`, `-t` | `false` | Repeat the watermark across the image instead of overlaying it once in the center. |
| `--scale` | `1.0` | Scale factor for the watermark relative to its native size (e.g. `0.5` halves it, `2` doubles it). The aspect ratio is preserved. Must be greater than `0`. |
| `--opacity`, `-o` | `0.25` | Opacity of the watermark layer, a float between `0.0` and `1.0`. |
| `--spacing-x` | `0` | Horizontal gap in pixels between adjacent tiles. Must not be negative. |
| `--spacing-y` | `0` | Vertical gap in pixels between tile rows. Must not be negative. |
| `--stagger` | `0` | Horizontal offset in pixels applied to every other row for a brick-like pattern. Negative values shift the alternating rows to the left. |

### Tile a watermark over a single image
```shell
watermarker image examples/example_image.jpg example_watermark.png \
  --tile --scale 0.6 --opacity 0.35 --spacing-x 24 --spacing-y 16 --stagger 12 \
  -n tiled_image.jpg
```

### Tile a watermark over a whole folder
```shell
watermarker bulk imageFolder/ example_watermark.png \
  --tile --scale 0.6 --opacity 0.35 --spacing-x 24 --spacing-y 16 --stagger 12
```

Without `--tile` every flag other than `--opacity` and `--scale` is ignored and
the tool behaves exactly as before: one centered overlay at the native
watermark size (or the scaled size when `--scale` is given).

## Validation and error behavior
- An out-of-range but finite `--opacity` (outside `0.0`–`1.0`) prints a warning
  and falls back to the default `0.25`, matching the historical behavior.
  `NaN`/infinite values are rejected with an error.
- A non-positive `--scale` is rejected. A scale that shrinks the watermark to a
  zero-size image (for example `--scale 0.001` on a tiny mark) is also rejected
  with an explicit error rather than producing an empty overlay.
- Negative `--spacing-x` / `--spacing-y` values are rejected.
- Images above the supported canvas limits (100000px on a side or
  100,000,000 pixels) are rejected so a malformed image cannot trigger
  unbounded memory use. In bulk mode such files are skipped with a message.
- The `image` command still overwrites the source file when no `--name` is
  given (the documented historical behavior). The `bulk` command refuses to
  write into its own source folder, because output files reuse the input
  names and would silently destroy the originals.
