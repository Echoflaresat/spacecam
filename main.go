package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/echoflaresat/spacecam/colors"
	"github.com/echoflaresat/spacecam/earth"
	"github.com/echoflaresat/spacecam/render"
	"github.com/echoflaresat/spacecam/vectors"
)

type config struct {
	lat, lon, alt      *float64
	fov, tilt, yaw     *float64
	size, supersample  *int
	out                *string
	day, night, clouds *string
	timeStr            *string
	showHelp           *bool
	panoramic          *bool
}

func defineFlags() config {
	return config{
		lat:  flag.Float64("lat", 0.0, "Camera latitude in degrees"),
		lon:  flag.Float64("lon", 120.0, "Camera longitude in degrees"),
		alt:  flag.Float64("alt", 8880.0, "Camera altitude in kilometers"),
		fov:  flag.Float64("fov", 60.0, "Camera field of view in degrees"),
		yaw:  flag.Float64("yaw", 0.0, "Camera yaw in degrees"),
		tilt: flag.Float64("tilt", 0.0, "Camera tilt in degrees"),

		size:        flag.Int("size", 1024, "Output image size (width/height in pixels)"),
		supersample: flag.Int("supersample", 3, "Supersampling factor (higher is slower but smoother)"),
		timeStr:     flag.String("time", "", "Time in RFC3339 format (e.g., 2025-08-02T15:04:05Z); defaults to now"),

		out: flag.String("out", "earth_view.png", "Output PNG file path"),

		day:    flag.String("day", "assets/world.200408.jpg", "Day texture path"),
		night:  flag.String("night", "assets/night.jpg", "Night texture path"),
		clouds: flag.String("clouds", "assets/cloud.2001210.jpg", "Clouds texture path"),

		panoramic: flag.Bool("panoramic", true, "Render a 2x2 panoramic view (90° apart in latitude)"),

		showHelp: flag.Bool("h", false, "Show this help message"),
	}
}

func printHelp() {
	fmt.Fprintf(os.Stderr, `Earth Renderer - Satellite View Generator

Usage:
  %[1]s [options]

`, os.Args[0])

	printGroup("Camera Options", []string{"lat", "lon", "alt", "fov", "tilt", "yaw"})
	printGroup("Rendering Options", []string{"size", "supersample", "time", "panoramic"})
	printGroup("Assets", []string{"day", "night", "clouds"})
	printGroup("Output", []string{"out"})
	printGroup("Misc", []string{"h"})
}

func printGroup(title string, keys []string) {
	fmt.Fprintf(os.Stderr, "%s:\n", title)
	for _, name := range keys {
		if f := flag.Lookup(name); f != nil {
			fmt.Fprintf(os.Stderr, "  -%-10s %s (default %q)\n", f.Name, f.Usage, f.DefValue)
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
	print("Generating " + *cfg.out + " ")

	renderTime := parseTimeOrExit(*cfg.timeStr)

	theme := render.Theme{
		DaySky:   colors.New(0.25, 0.60, 1.00, 0.5),
		NightSky: colors.New(0.15, 0.07, 0.20, 0.5),
		Warm:     colors.New(1.02, 1.0, 0.98, 1.0),
		Day:      *cfg.day,
		Night:    *cfg.night,
		Clouds:   *cfg.clouds,
	}

	numWorkers := runtime.GOMAXPROCS(0)
	sunDir := earth.SunDirectionECEF(renderTime)

	var img image.Image
	var err error
	if *cfg.panoramic {
		img, err = renderPanoramic(cfg, sunDir, theme, numWorkers)

	} else {
		img, err = renderSingle(cfg, sunDir, theme, numWorkers)
	}

	if err != nil {
		log.Fatalf("Could not generate image; %w", err)
	}

	if err := writePNG(*cfg.out, img); err != nil {
		log.Fatalf("Failed to write PNG: %v", err)
	}
}

func renderSingle(cfg config, sunDir vectors.Vec3, theme render.Theme, numWorkers int) (image.Image, error) {
	camera := render.NewCamera(*cfg.lat, *cfg.lon, *cfg.alt, *cfg.fov, *cfg.tilt, *cfg.yaw)
	return render.RenderScene(
		camera,
		sunDir,
		*cfg.size,
		*cfg.supersample,
		theme,
		numWorkers,
	)
}

func renderPanoramic(cfg config, sunDir vectors.Vec3, theme render.Theme, numWorkers int) (image.Image, error) {
	canvasSize := *cfg.size
	if canvasSize%2 != 0 {
		log.Fatalf("size must be even for panoramic; got %d", canvasSize)
	}
	tileSize := canvasSize / 2

	// Longitutes offset by 0, 90, 180, 270°
	lons := []float64{
		*cfg.lon + 0,
		*cfg.lon + 90,
		*cfg.lon + 180,
		*cfg.lon + 270,
	}

	tiles := make([]image.Image, 4)
	for i, lon := range lons {
		camera := render.NewCamera(*cfg.lat, lon, *cfg.alt, *cfg.fov, *cfg.tilt, *cfg.yaw)
		img, err := render.RenderScene(
			camera,
			sunDir,
			tileSize,
			*cfg.supersample,
			theme,
			numWorkers,
		)
		if err != nil {
			return nil, err
		}
		tiles[i] = img
	}

	out := image.NewRGBA(image.Rect(0, 0, canvasSize, canvasSize))
	positions := []image.Point{
		{0, 0}, {tileSize, 0},
		{0, tileSize}, {tileSize, tileSize},
	}
	for i := 0; i < 4; i++ {
		dst := image.Rectangle{Min: positions[i], Max: positions[i].Add(image.Point{tileSize, tileSize})}
		draw.Draw(out, dst, tiles[i], image.Point{}, draw.Src)
	}
	return out, nil
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
