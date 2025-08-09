package render

import (
	"math"

	"github.com/echoflaresat/spacecam/earth"
	"github.com/echoflaresat/spacecam/vectors"
)

// RayContext carries per-ray state and constants needed by the shader.
type RayContext struct {
	Origin            vectors.Vec3
	SunDir            vectors.Vec3
	GlobalSunFraction float64

	RayDir        vectors.Vec3
	TEarth        float64
	SurfaceNormal vectors.Vec3
	HitPoint      vectors.Vec3
	HitEarth      bool

	ViewDotNormal float64
	theme         Theme
	TexDay        Texture
	TexNight      Texture
	TexClouds     Texture

	HitsAtmosphere   bool
	AtmosphereEntryT float64
	AtmosphereExitT  float64

	HitsEarthShadow   bool
	EarthShadowEntryT float64
	EarthShadowExitT  float64
}

func NewRayContext(
	origin vectors.Vec3,
	sunDir vectors.Vec3,
	theme Theme,
	texDay Texture,
	texNight Texture,
	texClouds Texture,
) *RayContext {
	return &RayContext{
		Origin:    origin,
		SunDir:    sunDir,
		theme:     theme,
		TexDay:    texDay,
		TexNight:  texNight,
		TexClouds: texClouds,
	}
}

// SetRayDirection updates the per-ray fields like in your Python set_ray_direction().
// texElvation is used for bump mapping
func (c *RayContext) SetRayDirection(rayDirection vectors.Vec3) {
	c.RayDir = rayDirection

	// Step 1: Ray–sphere intersection with Earth
	hit, tEarth, _ := intersectSphereForward(c.Origin, c.RayDir, earth.Radius)
	c.HitEarth = hit
	c.TEarth = tEarth

	if c.HitEarth {
		c.HitPoint = c.Origin.Add(c.RayDir.Scale(c.TEarth))
		c.SurfaceNormal = c.HitPoint.Normalize()
		c.ViewDotNormal = -c.SurfaceNormal.Dot(c.RayDir)
	} else {
		c.HitPoint = vectors.Zero()
		c.SurfaceNormal = vectors.Zero()
		c.ViewDotNormal = 0.0
	}

	// Step 3: Atmosphere intersection
	hitsAtmo, tAtmosphereEntry, tAtmoshpereExit := intersectSphereForward(c.Origin, c.RayDir, earth.RadiusWithAtmosphere)
	c.HitsAtmosphere = hitsAtmo
	c.AtmosphereEntryT = tAtmosphereEntry
	c.AtmosphereExitT = tAtmoshpereExit
	if hitsAtmo && c.HitEarth {
		c.AtmosphereEntryT = math.Min(c.AtmosphereEntryT, c.TEarth)
		c.AtmosphereExitT = math.Min(c.AtmosphereExitT, c.TEarth)
	}

	// Step 3: Shadow intersection
	hitShadow, tShadowEntry, tShadowExit := intersectHalfCylinderForward(c.Origin, c.RayDir, c.SunDir.Scale(-1), earth.Radius)
	c.HitsEarthShadow = hitShadow
	c.EarthShadowEntryT = tShadowEntry
	c.EarthShadowExitT = tShadowExit
	if hitShadow && c.HitEarth {
		c.EarthShadowEntryT = math.Min(c.EarthShadowEntryT, c.TEarth)
		c.EarthShadowExitT = math.Min(c.EarthShadowExitT, c.TEarth)
	}

}

// intersectSphereForward checks whether a ray starting from `origin` in direction `dir`
// intersects a sphere of radius `radius` centered at the origin (0,0,0).
// Returns:
//   - bool: true if the ray hits the sphere in front of the origin
//   - float64: t0, the parametric distance to the first intersection point along the ray
//   - float64: t1, the parametric distance to the second intersection point along the ray
//
// If both intersections are behind the ray origin (t < 0), it returns false.
func intersectSphereForward(origin, dir vectors.Vec3, radius float64) (bool, float64, float64) {
	oc := origin
	a := dir.Dot(dir)
	b := 2.0 * oc.Dot(dir)
	c := oc.Dot(oc) - radius*radius

	discriminant := b*b - 4*a*c
	if discriminant < 0 {
		return false, 0, 0
	}
	sqrtD := math.Sqrt(discriminant)
	t0 := (-b - sqrtD) / (2 * a)
	t1 := (-b + sqrtD) / (2 * a)

	// if intersection is not in front of us, return false
	if t1 < 0 {
		return false, 0, 0
	}

	if t0 < 0 {
		t0 = 0.0
	}

	return true, t0, t1
}

// intersectHalfCylinderForward checks whether a ray starting from `origin` in direction `dir`
// intersects a **half-cylinder** of given `radius` whose axis runs along `axis` and passes
// through the origin (0,0,0).
//
// "Half-cylinder" here means only the region in front of the axis origin is considered;
// intersections behind the origin along the axis are ignored.
//
// Returns:
//   - bool: true if the ray hits the cylinder in front of the origin
//   - float64: t0, the distance along the ray to the first intersection point
//   - float64: t1, the distance along the ray to the second intersection point
func intersectHalfCylinderForward(
	origin, dir vectors.Vec3,
	axisDir vectors.Vec3, radius float64,
) (bool, float64, float64) {
	V := axisDir // Cylinder's axis direction (normalized)
	CO := origin // Vector from cylinder's center line to ray origin

	// Project ray direction onto plane perpendicular to axis
	dDotV := dir.Dot(V)              // Component of ray direction along axis
	dPerp := dir.Sub(V.Scale(dDotV)) // Perpendicular component

	// Project origin vector onto plane perpendicular to axis
	coDotV := CO.Dot(V)               // Component of origin along axis
	coPerp := CO.Sub(V.Scale(coDotV)) // Perpendicular component

	// Quadratic coefficients for intersection in 2D (perpendicular plane)
	a := dPerp.Dot(dPerp)
	b := 2 * dPerp.Dot(coPerp)
	c := coPerp.Dot(coPerp) - radius*radius

	// Discriminant test for real intersection
	discriminant := b*b - 4*a*c
	if discriminant < 0 || a == 0 {
		return false, 0, 0 // No hit or ray is parallel to cylinder surface
	}

	// Solve for intersection distances along the ray
	sqrtD := math.Sqrt(discriminant)
	t0 := (-b - sqrtD) / (2 * a) // First intersection
	t1 := (-b + sqrtD) / (2 * a) // Second intersection

	// Both intersections behind the origin → no forward hit
	if t1 < 0 {
		return false, 0, 0
	}

	// Check that the intersection is in the forward axial direction
	// (dot product with axis ensures we’re hitting the "front half" of the cylinder)
	M1 := origin.Add(dir.Scale(t0))
	if M1.Dot(V) < 0 {
		return false, 0, 0
	}

	// Clamp the first intersection to zero if behind the origin
	t0 = math.Max(0, t0)
	return true, t0, t1
}
