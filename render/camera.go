package render

import (
	"math"

	"github.com/echoflaresat/spacecam/earth"
	"github.com/echoflaresat/spacecam/vectors"
)

// Camera models a pinhole camera in ECEF coordinates.
type Camera struct {
	FOVDeg     float64
	TanHalfFOV float64
	Position   vectors.Vec3
	Forward    vectors.Vec3
	Right      vectors.Vec3
	Up         vectors.Vec3
}

// NewCamera constructs a camera from geodetic lat/lon (deg), altitude (km),
// field of view (deg), an additional tilt about the camera's Right axis (deg).
func NewCamera(latDeg, lonDeg, altKm, fovDeg, tiltDeg, yawDeg float64) Camera {
	lat := latDeg * math.Pi / 180.0
	lon := lonDeg * math.Pi / 180.0

	camRadius := earth.Radius + altKm
	x := camRadius * math.Cos(lat) * math.Cos(lon)
	y := camRadius * math.Cos(lat) * math.Sin(lon)
	z := camRadius * math.Sin(lat)

	pos := vectors.Vec3{X: x, Y: y, Z: z}

	// FOV
	fovRad := fovDeg * math.Pi / 180.0
	tanHalf := math.Tan(fovRad / 2.0)

	// Basis vectors
	fwd := pos.Normalize().Scale(-1.0) // look toward Earth center
	globalUp := vectors.Vec3{X: 0, Y: 0, Z: 1}
	right := fwd.Cross(globalUp)
	if right.Norm() < 1e-6 {
		right = vectors.Vec3{X: 1, Y: 0, Z: 0} // fallback if near poles / parallel
	}
	right = right.Normalize()
	up := right.Cross(fwd).Normalize()

	fwd, right, up = tiltCamera(fwd, right, up, 90)

	if yawDeg != 0 {
		fwd, right, up = yawCamera(fwd, right, up, yawDeg)
	}

	fwd, right, up = tiltCamera(fwd, right, up, -90)

	if tiltDeg != 0 {
		fwd, right, up = tiltCamera(fwd, right, up, tiltDeg)
	}
	return Camera{
		FOVDeg:     fovDeg,
		TanHalfFOV: tanHalf,
		Position:   pos,
		Forward:    fwd,
		Right:      right,
		Up:         up,
	}
}

// rotateVec applies Rodrigues’ rotation formula: rotate v around axis by (cosT, sinT).
func rotateVec(v, axis vectors.Vec3, cosT, sinT float64) vectors.Vec3 {
	// v*cos + (axis x v)*sin + axis*(axis·v)*(1-cos)
	return v.Scale(cosT).
		Add(axis.Cross(v).Scale(sinT)).
		Add(axis.Scale(axis.Dot(v) * (1.0 - cosT)))
}

// tiltCamera rotates forward/up around the Right axis by tiltDeg.
func tiltCamera(fwd, right, up vectors.Vec3, tiltDeg float64) (vectors.Vec3, vectors.Vec3, vectors.Vec3) {
	theta := tiltDeg * math.Pi / 180.0
	c, s := math.Cos(theta), math.Sin(theta)

	fwdNew := rotateVec(fwd, right, c, s).Normalize()
	upNew := rotateVec(up, right, c, s).Normalize()
	return fwdNew, right, upNew
}

// yawCamera rotates forward/right around the Up axis by yawDeg.
// This is a left-right (horizontal) camera pan.
func yawCamera(fwd, right, up vectors.Vec3, yawDeg float64) (vectors.Vec3, vectors.Vec3, vectors.Vec3) {
	theta := yawDeg * math.Pi / 180.0
	c, s := math.Cos(theta), math.Sin(theta)

	fwdNew := rotateVec(fwd, up, c, s).Normalize()
	rightNew := rotateVec(right, up, c, s).Normalize()
	return fwdNew, rightNew, up
}

// ComputeRay returns the normalized viewing direction for pixel (i,j)
// given the image dimensions (width,height). i,j can be fractional (for supersampling).
func (c Camera) ComputeRay(i, j float64, width, height int) vectors.Vec3 {
	w := float64(width)
	h := float64(height)

	// NDC in [-1, +1] (centered), flip Y to make +up in screen space.
	xNDC := (i - (w-1)/2.0) / ((w - 1) / 2.0)
	yNDC := -((j - (h-1)/2.0) / ((h - 1) / 2.0))

	xPlane := xNDC * c.TanHalfFOV
	yPlane := yNDC * c.TanHalfFOV
	zPlane := 1.0

	dir := c.Right.Scale(xPlane).
		Add(c.Up.Scale(yPlane)).
		Add(c.Forward.Scale(zPlane))

	return dir.Normalize()
}
