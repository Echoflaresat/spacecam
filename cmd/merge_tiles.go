package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/echoflaresat/spacecam/render"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Fprintf(os.Stderr, "Usage: %s <cols>x<rows> <output.png> <tile1> <tile2> ...\n", os.Args[0])
		os.Exit(1)
	}

	// Parse layout
	tileParts := strings.Split(os.Args[1], "x")
	if len(tileParts) != 2 {
		log.Fatalf("Invalid tile format: %s (expected NxM)", os.Args[1])
	}
	cols, err := strconv.Atoi(tileParts[0])
	if err != nil {
		log.Fatalf("Invalid cols: %v", err)
	}
	rows, err := strconv.Atoi(tileParts[1])
	if err != nil {
		log.Fatalf("Invalid rows: %v", err)
	}

	output := os.Args[2]
	inputFiles := os.Args[3:]
	if len(inputFiles) != cols*rows {
		log.Fatalf("Expected %d input files, got %d", cols*rows, len(inputFiles))
	}

	var canvas *image.NRGBA
	var tileW, tileH int
	// Draw each tile into its position
	for idx, path := range inputFiles {
		fmt.Printf("Processing %s\n", path)
		inFile, err := os.Open(path)
		if err != nil {
			log.Fatalf("Could not load first input file %q: %v", path, err)
		}

		tile, err := render.LoadImage(inFile)
		if err != nil {
			log.Fatalf("Could not load first input file %q: %v", path, err)
		}

		if canvas == nil {
			tileW = tile.Bounds().Dx()
			tileH = tile.Bounds().Dy()
			canvas = image.NewNRGBA(image.Rect(0, 0, cols*tileW, rows*tileH))
		} else if tileW != tile.Bounds().Dx() || tileH != tile.Bounds().Dy() {
			log.Fatalf("Tile size mismatch for %q: expected %dx%d, got %dx%d",
				path, tileW, tileH, tile.Bounds().Dx(), tile.Bounds().Dy())
		}

		col := idx % cols
		row := idx / cols
		x := col * tileW
		y := row * tileH
		draw.Draw(canvas, image.Rect(x, y, x+tileW, y+tileH), tile, image.Point{0, 0}, draw.Over)
		inFile.Close()
	}

	save(output, canvas)
}

func save(output string, canvas *image.NRGBA) {
	fmt.Printf("-> creating %s\n", output)
	outFile, err := os.Create(output)
	if err != nil {
		log.Fatalf("Could not create %s: %v", output, err)
	}
	defer outFile.Close()

	ext := strings.ToLower(filepath.Ext(output))
	switch ext {
	case ".png":
		if err := png.Encode(outFile, canvas); err != nil {
			log.Fatalf("Failed to encode PNG: %v", err)
		}
	case ".jpg", ".jpeg":
		opts := jpeg.Options{Quality: 95}
		if err := jpeg.Encode(outFile, canvas, &opts); err != nil {
			log.Fatalf("Failed to encode JPEG: %v", err)
		}
	default:
		log.Fatalf("Unsupported output format: %s", ext)
	}

}
