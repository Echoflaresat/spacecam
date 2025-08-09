package render

import (
	"fmt"
	"image"
	"math"

	"github.com/echoflaresat/spacecam/colors"
	"github.com/echoflaresat/spacecam/earth"
	"github.com/echoflaresat/spacecam/vectors"
)

var DayRim = colors.New(0.25, 0.60, 1.00, 1.0)
var NightRim = colors.New(0.05, 0.07, 0.20, 0.5)
var OuterRim = colors.New(0.6, 0.9, 1.2, 1.0)
var Warm = colors.New(1.02, 1.0, 0.98, 1.0)

type Theme struct {
	DayRim   colors.Color4
	NightRim colors.Color4
	OuterRim colors.Color4
	Warm     colors.Color4
	Day      string
	Night    string
	Clouds   string
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

func BlendNightDay(ctx *RayContext, CDay, CNight colors.Color4, light float64) colors.Color4 {
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
	sunLightIntensity := ctx.SurfaceNormal.Dot(ctx.SunDir)
	light := Smoothstep(-0.25, 0.05, sunLightIntensity)

	// 1. Blend day and night
	CBlended := BlendNightDay(ctx, CDay, CNight, light)

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

func ApplySpecularHighlight(ctx *RayContext, Crgb, Cday colors.Color4) colors.Color4 {

	if !IsOcean(Cday, 1.1) { // Only apply to ocean
		return Crgb
	}

	view := ctx.RayDir.Scale(-1).Normalize()
	N := ctx.SurfaceNormal
	L := ctx.SunDir.Scale(-1)
	R := Reflect(L, N).Normalize()

	specAngle := Clip(R.Dot(view), 0.0, 1.0)

	// Optional grazing angle falloff (to suppress edge blowouts)
	normalView := Clip(N.Dot(view), 0.0, 1.0)
	grazingFalloff := math.Pow(normalView, 3.0)

	exponent := 400.0 // Much sharper highlight
	strength := 1.0   // Can tweak this if it's too much

	specular := math.Pow(specAngle, exponent) * grazingFalloff * strength
	specular = Clip(specular, 0.0, 1.0)

	sunColor := colors.New(1.0, 0.97, 0.9, 1.0) // warm sun tint
	highlight := sunColor.Scale(specular)

	return Crgb.Add(highlight)
}

func ignore(x ...any) {

}

func Reflect(v, n vectors.Vec3) vectors.Vec3 {
	return v.Sub(n.Scale(2 * v.Dot(n)))
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

	ctx := NewRayContext(
		origin,
		sunDir,
		theme,
		texDay,
		texNight,
		texClouds,
	)

	// Produce an RGB buffer (H*W*3)
	img := RaytraceScenePixels(ctx, camera, outSize, supersampling)
	println("done")
	return img, nil
}

// ApplyAtmosphereOverlay simulates blue atmospheric scattering along the view ray.
// It increases near the horizon and scales with sunlight and depth.
// ApplyAtmosphereOverlay simulates atmospheric scattering along the view ray using ray tracing.
// It accounts for Rayleigh scattering, Earth's shadow, backlighting, and rays passing through thin air.
func ApplyAtmosphereOverlay(ctx *RayContext, base colors.Color4) colors.Color4 {
	const H = 55.0
	const rayleighStrength = 0.2

	if !ctx.HitsAtmosphere {
		return base
	}

	viewDot := ctx.ViewDotNormal
	rimAmount := math.Pow(1.0-viewDot, 3.0) * 0.3
	rimColor := ctx.theme.DayRim.Mix(ctx.theme.OuterRim, rimAmount)

	// Step 3: Shadow intersection
	hitShadow, tShadowEntry, tShadowExit := intersectHalfCylinderForward(ctx.Origin, ctx.RayDir, ctx.SunDir.Scale(-1), earth.Radius)

	litLen := ctx.AtmosphereExitT - ctx.AtmosphereEntryT
	unlitLen := 0.0
	if hitShadow {
		shadowStart := math.Max(ctx.AtmosphereEntryT, tShadowEntry)
		shadowEnd := math.Min(ctx.AtmosphereExitT, tShadowExit)

		if shadowEnd > shadowStart {
			shadowLen := shadowEnd - shadowStart

			if litLen > shadowLen && !ctx.HitEarth {
				// skip the area that is pure air but partially affected by shadow
				// without this we have an ugly V shape where the atmosphere fades into darkness
				// with darkess bitween the surface of the planet and the lit upper region
				// alphaCorr = math.Exp(-shadowLen)

			} else {
				litLen -= shadowLen
				unlitLen += shadowLen
			}
		}
	}

	if litLen <= 0 && unlitLen <= 0 {
		return base.Add(rimColor)
	}

	tMid := (ctx.AtmosphereExitT + ctx.AtmosphereEntryT) * 0.5
	midPoint := ctx.Origin.Add(ctx.RayDir.Scale(tMid))
	avgHeight := midPoint.Norm() - earth.Radius
	avgDensity := math.Exp(-avgHeight / H)

	// Light amount modulated by density
	litAmount := math.Log(litLen+unlitLen) * avgDensity * rayleighStrength
	litAmount = Clip(litAmount, 0.0, 1.0)

	if !ctx.HitEarth {
		litAmount = litAmount * 0.5
	}

	viewToSun := ctx.SunDir.Dot(ctx.RayDir) // [-1, 1]
	sunAngle := (1.0 - viewToSun) * 0.5     // 0 near sun, 1 opposite

	if !ctx.HitEarth && sunAngle > 0.5 {
		// this is supposed to fade out the atmosphere as it goes into the shadow
		litAmount *= Smoothstep(0.54, 0.5, sunAngle)
	}

	// Hue shift: warm when near sun, cool away
	skyColor := colors.New(
		Lerp(1.0, ctx.theme.DayRim.R, sunAngle), // R: white → neutral warm
		Lerp(1.0, ctx.theme.DayRim.G, sunAngle), // G: white → slightly greenish
		Lerp(1.0, ctx.theme.DayRim.B, sunAngle), // B: white → blueish
		ctx.theme.DayRim.A,
	)

	wNight := unlitLen / (litLen + unlitLen + 1e-5)
	tint := skyColor.Mix(ctx.theme.NightRim, wNight)
	out := base.Mix(tint, litAmount)

	return out

}

func Lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func RenderSunDisk(ctx *RayContext, base colors.Color4) colors.Color4 {

	if ctx.GlobalSunFraction == 0.0 {
		return base
	}

	// --- Add wide-angle forward scattering near sunrise ---
	sunViewAngle := math.Acos(ctx.SunDir.Dot(ctx.RayDir)) // radians
	horizonAngle := math.Acos(ctx.ViewDotNormal)          // radians

	sunInView := Smoothstep(0.3, 0.0, sunViewAngle) // 1 when near sun

	if sunInView > 0 {
		// 1 when sun near horizon
		sunNearHorizon := Smoothstep(-0.1, 0.1, math.Abs(horizonAngle-math.Pi/2))
		scatteringGlow := math.Pow(sunNearHorizon*sunInView*0.8, 3)
		glowColor := colors.New(1.0, 0.7, 0.4, 1.0) // warm orange
		base = base.Add(glowColor.Scale(scatteringGlow))
	}

	// Exit if Earth is in the way, this is intenionally after the sunInView
	// check so that we have a scattered orange effect in the dark
	// side of Earth upon sunrise
	if ctx.HitEarth {
		return base // Earth blocks the sun
	}

	rayDir, sunDir := ctx.RayDir, ctx.SunDir

	// Color of the sun (adjustable)
	sunColor := colors.New(1.0, 1.0, 1.0, 1.0)

	const sunAngularRadius = 0.0092 / 2            // radians
	const glowAngularRadius = sunAngularRadius * 5 // soft edge

	// Check if looking toward the sun
	cosTheta := rayDir.Dot(sunDir)
	if cosTheta <= 0 {
		return base // facing away
	}

	// Compute angular distance to sun center
	theta := math.Acos(cosTheta)
	if theta > glowAngularRadius {
		return base // too far off-center
	}

	// Compute intensity using smooth radial falloff
	t := Smoothstep(glowAngularRadius, sunAngularRadius, theta)
	sunIntensity := math.Pow(t, 2.0)

	// Blend the glow onto the base
	return base.Mix(sunColor, sunIntensity)
}

func SunVisibleFraction(camPos, sunDir vectors.Vec3) float64 {
	sunDistance := 149_597_870.7 // km
	sunRadius := 695_700.0       // km

	// Angular radii
	r := camPos.Norm()
	thetaE := math.Asin(earth.Radius / r)
	thetaS := math.Asin(sunRadius / sunDistance)

	// Angular separation
	cosAngle := camPos.Normalize().Dot(sunDir.Scale(-1))
	d := math.Acos(Clip(cosAngle, -1.0, 1.0)) // angle between Earth center and Sun center in radians

	// Convert angular radii to linear radii on unit sphere
	RE := thetaE
	RS := thetaS

	if d >= RE+RS {
		return 1.0 // Fully visible
	}
	if d <= math.Abs(RE-RS) {
		if RE > RS {
			return 0.0 // Fully blocked
		}
		return 1.0 // Sun entirely in front of Earth (unrealistic)
	}

	// Circle-circle overlap area on unit disk
	// (normalized to return fraction of the *sun's* area that is visible)
	part1 := RS * RS * math.Acos((d*d+RS*RS-RE*RE)/(2*d*RS))
	part2 := RE * RE * math.Acos((d*d+RE*RE-RS*RS)/(2*d*RE))
	part3 := 0.5 * math.Sqrt((-d+RS+RE)*(d+RS-RE)*(d-RS+RE)*(d+RS+RE))

	areaVisible := math.Pi*RS*RS - (part1 + part2 - part3)
	visibleFraction := Clip(areaVisible/(math.Pi*RS*RS), 0.0, 1.0)

	return visibleFraction
}

// Update RaytraceScenePixels to apply tone mapping and adjusted saturation
func RaytraceScenePixels(ctx *RayContext, camera Camera, outSize, supersampling int) *image.NRGBA {
	ar := 1.0 // 9.0 / 16.0
	W, H := outSize, int(float64(outSize)*ar)
	offsets := GenerateSupersamplingOffsets(supersampling)
	N := float64(len(offsets))

	progressMilestone := 0

	img := image.NewNRGBA(image.Rect(0, 0, W, H))

	ctx.GlobalSunFraction = SunVisibleFraction(camera.Position, ctx.SunDir)

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
				rayDir := camera.ComputeRay(float64(x)+dx, (float64(y)+dy)*ar, W, H)
				ctx.SetRayDirection(rayDir)

				hitEarth := ctx.TEarth > 0
				c := colors.Black()
				if hitEarth {
					// Earth is hit before Sun
					c = RenderEarthSurface(ctx)
				}

				c = ApplyAtmosphereOverlay(ctx, c)
				// Add solar disk and glow if visible
				c = RenderSunDisk(ctx, c)

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
