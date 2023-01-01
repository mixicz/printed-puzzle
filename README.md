# 3D printed jigsaw puzzle

Tools for creating 3D printed puzzle. Both colorprint (each color on separate layer) and multimaterial print are supported and even can be combined.

It is also published on printables.com: https://www.printables.com/model/355978-3d-printed-puzzle

Project consists of 2 main parts:
* image conversion tool written in Go (in directory `convert/`),
* OpenSCAD source to render puzzle model (in directory `scad/`).

# Image conversion tool

## Build
Right now, no prebuilt binaries are shipped, so manual build is required:
* prerequisities:
  * [recent Go distribution](https://go.dev/doc/install)
* steps: 
```
git clone https://github.com/mixicz/printed-puzzle.git
cd printed-puzzle/convert
go build convert.go
```

I have only tested linux build at this moment, but there are no dependencies that should prevent building it for other platforms.

## Usage
When run without parameters (or with `-help` parameter), the tool will print out basic usage information:
```
./convert 
Usage: convert [flags] <source image>
  -bezier-segments float
        how many segments should we use to interpolate bezier curves (larger number may significantly increase rendering time) (default 5)
  -help
        prints this help message
  -layer-colors int
        number of possible colors in single layer for MMU (default 1)
  -nozzle float
        nozzle diameter (used to determine level of details to keep) (default 0.4)
  -palette string
        palette file (filament colors)
  -size float
        physical dimension of resulting puzzle in milimeters (larger dimension, other will be computed to keep original image aspect ratio) (default 200)
  -test-patterns
        write out files with test patterns, named 'test-pattern#.png'
```

No flags are mandatory, but using `-palette <palette file>` with custom palette is highly recomended, as default palette is only basic white+CMYK colors.

The tool will write 2 files:
* `preview.png` – image with adjusted resolution and converted to required palette,
* `out.scad` – image converted to vectors for use in OpenSCAD

### Palette preparation
Palette file is simple text file, with each line containing RGB/RGBA color value in hex format (e.g. #ff0000 for red), followed by optional text description. Conversion tool will use all color from palette in order they are specified (first color will be bottom layer), so make sure your palette contain only colors you intend to use for printing. It is also good idea to order colors from lightest to darkest.

Palette should contain real filament colors (many manufacturers document their filament colors in RAL or Pantone which can be converted to RGB using online tools like https://rgb.to and similar).

Sample palette file is provided in `colors/pla.txt` file. It is good idea to use one palette file as database of all filaments and copy only required lines to palette files for individual images. You may also want to experiment with different filament colors for best result.

# Puzzle pieces rendering
Now you can move `out.scad` to same directory where `puzzle.scad` resides (or use symlink) and open `puzzle.scad` in OpenSCAD.

## Puzzle parameters
Some parameters like total dimension or colors per layer are specified in `out.scad` and those must not be modified directly. If you need to modify them, use apropriate command line parameters in conversion tool.

Parameters specified in `puzzle.scad` are freely customizable and documented by comments in source file. Most important ones are:
* `pieces` – defines amount of pieces in X and Y dimensions respectively,
* `spc` – spacing between puzzle pieces, you may need to adjust this according your printer, so you can easily separate the pieces after printing while not making them too loose,

Also note, it is possible to re-render same-shaped puzzle pieces later, providing following parameters have same value as original print: `pieces`, `lock_size`, `rand_alter`, `seed`. Those parameters are directly affecting puzzle pieces shape.

## Colorprint (single color per layer)
As long as your puzzle fits on print bed, it should be enough to just render and export the model.

## MMU print (multiple colors per layer)
For MMU, you need to add parameter with rendered color to `piece_set();` command (or simply comment it out and uncomment commands below). You need to render and export each color individually in this case, to be later joined together in slicer.

## Large puzzle
In case you want puzzle larger than would fit on your print bed, you can limit number of puzzle pieces using start and dim parameters, for example: `piece_set(start=[0, 5], dim=[5, 5]);`

# Slicing
Only requirement for slicing is to match `layerHeight` parameter in `puzzle.scad` (default 0.2mm). Otherwise use your preferred software and methods to prepare color prints, like [prusa-slicer](https://help.prusa3d.com/article/color-change_1687).

# Credits
Sample image from [Pepper & Carrot](https://www.peppercarrot.com/en/) by David Revoy, License https://creativecommons.org/licenses/by/4.0/
