package render

import (
	"fmt"
	"image"
	"math"

	"github.com/echoflaresat/spacecam/colors"
	"github.com/echoflaresat/spacecam/earth"
	"github.com/echoflaresat/spacecam/texture"
	"github.com/echoflaresat/spacecam/vectors"
)

type Theme struct {
	SkyRim colors.Color4
	DayRim colors.Color4
	Warm   colors.Color4
	Day    string
	Night  string
	Clouds string
}

// Smoothstep performs a Hermite interpolation between 0 and 1 across [edge0, edge1].
// Returns 0 if x < edge0, 1 if x > edge1.
func Smoothstep(edge0, edge1, x float64) float64 {
	// Avoid division by zero
	if edge0 == edge1 {
		if x < edge0 {
			return 0.0
		}
		return 1.0
	}

	t := (x - edge0) / (edge1 - edge0)
	if t < 0.0 {
		t = 0.0
	} else if t > 1.0 {
		t = 1.0
	}
	return t * t * (3.0 - 2.0*t)
}

// Clip clamps x into the inclusive range [min, max].
func Clip(x, min, max float64) float64 {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}

// RenderSkyRimGlow renders a faint atmospheric halo when a ray misses Earth
// but grazes the atmosphere. Returns a linear RGBA color in [0,1].
func RenderSkyRimGlow(ctx *RayContext) colors.Color4 {
	// Fade in as the ray’s closest approach nears the Earth’s radius (plus ~200 km margin).
	edgeFade := Smoothstep(earth.Radius+200.0, earth.Radius, ctx.DistToCenter)
	if edgeFade <= 0 {
		return colors.Color4{}
	}

	// Day-side glow ramps with rim alignment to the sun.
	litGlow := Smoothstep(0.0, 0.5, ctx.RimLightFactor)

	// Night-side Earthshine glow (stronger the more shadowed).
	darkGlow := Smoothstep(-0.5, -0.2, -ctx.RimLightFactor)

	// Combine contributions.
	glowStrength := edgeFade * (0.9*litGlow + 0.3*darkGlow)

	// Slightly cooler tone (taken from ctx.ColorSkyRimGlow), scaled by strength.
	return ctx.theme.SkyRim.Scale(glowStrength)
}

// BlendNightDayEnergyConserving blends day and night colors using an
// energy-conserving root-sum-square method to ensure a smooth transition.
func BlendNightDayEnergyConserving(CDay, CNight colors.Color4, light float64) colors.Color4 {
	r := math.Sqrt((1-light)*CNight.R*CNight.R + light*CDay.R*CDay.R)
	g := math.Sqrt((1-light)*CNight.G*CNight.G + light*CDay.G*CDay.G)
	b := math.Sqrt((1-light)*CNight.B*CNight.B + light*CDay.B*CDay.B)
	return colors.Color4{R: r, G: g, B: b, A: 1.0}
}

// RenderEarthSurface renders the visible surface color at the intersection point.
// It blends day/night textures, clouds, specular, glow, and rim lighting.
func RenderEarthSurface(ctx *RayContext, CDay, CNight, CClouds colors.Color4) colors.Color4 {
	// Compute how much sunlight is hitting the surface (soft transition)
	light := Smoothstep(-0.1, 0.1, ctx.SunLightIntensity)

	// 1. Blend day and night
	CBlended := BlendNightDayEnergyConserving(CDay, CNight, light)

	// 2. Blend clouds
	CBlended = BlendClouds(CBlended, CClouds, light, 2.0)

	// 3. Specular highlight (glint on oceans)
	CBlended = ApplySpecularHighlight(ctx, CBlended, CDay)

	// 4. Atmospheric glow
	CBlended = ApplyGlow(ctx, CBlended, light)

	// 5. Day rim glow (soft limb highlight)
	CBlended = ApplyDayRimGlow(ctx, CBlended)

	return CBlended
}

// BlendClouds overlays cloud RGB texture onto the base surface color using inferred alpha.
// 'light' is the sunlight factor (0..1), 'boost' increases cloud visibility.
func BlendClouds(C, CCloud colors.Color4, light, boost float64) colors.Color4 {
	brightness := (CCloud.R + CCloud.G + CCloud.B) / 3.0
	cloudAlpha := brightness * light * boost

	r := C.R + (1.0-C.R)*CCloud.R*cloudAlpha
	g := C.G + (1.0-C.G)*CCloud.G*cloudAlpha
	b := C.B + (1.0-C.B)*CCloud.B*cloudAlpha
	a := C.A // preserve base alpha

	return colors.Color4{R: r, G: g, B: b, A: a}
}

// IsOcean returns true if the color is likely an ocean pixel,
// determined by whether blue is dominant relative to red and green.
func IsOcean(color colors.Color4, blueThreshold float64) bool {
	return (color.B > color.R*blueThreshold) && (color.B > color.G*blueThreshold)
}

// ApplySpecularHighlight adds a sun glint via a Blinn-Phong–style specular model.
// Returns the adjusted RGB color (alpha unchanged).
func ApplySpecularHighlight(ctx *RayContext, Crgb, Cday colors.Color4) colors.Color4 {
	// Only apply on ocean-like pixels.
	if !IsOcean(Cday, 1.05) {
		return Crgb
	}
	// Only when the surface is lit.
	if ctx.SunLightIntensity <= 0 {
		return Crgb
	}

	// Half-vector between view (−ray) and light; here using sunDir - rayDir then normalize.
	halfVec := ctx.SunDir.Sub(ctx.RayDirection).Normalize()

	// Cosine of angle between surface normal and half-vector.
	specAngle := ctx.SurfaceNormal.Dot(halfVec)
	if specAngle <= 0 {
		return Crgb
	}

	// Tight highlight.
	specular := math.Pow(specAngle, 20)

	// Modulate by ocean brightness (fake Fresnel/roughness).
	oceanReflectivity := Clip(Cday.B, 0.2, 1.0)

	strength := specular * oceanReflectivity * 0.8
	if strength > 0 {
		return Crgb.Add(colors.White().Scale(strength))
	}
	return Crgb
}

// ApplyGlow adds a soft atmospheric glow near the grazing angles.
// - light is typically Smoothstep(-0.1, 0.1, ctx.SunLightIntensity).
func ApplyGlow(ctx *RayContext, CBlended colors.Color4, light float64) colors.Color4 {
	// Grazing factor ~ how much the ray grazes the surface (clamped 0..1).
	grazing := 1.0 - (ctx.T / (ctx.AltitudeKm + earth.Radius))
	grazing = Clip(grazing, 0.0, 1.0)

	// Base glow strength scales with light and grazing^2.
	glow := light * (grazing * grazing)

	// Scale based on camera altitude (distance from surface).
	altRatio := Clip(ctx.AltitudeKm/10000.0, 0.0, 1.0)
	glow *= altRatio

	// Cooler blue bias as altitude increases.
	blueFactor := Clip((ctx.AltitudeKm-300.0)/1000.0, 0.0, 1.0)

	// Mix toward the sky rim glow color with altitude-dependent blue factor.
	r := CBlended.R*(1.0-glow) + ctx.theme.SkyRim.R*blueFactor*glow
	g := CBlended.G*(1.0-glow) + ctx.theme.SkyRim.G*blueFactor*glow
	b := CBlended.B*(1.0-glow) + ctx.theme.SkyRim.B*blueFactor*glow
	a := CBlended.A*(1.0-glow) + 0.5*blueFactor*glow

	return colors.Color4{R: r, G: g, B: b, A: a}
}

// GaussianFade returns a smooth Gaussian falloff centered at `center`
// with standard deviation `width`.
func GaussianFade(x, center, width float64) float64 {
	return math.Exp(-((x - center) * (x - center)) / (2.0 * width * width))
}

// Convenience version using your Python defaults: center=0.0, width=0.25.
func GaussianFadeDefault(x float64) float64 {
	return GaussianFade(x, 0.0, 0.25)
}

// ApplyDayRimGlow adds a soft atmospheric rim glow to the surface — including
// a subtle Earthshine component on the night side.
//
// Mirrors the Python version:
//
//	edge_alpha = gaussian_fade(ctx.view_dot_normal, center=0.0, width=0.50)
//	light_fade = smoothstep(-0.2, 0.1, ctx.sun_light_intensity)
//	shadow_fade = smoothstep(-0.7, -0.3, ctx.sun_light_intensity)
//	total_glow = edge_alpha*(0.3*light_fade) + edge_alpha*(0.15*shadow_fade)
//	if total_glow > 0: C + COLOR_DAY_RIM_GLOW*total_glow
func ApplyDayRimGlow(ctx *RayContext, CBlended colors.Color4) colors.Color4 {
	edgeAlpha := GaussianFade(ctx.ViewDotNormal, 0.0, 0.50) // fades at limb

	// Day-side glow
	lightFade := Smoothstep(-0.2, 0.1, ctx.SunLightIntensity)
	litStrength := edgeAlpha * lightFade * 0.3

	// Night-side Earthshine glow
	shadowFade := Smoothstep(-0.7, -0.3, ctx.SunLightIntensity) // fade in when in shadow
	darkStrength := edgeAlpha * shadowFade * 0.15               // dimmer than day-side

	totalGlow := litStrength + darkStrength
	if totalGlow > 0 {
		return CBlended.Add(ctx.theme.DayRim.Scale(totalGlow))
	}
	return CBlended
}

// GenerateSupersamplingOffsets returns n×n offsets in [-0.5, +0.5] for
// supersampling, as pairs (dx, dy) with pixel-center spacing.
func GenerateSupersamplingOffsets(n int) [][2]float64 {
	if n <= 0 {
		return nil
	}
	step := 1.0 / float64(n)
	out := make([][2]float64, 0, n*n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			dx := (float64(i)+0.5)*step - 0.5
			dy := (float64(j)+0.5)*step - 0.5
			out = append(out, [2]float64{dx, dy})
		}
	}
	return out
}

// RenderScene mirrors your Python function. It loads three textures,
// builds a RayContext, and raytraces the frame.
func RenderScene(
	camera Camera,
	sunDir vectors.Vec3,
	outSize int,
	supersampling int,
	theme Theme,
) (*image.NRGBA, error) {

	println("loading")
	texDay, err := texture.Load(theme.Day)
	if err != nil {
		return nil, err
	}
	texNight, err := texture.Load(theme.Night)
	if err != nil {
		return nil, err
	}
	// Using day for clouds here, like your Python stub.
	texClouds, err := texture.Load(theme.Clouds)
	if err != nil {
		return nil, err
	}

	origin := camera.Position
	altitudeKm := origin.Norm() - earth.Radius

	ctx := NewRayContext(
		origin,
		sunDir,
		altitudeKm,
		theme,
	)

	// Produce an RGB buffer (H*W*3)
	println("raytracescenepixels")
	img := RaytraceScenePixels(ctx, camera, outSize, supersampling, texDay, texNight, texClouds)
	println("done")
	return img, nil
}

// RaytraceScenePixels renders an outSize×outSize RGB frame using supersampling.
// Returns a packed RGB buffer (row-major): len = outSize*outSize*3.
func RaytraceScenePixels(ctx *RayContext, camera Camera, outSize, supersampling int,
	texDay, texNight, texClouds texture.Texture,
) *image.NRGBA {

	W, H := outSize, outSize
	offsets := GenerateSupersamplingOffsets(supersampling)
	N := float64(len(offsets))

	progressMilestone := 0

	img := image.NewNRGBA(image.Rect(0, 0, W, H))
	for y := 0; y < H; y++ {
		progress := (y * 100) / H
		if progress >= progressMilestone {
			fmt.Printf(" %3d%% ", progressMilestone)
			progressMilestone += 10
		}

		for x := 0; x < W; x++ {
			colorAccum := colors.Color4{}

			for _, off := range offsets {
				dx, dy := off[0], off[1]

				rayDir := camera.ComputeRay(float64(x)+dx, float64(y)+dy, W, H)
				ctx.SetRayDirection(rayDir)

				var c colors.Color4
				if ctx.T < 0 {
					// misses Earth → sky rim glow
					c = RenderSkyRimGlow(ctx)
				} else {
					// hit Earth → sample textures & shade surface
					cDay := texDay.Sample(ctx.HitPoint)
					cNight := texNight.Sample(ctx.HitPoint)
					cClouds := texClouds.Sample(ctx.HitPoint)
					c = RenderEarthSurface(ctx, cDay, cNight, cClouds)
				}
				colorAccum = colorAccum.Add(c)
			}

			// Average
			colorOut := colorAccum.Scale(1.0 / N)

			// Warmth in linear space
			colorOut = colorOut.Mul(ctx.theme.Warm)

			// Composite over black
			colorOut = colorOut.Scale(colorOut.A).Add(colors.Black())

			img.SetNRGBA(x, y, colorOut.ToNRGBA())
		}
	}

	fmt.Printf("100%% complete\n")
	return img
}
