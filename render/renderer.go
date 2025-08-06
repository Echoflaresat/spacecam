package render

import (
	"fmt"
	"image"
	"math"

	"github.com/echoflaresat/spacecam/colors"
	"github.com/echoflaresat/spacecam/earth"
	"github.com/echoflaresat/spacecam/vectors"
)

type Theme struct {
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
func RenderEarthSurface(ctx *RayContext) colors.Color4 {

	CDay := ctx.TexDay.Sample(ctx.HitPoint)
	CNight := ctx.TexNight.Sample(ctx.HitPoint)
	CClouds := ctx.TexClouds.Sample(ctx.HitPoint)

	// Compute how much sunlight is hitting the surface (soft transition)
	light := Smoothstep(-0.1, 0.1, ctx.SunLightIntensity)

	// 1. Blend day and night
	CBlended := BlendNightDayEnergyConserving(CDay, CNight, light)

	// 2. Blend clouds
	CBlended = BlendClouds(CBlended, CClouds, light, 2.0)

	// 3. Specular highlight (glint on oceans)
	CBlended = ApplySpecularHighlight(ctx, CBlended, CDay)

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

	if ctx.SunLightIntensity <= 0 {
		return Crgb
	}

	view := ctx.RayDirection.Scale(-1).Normalize()
	light := ctx.SunDir.Normalize()
	halfVec := view.Add(light).Normalize()

	specAngle := Clip(ctx.SurfaceNormal.Dot(halfVec), 0.0, 1.0)
	specular := math.Pow(specAngle, 30)
	oceanFactor := Clip((Cday.B-0.5*(Cday.R+Cday.G))*10.0, 0.0, 1.0)
	fresnel := math.Pow(1.0-ctx.ViewDotNormal, 2.0)

	reflectivity := oceanFactor

	strength := specular * reflectivity * (0.2 + 0.8*fresnel)

	sunColor := colors.Color4{R: 1.0, G: 0.97, B: 0.9, A: 1.0}
	return Crgb.Add(sunColor.Scale(strength))
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
	texDay, err := loadTexture(theme.Day)
	if err != nil {
		return nil, err
	}
	texNight, err := loadTexture(theme.Night)
	if err != nil {
		return nil, err
	}
	// Using day for clouds here, like your Python stub.
	texClouds, err := loadTexture(theme.Clouds)
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
		texDay,
		texNight,
		texClouds,
	)

	// Produce an RGB buffer (H*W*3)
	println("raytracescenepixels")
	img := RaytraceScenePixels(ctx, camera, outSize, supersampling)
	println("done")
	return img, nil
}

// ApplyAtmosphereOverlay simulates blue atmospheric scattering along the view ray.
// It increases near the horizon and scales with sunlight and depth.
// ApplyAtmosphereOverlay simulates atmospheric scattering along the view ray using ray tracing.
// It accounts for Rayleigh scattering, Earth's shadow, backlighting, and rays passing through thin air.
func ApplyAtmosphereOverlay(ctx *RayContext, base colors.Color4) colors.Color4 {
	const H = 25.0          // scale height (km)
	const maxHeight = 120.0 // max atmosphere extent (km)
	const rayleighStrength = 0.008

	atmoRadius := earth.Radius + maxHeight

	// Step 1: Ray-atmosphere intersection
	hitAtmo, tEntryAtmo, tExitAtmo := intersectSphereFull(ctx.Origin, ctx.RayDirection, atmoRadius)
	if !hitAtmo || tExitAtmo < 0 {
		return base
	}

	// Step 2: Ray-ground intersection
	hitEarth, tEntryEarth, _ := intersectSphereFull(ctx.Origin, ctx.RayDirection, earth.Radius)

	// Clip to visible atmosphere
	tMin := math.Max(0, tEntryAtmo)
	tMax := tExitAtmo
	if hitEarth && tEntryEarth > 0 && tEntryEarth < tMax {
		tMax = tEntryEarth
	}
	if tMax <= tMin {
		return base
	}

	// Step 3: Shadow intersection
	hitShadow, tShadowEntry, tShadowExit := IntersectShadowCylinder(ctx.Origin, ctx.RayDirection, ctx.SunDir, earth.Radius)

	// Step 4: Compute total lit length
	litLen := tMax - tMin
	if hitShadow {
		shadowStart := math.Max(tMin, tShadowEntry)
		shadowEnd := math.Min(tMax, tShadowExit)
		if shadowEnd > shadowStart {
			litLen -= (shadowEnd - shadowStart)
		}
	}
	if litLen <= 0 {
		return base
	}

	// Step 5: Estimate average altitude
	tMid := (tMin + tMax) * 0.5
	midPoint := ctx.Origin.Add(ctx.RayDirection.Scale(tMid))
	avgHeight := midPoint.Norm() - earth.Radius
	avgDensity := math.Exp(-avgHeight / H)

	amount := litLen * avgDensity * rayleighStrength
	amount = Clip(amount, 0.0, 1.0)

	return base.Mix(ctx.theme.DayRim, amount)
}

func IntersectShadowCylinder(
	rayOrigin, rayDir, sunDir vectors.Vec3,
	earthRadius float64,
) (bool, float64, float64) {
	V := sunDir.Normalize().Scale(-1) // Axis direction
	CO := rayOrigin

	// Vector from cylinder origin to ray origin

	// Project ray and offset onto plane perpendicular to V
	dDotV := rayDir.Dot(V)
	dPerp := rayDir.Sub(V.Scale(dDotV))

	coDotV := CO.Dot(V)
	coPerp := CO.Sub(V.Scale(coDotV))

	a := dPerp.Dot(dPerp)
	b := 2 * dPerp.Dot(coPerp)
	c := coPerp.Dot(coPerp) - earthRadius*earthRadius

	discriminant := b*b - 4*a*c
	if discriminant < 0 || a == 0 {
		return false, 0, 0
	}

	sqrtD := math.Sqrt(discriminant)
	t0 := (-b - sqrtD) / (2 * a)
	t1 := (-b + sqrtD) / (2 * a)

	if t1 < 0 {
		return false, 0, 0
	}

	M1 := rayOrigin.Add(rayDir.Scale(t0))
	if M1.Dot(V) < 0 {
		return false, 0, 0
	}

	entry := math.Max(0, t0)
	exit := t1
	return true, entry, exit
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// Update RaytraceScenePixels to apply tone mapping and adjusted saturation
func RaytraceScenePixels(ctx *RayContext, camera Camera, outSize, supersampling int) *image.NRGBA {
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

				c := colors.Black()
				if ctx.T > 0 {
					c = RenderEarthSurface(ctx)
				}

				c = ApplyAtmosphereOverlay(ctx, c)
				colorAccum = colorAccum.Add(c)
			}

			colorOut := colorAccum.Scale(1.0 / N)

			// Warmth
			colorOut = colorOut.Mul(ctx.theme.Warm)

			// Gentle saturation boost
			colorOut = colorOut.BoostSaturation(1.5)

			colorOut = colorOut.CompositeOverBlack()
			img.SetNRGBA(x, y, colorOut.ToNRGBA())
		}
	}

	fmt.Printf("100%% complete\n")
	return img
}
