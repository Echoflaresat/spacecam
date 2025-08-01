package main

import (
	"image"
	"image/png"
	"log"
	"os"
	"time"

	"github.com/echoflaresat/spacecam/base"
	"github.com/echoflaresat/spacecam/earth"
	"github.com/echoflaresat/spacecam/render"
)

func main() {

	latDeg, lonDeg, altitudeKm := 47.0, 19.0, 8878.0
	fovDeg := 60.0
	tiltDeg := 0.0

	outputSize := 4096
	supersampling := 3
	sunDir := earth.SunDirectionECEF(time.Now())

	camera := render.NewCamera(latDeg, lonDeg, altitudeKm, fovDeg, tiltDeg)
	outputImage, err := render.RenderScene(
		camera,
		sunDir,
		outputSize,
		supersampling,
		render.Theme{
			SkyRim: base.NewColor(0.3, 0.55, 1.0, 1.0),
			DayRim: base.NewColor(0.3, 0.55, 1.0, 0.5),
			Warm:   base.NewColor(1.02, 1.0, 0.98, 1.0),
		},
	)

	if err != nil {
		log.Fatal(err)
	}
	writePNG("earth_view.png", outputImage)
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	// Using stdlib PNG encoder
	return (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(f, img)
}
