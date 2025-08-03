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

	// Hit point and surface normal (normalize even if T<0, to mirror Python behavior).
	c.HitPoint = c.Origin.Add(c.RayDirection.Scale(c.T))
	c.SurfaceNormal = c.HitPoint.Normalize()

	// Lighting cosines used by the shader.
	c.SunLightIntensity = c.SurfaceNormal.Dot(c.SunDir)
	c.ViewDotNormal = -c.SurfaceNormal.Dot(c.RayDirection)
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
