package vectors

import "math"

// Vec3 is a simple 3D vector with float64 components.
type Vec3 struct {
	X, Y, Z float64
}

func Zero() Vec3 {
	return Vec3{X: 0.0, Y: 0.0, Z: 0.0}
}

// Add returns v + o.
func (v Vec3) Add(o Vec3) Vec3 {
	return Vec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z}
}

// Sub returns v - o.
func (v Vec3) Sub(o Vec3) Vec3 {
	return Vec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z}
}

// Scale returns v * s.
func (v Vec3) Scale(s float64) Vec3 {
	return Vec3{v.X * s, v.Y * s, v.Z * s}
}

// Dot returns the dot product v · o.
func (v Vec3) Dot(o Vec3) float64 {
	return v.X*o.X + v.Y*o.Y + v.Z*o.Z
}

// Norm returns the Euclidean length ||v||.
func (v Vec3) Norm() float64 {
	return math.Sqrt(v.Dot(v))
}

// Normalize returns the unit vector v / ||v||.
// If ||v|| == 0, it returns the zero vector (0,0,0).
func (v Vec3) Normalize() Vec3 {
	n := v.Norm()
	if n == 0 {
		return Vec3{}
	}
	inv := 1.0 / n
	return Vec3{v.X * inv, v.Y * inv, v.Z * inv}
}

// Cross returns the cross product v × o.
func (v Vec3) Cross(o Vec3) Vec3 {
	return Vec3{
		X: v.Y*o.Z - v.Z*o.Y,
		Y: v.Z*o.X - v.X*o.Z,
		Z: v.X*o.Y - v.Y*o.X,
	}
}

// Orthogonal returns a unit vector that's perpendicular to v.
func (v Vec3) Orthogonal() Vec3 {
	if math.Abs(v.X) < 0.9 {
		// cross with X axis
		return v.Cross(Vec3{1, 0, 0}).Normalize()
	}
	// otherwise, cross with Y axis
	return v.Cross(Vec3{0, 1, 0}).Normalize()
}

func Distance(v1, v2 Vec3) float64 {
	return v1.Sub(v2).Norm()
}
