package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"

	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"

	"golang.org/x/image/draw"

	gotrace "github.com/dennwc/gotrace"
	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/tools/bezier"
	"gonum.org/v1/plot/vg"
)

var (
	scale       = float64(1)
	bstep       = 0.2
	imageSize   = 200.0
	xSize       = 0.0
	ySize       = 0.0
	nozzleSize  = 0.4
	layerColors = 1
)

type ppColor struct {
	Chan [4]uint8
	Name string
}

type ppPalette []ppColor

func (p *ppPalette) toPalette() (pal color.Palette) {
	for _, c := range *p {
		pal = append(pal, color.RGBA{
			R: c.Chan[0],
			G: c.Chan[1],
			B: c.Chan[2],
			A: c.Chan[3],
		})
	}
	return
}

// func (p *ppPalette) findNearest(c color.Color) (idx int) {
// 	h, s := rgb2hc(c)
// 	var diff float64 = 99999
// 	for i, pp := range *p {
// 		dist := 3*math.Abs(h-pp.h) + math.Abs(s-pp.s)
// 		if dist < diff {
// 			idx = i
// 			diff = dist
// 		}
// 	}
// 	// }
// 	return
// }

// func distance(p, n gotrace.Point) float64 {
// 	dx := p.X - n.X
// 	dy := p.Y - n.Y
// 	return math.Sqrt(float64(dx*dx + dy*dy))
// }

// func abs(a int) int {
// 	if a < 0 {
// 		a = -a
// 	}
// 	return a
// }

// func rgb2hs(rr, gg, bb uint8) (h, s float64) {
// 	var huePrime float64
// 	r := float64(rr)
// 	g := float64(gg)
// 	b := float64(bb)
// 	max := math.Max(math.Max(r, g), b)
// 	min := math.Min(math.Min(r, g), b)
// 	chroma := (max - min)
// 	lvi := max

// 	if chroma == 0 {
// 		h = 0
// 	} else {
// 		if r == max {
// 			huePrime = math.Mod(((g - b) / chroma), 6)
// 		} else if g == max {
// 			huePrime = ((b - r) / chroma) + 2

// 		} else if b == max {
// 			huePrime = ((r - g) / chroma) + 4

// 		}

// 		h = huePrime * 60
// 	}
// 	if lvi == 0 {
// 		s = 0
// 	} else {
// 		s = (chroma / lvi)
// 	}
// 	if math.IsNaN(s) {
// 		s = 0
// 	}
// 	return
// }

// func rgb2hc(c color.Color) (float64, float64) {
// 	r, g, b, _ := c.RGBA()
// 	return rgb2hs(uint8(r/0x100), uint8(g/0x100), uint8(b/0x100))
// }

// Reads the palette from given file
// File format is simple text file with each line being HTML RGB(A) color representation followed by optional space separated filament name
// e.g. "#ffffff PLA White"
func readPalette(fileName string) (pal ppPalette, err error) {
	reader, e := os.Open(fileName)
	if e != nil {
		log.Fatal(e)
	}
	defer reader.Close()
	scanner := bufio.NewScanner(reader)
	r, _ := regexp.Compile(`^#?([0-9a-fA-F]{3,8})\s*(.*)$`)
	for scanner.Scan() {
		var c ppColor
		c.Chan[3] = 0xff // default alpha is opaque
		ok := true
		l := scanner.Text()
		m := r.FindStringSubmatch(l)

		// ignore empty (and non-matching) lines
		if len(m) >= 3 {
			for i := range c.Chan {
				if len(m[1]) >= i*2+2 {
					u, err := strconv.ParseUint(m[1][i*2:i*2+2], 16, 64)
					if err != nil {
						ok = false
						break
					}
					c.Chan[i] = uint8(u)
				}
			}
			c.Name = m[2]
			// c.h, c.s = rgb2hs(c.Chan[0], c.Chan[1], c.Chan[2])
			if ok {
				pal = append(pal, c)
			}
		}
	}
	if len(pal) <= 0 {
		err = errors.New("error parsing palette")
	}
	return
}

func mulSat(i uint32, m float32) uint8 {
	ti := float32(0xffff-i) * m
	switch {
	case ti > 0xffff:
		return 0
	case ti < 0:
		return 0xff
	default:
		return uint8((0xffff - ti) / 0x100)
	}
}

func contrast(c color.Color, m float32) (n color.RGBA) {
	r, g, b, a := c.RGBA()
	n.R = mulSat(r, m)
	n.G = mulSat(g, m)
	n.B = mulSat(b, m)
	n.A = uint8(a / 0x100)
	return
}

func channelFlag(i, bit int) uint8 {
	if i&1 == 0 {
		return 255
	}
	if i&(1<<(bit+1)) == 0 {
		return 255
	}
	return 0
}

func writeTestImage(fileName string) {
	writerPng, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer writerPng.Close()

	m := image.NewRGBA(image.Rect(0, 0, 512, 512))
	for i := 0; i < 16; i++ {
		b := image.Rectangle{
			Min: image.Point{
				X: i * 16,
				Y: i * 16,
			},
			Max: image.Point{
				X: 512 - i*16,
				Y: 512 - i*16,
			},
		}
		c := color.RGBA{channelFlag(i, 0), channelFlag(i, 1), channelFlag(i, 2), 255}
		log.Printf("color[%d]: %v", i, c)
		draw.Draw(m, b, &image.Uniform{c}, image.Point{}, draw.Src)
	}

	png.Encode(writerPng, m)
}

func writeTestImage2(fileName string) {
	writerPng, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer writerPng.Close()

	m := image.NewRGBA(image.Rect(-256, -256, 256, 256))
	b := m.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r := int(math.Sqrt(float64(x*x+y*y))) / 16
			c := color.RGBA{channelFlag(r, 0), channelFlag(r, 1), channelFlag(r, 2), 255}
			m.Set(x, y, c)
		}
	}

	png.Encode(writerPng, m)
}

func writeTestImage3(fileName string) {
	writerPng, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer writerPng.Close()

	m := image.NewRGBA(image.Rect(-256, -256, 256, 256))
	b := m.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r := int(math.Max(math.Abs(float64(x)), math.Abs(float64(y)))) / 16
			if x > 0 && y > 0 {
				r = int(math.Sqrt(float64(x*x+y*y))) / 16
			}
			c := color.RGBA{channelFlag(r, 0), channelFlag(r, 1), channelFlag(r, 2), 255}
			m.Set(x, y, c)
		}
	}

	png.Encode(writerPng, m)
}

type scadPoly struct {
	Pts  []gotrace.Point
	Path [][]int
}

type scadLayer struct {
	Poly  []scadPoly
	Color string
	Z     int
}

// const maxDist = 30

func curveToPoints(curve []gotrace.Segment, i *int) (pts []gotrace.Point, p0 []int) {
	i0 := *i
	for ci, c := range curve {
		switch c.Type {
		case gotrace.TypeCorner:
			p0 = append(p0, *i)
			pts = append(pts, gotrace.Point{X: c.Pnt[1].X * scale, Y: -c.Pnt[1].Y * scale})
			// ii := *i - i0
			// if ii > 0 && distance(pts[ii], pts[ii-1]) > maxDist {
			// 	log.Printf("WARN: Large segment point distance detected: %v <=> %v (%1.2f)", pts[ii-1], pts[ii], distance(pts[ii], pts[ii-1]))
			// }
			*i++
			p0 = append(p0, *i)
			pts = append(pts, gotrace.Point{X: c.Pnt[2].X * scale, Y: -c.Pnt[2].Y * scale})
			// ii = *i - i0
			// if ii > 0 && distance(pts[ii], pts[ii-1]) > maxDist {
			// 	log.Printf("WARN: Large segment point distance detected: %v <=> %v (%1.2f)", pts[ii-1], pts[ii], distance(pts[ii], pts[ii-1]))
			// }
			*i++

		case gotrace.TypeBezier:
			var vp [4]vg.Point
			if ci == 0 {
				vp[0].X = font.Length(curve[len(curve)-1].Pnt[2].X)
				vp[0].Y = font.Length(-curve[len(curve)-1].Pnt[2].Y)
			} else {
				vp[0].X = font.Length(curve[ci-1].Pnt[2].X)
				vp[0].Y = font.Length(-curve[ci-1].Pnt[2].Y)
			}
			for i := range c.Pnt {
				vp[i+1].X = font.Length(c.Pnt[i].X)
				vp[i+1].Y = font.Length(-c.Pnt[i].Y)
			}
			b := bezier.New(vp[0], vp[1], vp[2], vp[3])
			// step := 1 / distance(c.Pnt[0], c.Pnt[2])
			// if step > bstep {
			// 	step = bstep
			// }
			// fmt.Printf("bezier pts = (%d) %v, step = %f:", ci, vp, step)
			for t := 0.0; t < 1; t += bstep {
				pt := b.Point(t)
				p0 = append(p0, *i)
				pts = append(pts, gotrace.Point{X: float64(pt.X) * scale, Y: float64(pt.Y) * scale})
				// ii := *i - i0
				// if ii > 0 && distance(pts[ii], pts[ii-1]) > maxDist {
				// 	log.Printf("WARN: Large segment point distance detected: %v <=> %v (%1.2f)", pts[ii], pts[ii-1], distance(pts[ii], pts[ii-1]))
				// }
				// fmt.Printf(" [%1.2f,%1.2f],", p.Pts[i].X, p.Pts[i].Y)
				*i++
			}
			// fmt.Print("\n")
			p0 = append(p0, *i)
			pts = append(pts, gotrace.Point{X: c.Pnt[2].X * scale, Y: -c.Pnt[2].Y * scale})
			// ii := *i - i0
			// if ii > 0 && distance(pts[ii], pts[ii-1]) > maxDist {
			// 	log.Printf("WARN: Large segment point distance detected: %v <=> %v (%1.2f)", pts[ii], pts[ii-1], distance(pts[ii], pts[ii-1]))
			// }
			*i++
		}
	}
	p0 = append(p0, i0)
	return
}

func pathToPoly(path gotrace.Path) (poly []scadPoly) {
	if path.Sign < 0 {
		return
	}
	var p scadPoly
	i := 0
	pts, p0 := curveToPoints(path.Curve, &i)
	p.Pts = append(p.Pts, pts...)
	p.Path = append(p.Path, p0)

	for _, ch := range path.Childs {
		if ch.Sign < 0 {
			pts, pp := curveToPoints(ch.Curve, &i)
			p.Pts = append(p.Pts, pts...)
			p.Path = append(p.Path, pp)
			for _, sch := range ch.Childs {
				poly = append(poly, pathToPoly(sch)...)
			}
		} else {
			poly = append(poly, pathToPoly(ch)...)
		}
	}
	poly = append(poly, p)

	return
}

func pathToLayer(paths []gotrace.Path, c ppColor, z int) (layer scadLayer) {
	for _, p := range paths {
		layer.Poly = append(layer.Poly, pathToPoly(p)...)
	}
	layer.Color = fmt.Sprintf("#%02x%02x%02x", c.Chan[0], c.Chan[1], c.Chan[2])
	layer.Z = z
	return
}

func WriteScad(w io.Writer, layers []scadLayer) (err error) {
	fmt.Fprintf(w, "puzzle_mmu_colors = %d;\n", layerColors)
	fmt.Fprintf(w, "puzzle_dim = [%f, %f];\n", xSize, ySize)
	// array with all layers
	fmt.Fprint(w, "layers = [\n")
	for i, l := range layers {
		fmt.Fprintf(w, "  [  // layer #%d\n    [\n", i)
		// for each layer list of polygons
		for i, p := range l.Poly {
			fmt.Fprint(w, "      [\n        [ ")
			c := ", "
			if i == len(l.Poly)-1 {
				c = "" // last element without ","
			}
			// for each polygon list of points ...
			for i, pt := range p.Pts {
				c := ", "
				if i == len(p.Pts)-1 {
					c = "" // last element without ","
				}
				fmt.Fprintf(w, "[%f, %f]%s", pt.X, pt.Y, c)
			}
			fmt.Fprint(w, "],\n")
			fmt.Fprint(w, "        [ ")
			/// ... and list of paths
			for i, pa := range p.Path {
				fmt.Fprint(w, "[ ")
				for i, pap := range pa {
					c := ", "
					if i == len(pa)-1 {
						c = "" // last element without ","
					}
					fmt.Fprintf(w, "%d%s", pap, c)
				}
				c := ", "
				if i == len(p.Path)-1 {
					c = "" // last element without ","
				}
				fmt.Fprintf(w, "]%s", c)
			}
			fmt.Fprintf(w, "]\n      ]%s\n", c)
		}
		c := ", "
		if i == len(layers)-1 {
			c = "" // last element without ","
		}
		fmt.Fprintf(w, "    ], \"%s\", %d\n  ]%s\n", l.Color, l.Z, c)
	}
	fmt.Fprint(w, "];\n")
	return
}

func main() {
	// command line parameter processing
	// TODO preview file name, output file name, TurdSize, colors per layer
	var testPatterns, help bool
	var bseg float64
	flag.Float64Var(&imageSize, "size", 200, "physical dimension of resulting puzzle in milimeters (larger dimension, other will be computed to keep original image aspect ratio)")
	flag.Float64Var(&nozzleSize, "nozzle", 0.4, "nozzle diameter (used to determine level of details to keep)")
	flag.BoolVar(&testPatterns, "test-patterns", false, "write out files with test patterns, named 'test-pattern#.png'")
	flag.BoolVar(&help, "help", false, "prints this help message")
	flag.Float64Var(&bseg, "bezier-segments", 5, "how many segments should we use to interpolate bezier curves (larger number may significantly increase rendering time)")
	flag.IntVar(&layerColors, "layer-colors", 1, "number of possible colors in single layer for MMU")
	pfile := flag.String("palette", "", "palette file (filament colors)")
	flag.Parse()
	bstep = 1 / bseg

	if testPatterns {
		writeTestImage("test-pattern1.png")
		writeTestImage2("test-pattern2.png")
		writeTestImage3("test-pattern3.png")
		os.Exit(0)
	}

	if help || flag.NArg() != 1 {
		fmt.Println("Usage: convert [flags] <source image>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// load palette
	var ppal ppPalette
	if *pfile == "" {
		// default palette (CMYK colors)
		ppal = ppPalette{
			ppColor{Chan: [4]uint8{255, 255, 255, 255}, Name: "White"},
			ppColor{Chan: [4]uint8{0, 255, 255, 255}, Name: "Cyan"},
			ppColor{Chan: [4]uint8{255, 0, 255, 255}, Name: "Magenta"},
			ppColor{Chan: [4]uint8{255, 255, 0, 255}, Name: "Yellow"},
			ppColor{Chan: [4]uint8{0, 0, 0, 255}, Name: "Black"},
		}
	} else {
		pp, err := readPalette(*pfile)
		if err != nil {
			log.Fatal(err)
		}
		ppal = pp
	}
	pal := ppal.toPalette()

	// open source image and output files
	reader, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	writerScad, err := os.Create("out.scad")
	if err != nil {
		log.Fatal(err)
	}
	defer writerScad.Close()

	writerPng, err := os.Create("preview.png")
	if err != nil {
		log.Fatal(err)
	}
	defer writerPng.Close()

	// read source image
	img, _, err := image.Decode(reader)
	if err != nil {
		log.Fatal(err)
	}
	// resize if source image is too large (to avoid hard to print image features and potentialy speed up rendering)
	b := img.Bounds()
	bmax := math.Max(float64(b.Dx()), float64(b.Dy()))
	if imageSize/bmax < nozzleSize {
		downScale := bmax * nozzleSize / imageSize
		tmp := image.NewRGBA(image.Rect(0, 0, int(float64(b.Dx())/downScale), int(float64(b.Dy())/downScale)))
		draw.BiLinear.Scale(tmp, tmp.Rect, img, b, draw.Over, nil)
		img = tmp
		b = img.Bounds()
		bmax = math.Max(float64(b.Dx()), float64(b.Dy()))
	}

	// convert image to custom palette
	imgPal := image.NewPaletted(b, pal)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			imgPal.Set(x, y, contrast(img.At(x, y), 1.4))
		}
	}
	scale = imageSize / bmax
	xSize = scale * float64(b.Max.X)
	ySize = scale * float64(b.Max.Y)

	// write out the reduced color image for preview
	png.Encode(writerPng, imgPal)

	// convert image to vectors
	var par = gotrace.Params{
		TurdSize:     10,
		TurnPolicy:   gotrace.TurnMinority,
		AlphaMax:     1,
		OptiCurve:    true,
		OptTolerance: 0.2,
	}
	var layers []scadLayer
	// trace image color by color with respect to layers
	for l := 0; l < len(pal); l++ {
		bm := gotrace.NewBitmapFromImage(imgPal, func(x, y int, c color.Color) bool {
			i := pal.Index(c)
			return i%layerColors == l%layerColors && i >= l
		})
		paths, _ := gotrace.Trace(bm, &par)
		layers = append(layers, pathToLayer(paths, ppal[l], l/layerColors))
	}
	WriteScad(writerScad, layers)
}
