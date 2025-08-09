package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/echoflaresat/spacecam/colors"
	"github.com/echoflaresat/spacecam/earth"
	"github.com/echoflaresat/spacecam/render"
)

type config struct {
	lat, lon, alt      *float64
	fov, tilt, yaw     *float64
	size, supersample  *int
	out                *string
	day, night, clouds *string
	timeStr            *string
	showHelp           *bool
}

func defineFlags() config {
	return config{
		lat:  flag.Float64("lat", 0.0, "Camera latitude in degrees"),
		lon:  flag.Float64("lon", 40.0, "Camera longitude in degrees"),
		alt:  flag.Float64("alt", 8800.0, "Camera altitude in kilometers"),
		fov:  flag.Float64("fov", 60.0, "Camera field of view in degrees"),
		yaw:  flag.Float64("yaw", 0.0, "Camera yaw in degrees"),
		tilt: flag.Float64("tilt", 0.0, "Camera tilt in degrees"),

		size:        flag.Int("size", 640, "Output image size (width/height in pixels)"),
		supersample: flag.Int("supersample", 3, "Supersampling factor (higher is slower but smoother)"),
		timeStr:     flag.String("time", "", "Time in RFC3339 format (e.g., 2025-08-02T15:04:05Z); defaults to now"),

		out: flag.String("out", "earth_view.png", "Output PNG file path"),

		day:    flag.String("day", "assets/world.200408.tif", "Day texture path"),
		night:  flag.String("night", "assets/night.tif", "Night texture path"),
		clouds: flag.String("clouds", "assets/cloud.2001210.tif", "Clouds texture path"),

		showHelp: flag.Bool("h", false, "Show this help message"),
	}
}

func printHelp() {
	fmt.Fprintf(os.Stderr, `Earth Renderer - Satellite View Generator

Usage:
  %[1]s [options]

`, os.Args[0])

	printGroup("Camera Options", []string{"lat", "lon", "alt", "fov", "tilt", "yaw"})
	printGroup("Rendering Options", []string{"size", "ss", "time"})
	printGroup("Assets", []string{"day", "night", "clouds"})
	printGroup("Output", []string{"out"})
	printGroup("Misc", []string{"h"})
}

func printGroup(title string, keys []string) {
	fmt.Fprintf(os.Stderr, "%s:\n", title)
	for _, name := range keys {
		if f := flag.Lookup(name); f != nil {
			fmt.Fprintf(os.Stderr, "  -%-8s %s (default %q)\n", f.Name, f.Usage, f.DefValue)
		}
	}
	fmt.Fprintln(os.Stderr)
}

func main() {

	cfg := defineFlags()
	flag.Usage = printHelp
	flag.Parse()

	if *cfg.showHelp {
		printHelp()
		return
	}

	renderTime := parseTimeOrExit(*cfg.timeStr)

	theme := render.Theme{
		DayRim:   colors.New(0.25, 0.60, 1.00, 1.0),
		NightRim: colors.New(0.05, 0.07, 0.20, 0.5),
		OuterRim: colors.New(0.6, 0.9, 1.2, 1.0),
		Warm:     colors.New(1.02, 1.0, 0.98, 1.0),
		Day:      *cfg.day,
		Night:    *cfg.night,
		Clouds:   *cfg.clouds,
	}

	testViews(theme)

	img, err := renderImage(*cfg.lat, *cfg.lon, *cfg.alt, *cfg.fov, *cfg.tilt, *cfg.yaw, *cfg.size, *cfg.supersample, renderTime, theme)
	if err != nil {
		log.Fatal(err)
	}

	if err := writePNG(*cfg.out, img); err != nil {
		log.Fatalf("Failed to write PNG: %v", err)
	}

}

func parseTimeOrExit(timeStr string) time.Time {
	if timeStr == "" {
		return time.Now()
	}
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		log.Fatalf("Invalid time format: %v", err)
	}
	return t
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(f, img)
}

// renderImage renders the Earth view and returns the image.
func renderImage(lat, lon, alt, fov, tilt, yaw float64, size, supersample int, renderTime time.Time, theme render.Theme) (image.Image, error) {
	sunDir := earth.SunDirectionECEF(renderTime)
	camera := render.NewCamera(lat, lon, alt, fov, tilt, yaw)

	return render.RenderScene(
		camera,
		sunDir,
		size,
		supersample,
		theme,
	)
}

func testViews(theme render.Theme) {
	fov := 60.0
	tilt := 0.0
	yaw := 0.0
	size := 640
	supersample := 3
	renderTime, _ := time.Parse(time.RFC3339, "2024-08-08T09:23:00Z")
	const outDir = "tests"

	// Clean + recreate output dir
	if err := os.RemoveAll(outDir); err != nil {
		panic(fmt.Errorf("failed to remove test_outputs: %w", err))
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		panic(fmt.Errorf("failed to create test_outputs: %w", err))
	}

	renderAndSave("tests/night.png", 0, 180, 8800, fov, tilt, yaw, size, supersample, renderTime, theme)
	renderAndSave("tests/full.png", 0, 0, 8800, fov, tilt, yaw, size, supersample, renderTime, theme)
	renderAndSave("tests/sunrise.png", 0, 240, 8800, fov, tilt, yaw, size, supersample, renderTime, theme)
}

func renderAndSave(output string, lat, lon, alt, fov, tilt, yaw float64, size, supersample int, renderTime time.Time, theme render.Theme) {
	img, err := renderImage(lat, lon, alt, 60.0, tilt, yaw, size, supersample, renderTime, theme)
	if err != nil {
		panic(fmt.Errorf("render failed at view %w", err))
	}

	if err := writePNG(output, img); err != nil {
		panic(fmt.Errorf("failed to write %s: %w", output, err))
	}
	if err := writePNG("earth_view.png", img); err != nil {
		panic(fmt.Errorf("failed to write %s: %w", "earth_view.png", err))
	}
	fmt.Printf("Wrote %s\n", output)
}

// testMultiView renders 9 views from different longitudes
func testMultiView(theme render.Theme) error {

	size := 640
	supersample := 3
	renderTime, _ := time.Parse(time.RFC3339, "2024-08-08T09:23:00Z")
	const outDir = "frames"

	// Clean + recreate output dir
	if err := os.RemoveAll(outDir); err != nil {
		return fmt.Errorf("failed to remove test_outputs: %w", err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create test_outputs: %w", err)
	}

	for i := 1; i < 100; i++ {

		lat := 0.0
		lon := float64(200 + 40)
		tilt := 0.0
		alt := 8800.0
		// lat := rand.Float64()*180 - 90
		// lon := rand.Float64()*360 - 180
		// tilt := rand.Float64()*90 - 45
		yaw := 0.0
		// alt := rand.Float64()*2000.0 + 1000.0

		img, err := renderImage(lat, lon, alt, 60.0, tilt, yaw, size, supersample, renderTime, theme)
		if err != nil {
			return fmt.Errorf("render failed at view %d: %w", i, err)
		}

		outPath := filepath.Join(outDir, fmt.Sprintf("test_view_%02d.png", i))
		if err := writePNG(outPath, img); err != nil {
			return fmt.Errorf("failed to write %s: %w", outPath, err)
		}
		if err := writePNG("earth_view.png", img); err != nil {
			return fmt.Errorf("failed to write %s: %w", outPath, err)
		}
		fmt.Printf("Wrote %s\n", outPath)
	}

	return nil
}
