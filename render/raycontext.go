package render

import (
	"math"

	"github.com/echoflaresat/spacecam/base"
	"github.com/echoflaresat/spacecam/earth"
)

// RayContext carries per-ray state and constants needed by the shader.
type RayContext struct {
	Origin            base.Vec3
	SunDir            base.Vec3
	AltitudeKm        float64
	RayDirection      base.Vec3
	DistToCenter      float64
	T                 float64
	HitPoint          base.Vec3
	SurfaceNormal     base.Vec3
	RimLightFactor    float64
	SunLightIntensity float64
	ViewDotNormal     float64
	theme             Theme
}

// NewRayContext mirrors your Python constructor.
func NewRayContext(
	origin base.Vec3,
	sunDir base.Vec3,
	altitudeKm float64,
	theme Theme,
) RayContext {
	return RayContext{
		Origin:     origin,
		SunDir:     sunDir,
		AltitudeKm: altitudeKm,
		theme:      theme,
	}
}

// SetRayDirection updates the per-ray fields like in your Python set_ray_direction().
func (c *RayContext) SetRayDirection(rayDirection base.Vec3) {
	c.RayDirection = rayDirection

	// Closest approach of the ray to the origin (Earth center).
	dotOriginRay := c.Origin.Dot(c.RayDirection)
	closestPointToCenter := c.Origin.Sub(c.RayDirection.Scale(dotOriginRay))
	c.DistToCenter = closestPointToCenter.Norm()

	// Ray–sphere intersection with Earth (spherical).
	c.T = intersectSphere(c.Origin, c.RayDirection, earth.Radius)

	// Hit point and surface normal (normalize even if T<0, to mirror Python behavior).
	c.HitPoint = c.Origin.Add(c.RayDirection.Scale(c.T))
	c.SurfaceNormal = c.HitPoint.Normalize()

	// Rim light factor = cosine between sunDir and normalized closest vector.
	if c.DistToCenter > 0 {
		c.RimLightFactor = closestPointToCenter.Scale(1.0 / c.DistToCenter).Dot(c.SunDir)
	} else {
		c.RimLightFactor = 0.0
	}

	// Lighting cosines used by the shader.
	c.SunLightIntensity = c.SurfaceNormal.Dot(c.SunDir)
	c.ViewDotNormal = -c.SurfaceNormal.Dot(c.RayDirection)
}

// intersectSphere computes the parametric distance t along ray O + t*D to the first
// intersection with a sphere of radius r. Returns -1 if there is no hit.
// Matches your quadratic form: b = 2*(O·D), c = O·O - r^2.
func intersectSphere(O, D base.Vec3, r float64) float64 {
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
