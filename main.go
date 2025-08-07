package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"math"
	"os"
	"time"

	"github.com/echoflaresat/spacecam/colors"
	"github.com/echoflaresat/spacecam/earth"
	"github.com/echoflaresat/spacecam/render"
	"github.com/echoflaresat/spacecam/vectors"
)

type config struct {
	lat, lon, alt      *float64
	fov, tilt          *float64
	size, supersample  *int
	out                *string
	day, night, clouds *string
	timeStr            *string
	showHelp           *bool
}

func defineFlags() config {
	return config{
		lat:  flag.Float64("lat", 0.0, "Camera latitude in degrees"),
		lon:  flag.Float64("lon", 0.0, "Camera longitude in degrees"),
		alt:  flag.Float64("alt", 8880.0, "Camera altitude in kilometers"),
		fov:  flag.Float64("fov", 80.0, "Camera field of view in degrees"),
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

	printGroup("Camera Options", []string{"lat", "lon", "alt", "fov", "tilt"})
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

	// Parse time
	var renderTime time.Time
	var err error
	if *cfg.timeStr != "" {
		renderTime, err = time.Parse(time.RFC3339, *cfg.timeStr)
		if err != nil {
			log.Fatalf("Invalid time format: %v", err)
		}
	} else {
		renderTime = time.Now()
	}

	sunDir := earth.SunDirectionECEF(renderTime)
	phi := 90.0
	sunDir = vectors.Vec3{math.Cos(phi / 180.0 * math.Pi), math.Sin(phi / 180.0 * math.Pi), 0}
	// positive z: up
	//
	camera := render.NewCamera(*cfg.lat, *cfg.lon, *cfg.alt, *cfg.fov, *cfg.tilt)

	img, err := render.RenderScene(
		camera,
		sunDir,
		*cfg.size,
		*cfg.supersample,
		render.Theme{
			// DayRim:   colors.New(0.529, 0.808, 0.980, 0.5),
			DayRim:   colors.New(0.529, 0.808, 0.980, 0.5),
			NightRim: colors.New(0.1, 0.1, 0.1, 0.1),
			Warm:     colors.New(1.02, 1.0, 0.98, 1.0),
			Day:      *cfg.day,
			Night:    *cfg.night,
			Clouds:   *cfg.clouds,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := writePNG(*cfg.out, img); err != nil {
		log.Fatalf("Failed to write PNG: %v", err)
	}
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(f, img)
}
