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
	AltitudeKm        float64
	RayDirection      vectors.Vec3
	DistToCenter      float64
	T                 float64
	HitPoint          vectors.Vec3
	SurfaceNormal     vectors.Vec3
	RimLightFactor    float64
	SunLightIntensity float64
	ViewDotNormal     float64
	theme             Theme
	TexDay            Texture
	TexNight          Texture
	TexClouds         Texture

	InsideAtmosphere bool
	AtmosphereEntryT float64 // Ray parameter where it enters atmosphere
	AtmosphereExitT  float64 // Ray parameter where it exits atmosphere
}

func NewRayContext(
	origin vectors.Vec3,
	sunDir vectors.Vec3,
	altitudeKm float64,
	theme Theme,
	texDay Texture,
	texNight Texture,
	texClouds Texture,
) *RayContext {
	return &RayContext{
		Origin:     origin,
		SunDir:     sunDir,
		AltitudeKm: altitudeKm,
		theme:      theme,
		TexDay:     texDay,
		TexNight:   texNight,
		TexClouds:  texClouds,
	}
}

// SetRayDirection updates the per-ray fields like in your Python set_ray_direction().
// texElvation is used for bump mapping
func (c *RayContext) SetRayDirection(rayDirection vectors.Vec3) {
	c.RayDirection = rayDirection

	// Closest approach of the ray to the origin (Earth center).
	dotOriginRay := c.Origin.Dot(c.RayDirection)
	closestPointToCenter := c.Origin.Sub(c.RayDirection.Scale(dotOriginRay))
	c.DistToCenter = closestPointToCenter.Norm()

	// Rim light factor = cosine between sunDir and normalized closest vector.
	if c.DistToCenter > 0 {
		c.RimLightFactor = closestPointToCenter.Scale(1.0 / c.DistToCenter).Dot(c.SunDir)
	} else {
		c.RimLightFactor = 0.0
	}

	// Ray–sphere intersection with Earth (spherical).
	c.T = intersectSphere(c.Origin, c.RayDirection, earth.Radius)

	if c.T >= 0 {
		c.HitPoint = c.Origin.Add(c.RayDirection.Scale(c.T))
		c.SurfaceNormal = c.HitPoint.Normalize()
		c.SunLightIntensity = c.SurfaceNormal.Dot(c.SunDir)
		c.ViewDotNormal = -c.SurfaceNormal.Dot(c.RayDirection)
	} else {
		// No hit: use sun alignment along the view ray for atmospheric scattering
		c.HitPoint = vectors.Vec3{}
		c.SurfaceNormal = vectors.Vec3{}
		c.ViewDotNormal = 0.0
		c.SunLightIntensity = Clip(c.RayDirection.Dot(c.SunDir), 0.0, 1.0)
	}

	hitsAtmo, t0, t1 := intersectSphereFull(c.Origin, c.RayDirection, earth.RadiusWithAtmosphere)
	c.InsideAtmosphere = hitsAtmo
	if hitsAtmo {
		// Clamp exit point to before planet surface
		if c.T > 0 && t1 > c.T {
			t1 = c.T
		}
		c.AtmosphereEntryT = math.Max(t0, 0.0)
		c.AtmosphereExitT = t1
	} else {
		c.AtmosphereEntryT = 0.0
		c.AtmosphereExitT = 0.0
	}

}

// IntersectEarth calculates the intersection of a ray (O + t*D) with a sphere of radius r.
// Returns the closest positive t, or -1.0 if there is no intersection.
func intersectSphere(O, D vectors.Vec3, r float64) float64 {
	// b = 2*O·D, c = O·O - r^2, solve t^2 + b t + c = 0
	OdotD := O.Dot(D)
	b := 2.0 * OdotD
	c := O.Dot(O) - r*r

	discriminant := b*b - 4.0*c
	if discriminant < 0 {
		return -1.0
	}

	sqrtDisc := math.Sqrt(discriminant)
	t1 := (-b - sqrtDisc) / 2.0
	t2 := (-b + sqrtDisc) / 2.0

	if t1 > 0 && t2 > 0 {
		if t1 < t2 {
			return t1
		}
		return t2
	}
	if t1 > 0 {
		return t1
	}
	if t2 > 0 {
		return t2
	}
	return -1.0
}

func intersectSphereFull(origin, dir vectors.Vec3, radius float64) (bool, float64, float64) {
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
	if t0 > t1 {
		t0, t1 = t1, t0
	}
	return true, t0, t1
}
