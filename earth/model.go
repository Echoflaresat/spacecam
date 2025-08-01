package earth

import (
	"time"

	"github.com/echoflaresat/spacecam/base"
	"github.com/soniakeys/meeus/v3/julian"
	"github.com/soniakeys/meeus/v3/sidereal"
	"github.com/soniakeys/meeus/v3/solar"
)

const Radius = 6371.0 // Earth radius in km (spherical approximation)

func SunDirectionECEF(t time.Time) base.Vec3 {
	t = t.UTC()
	jd := julian.TimeToJD(t)

	// Step 1: Apparent RA/Dec of the Sun (in radians)
	ra, dec := solar.ApparentEquatorial(jd)

	// Step 2: Unit vector in ECI (Earth-centered inertial)
	x := dec.Cos() * ra.Cos()
	y := dec.Cos() * ra.Sin()
	z := dec.Sin()

	// Step 3: Rotate ECI → ECEF using GMST
	gmst := sidereal.Apparent0UT(jd)
	cosGMST := gmst.Angle().Cos()
	sinGMST := gmst.Angle().Sin()

	xe := x*cosGMST + y*sinGMST
	ye := -x*sinGMST + y*cosGMST
	ze := z

	return base.Vec3{X: xe, Y: ye, Z: ze}
}
