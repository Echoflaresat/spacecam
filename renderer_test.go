package main

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/echoflaresat/spacecam/earth"
	"github.com/echoflaresat/spacecam/render"
)

func TestViews(t *testing.T) {
	theme := render.Theme{
		DaySky:   render.DayRim,
		NightSky: render.NightRim,
		Warm:     render.Warm,
		Day:      "assets/world.200408.jpg",
		Night:    "assets/night.jpg",
		Clouds:   "assets/cloud.2001210.jpg",
	}

	fov := 60.0
	tilt := 0.0
	yaw := 0.0
	size := 640
	supersample := 3
	renderTime, err := time.Parse(time.RFC3339, "2024-08-08T09:23:00Z")
	if err != nil {
		t.Fatalf("failed to parse time: %v", err)
	}

	const outDir = "samples"
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("failed to create %s: %v", outDir, err)
	}

	cases := []struct {
		name string
		lat  float64
		lon  float64
		alt  float64
	}{
		{"60", 0, -60, 8800},
		{"night", 0, 180, 8800},
		{"full", 0, 0, 8800},
		{"sunrise", 0, 240, 8800},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			runGoldenImageTest(
				t,
				filepath.Join(outDir, c.name+".png"),
				func() (image.Image, error) {

					numWorkers := runtime.GOMAXPROCS(0)
					sunDir := earth.SunDirectionECEF(renderTime)
					camera := render.NewCamera(c.lat, c.lon, c.alt, fov, tilt, yaw)

					return render.RenderScene(
						camera,
						sunDir,
						size,
						supersample,
						theme,
						numWorkers,
					)
				},
			)
		})
	}
}

// runGoldenImageTest renders an image using renderFunc, compares it against the golden image at expectedPath,
// and fails if they differ. If the golden image doesn't exist, it is created and the test fails.
func runGoldenImageTest(t *testing.T, expectedPath string, renderFunc func() (image.Image, error)) {
	t.Helper()

	// Render new image
	img, err := renderFunc()
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	// If baseline doesn't exist, create it and fail
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		if err := writePNG(expectedPath, img); err != nil {
			t.Fatalf("failed to write baseline image: %v", err)
		}
		t.Fatalf("baseline image %s did not exist, created one", expectedPath)
	}

	// Load expected image
	expectedFile, err := os.Open(expectedPath)
	if err != nil {
		t.Fatalf("failed to open expected image: %v", err)
	}
	defer expectedFile.Close()

	expectedImg, err := png.Decode(expectedFile)
	if err != nil {
		t.Fatalf("failed to decode expected image: %v", err)
	}

	// Compare
	if !imagesEqual(expectedImg, img) {
		newPath := expectedPath
		if err := writePNG(newPath, img); err != nil {
			t.Fatalf("failed to write new differing image: %v", err)
		}
		t.Fatalf("image differs from baseline; saved new image to %s", newPath)
	}
}

func imagesEqual(a, b image.Image) bool {
	var bufA, bufB bytes.Buffer
	_ = png.Encode(&bufA, a)
	_ = png.Encode(&bufB, b)
	return bytes.Equal(bufA.Bytes(), bufB.Bytes())
}
