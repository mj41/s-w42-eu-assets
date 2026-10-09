package robot3d

import (
	"math"
	"sync"
)

// The robot's sizes in millimetres, measured from photos of an M5Stack Stackchan: the head is
// the CoreS3 cube, the screen 320x240 (40.8 x 30.6 mm) in a black glass front; under the head a
// light turntable (the yaw servo) on a dark chamfered plate.
const (
	headW, headH, headD = 54.0, 54.0, 52.0
	headR               = 5.0 // the head's rounded edges
	plateHalf           = 22.0
	plateChamfer        = 6.5
	plateTop            = 10.0
	plateBevel          = 1.2
	discR               = 17.0
	discTop             = 14.0
	headBottom          = discTop + 0.6
	screenW, screenH    = 40.8, 30.6
	screenCY            = 0.6 // the screen's centre above the head's centre
	rim                 = 1.6 // the shell's rim around the glass front
)

// pivot is the pitch axis in head space (head centre = origin): low near the back, so the front
// lifts when the robot looks up and the back stays over the turntable, as on the robot.
var pivot = V3{0, -headH/2 + 5, -headD/2 + 6}

// headRest is the head's centre at yaw 0, pitch 0.
var headRest = V3{0, headBottom + headH/2, 0}

// Parts move differently: the plate stays, the disc turns with yaw, the head with yaw and pitch.
const (
	partPlate = iota
	partDisc
	partHead
)

// Materials.
const (
	matPlate = iota
	matPlateHole
	matDisc
	matDiscRing
	matHead
)

type vert struct{ p, n V3 } // in the part's own space

type tri struct {
	v    [3]vert
	part int
	mat  int
}

var (
	meshOnce sync.Once
	meshTris []tri
)

func mesh() []tri {
	meshOnce.Do(func() {
		var m []tri
		m = append(m, plate()...)
		m = append(m, disc()...)
		m = append(m, roundedBox(V3{headW / 2, headH / 2, headD / 2}, headR, 7, partHead, matHead)...)
		meshTris = m
	})
	return meshTris
}

// transforms gives each part's matrix (part space to world) for a pose.
func transforms(yaw, pitch float64) [3]M4 {
	ry := rotY(rad(yaw)) // +yaw: to the robot's left (+X), counter-clockwise seen from above
	head := ry.Mul(translate(headRest.Add(pivot))).Mul(rotX(-rad(pitch))).Mul(translate(pivot.Mul(-1)))
	return [3]M4{identity(), ry, head}
}

func quad(a, b, c, d vert, part, mat int) []tri {
	return []tri{{[3]vert{a, b, c}, part, mat}, {[3]vert{a, c, d}, part, mat}}
}

// roundedBox is a box of half sizes h with edges rounded by r: a grid on each face, pushed out
// from the inner box, denser where the edges bend.
func roundedBox(h V3, r float64, steps int, part, mat int) []tri {
	in := V3{h.X - r, h.Y - r, h.Z - r}
	coords := func(inner float64) []float64 {
		var c []float64
		for k := steps; k >= 1; k-- {
			c = append(c, -(inner + r*math.Tan(float64(k)*math.Pi/4/float64(steps))))
		}
		c = append(c, -inner, inner)
		for k := 1; k <= steps; k++ {
			c = append(c, inner+r*math.Tan(float64(k)*math.Pi/4/float64(steps)))
		}
		return c
	}
	get := func(v V3, i int) float64 { return [3]float64{v.X, v.Y, v.Z}[i] }
	mk := func(a [3]float64) V3 { return V3{a[0], a[1], a[2]} }
	point := func(p V3) vert {
		q := V3{clamp(p.X, -in.X, in.X), clamp(p.Y, -in.Y, in.Y), clamp(p.Z, -in.Z, in.Z)}
		n := p.Sub(q).Norm()
		return vert{q.Add(n.Mul(r)), n}
	}
	var out []tri
	for axis := 0; axis < 3; axis++ {
		u, v := (axis+1)%3, (axis+2)%3
		cu, cv := coords(get(in, u)), coords(get(in, v))
		for _, s := range []float64{-1, 1} {
			for i := 0; i+1 < len(cu); i++ {
				for j := 0; j+1 < len(cv); j++ {
					var c [4]vert
					for k, ij := range [4][2]int{{i, j}, {i + 1, j}, {i + 1, j + 1}, {i, j + 1}} {
						var a [3]float64
						a[axis] = s * get(h, axis)
						a[u], a[v] = cu[ij[0]], cv[ij[1]]
						c[k] = point(mk(a))
					}
					out = append(out, quad(c[0], c[1], c[2], c[3], part, mat)...)
				}
			}
		}
	}
	return out
}

// plate is the base: an octagon (a square with chamfered corners) with a bevelled top edge and
// four screw holes.
func plate() []tri {
	oct := func(half, ch float64) []V3 {
		return []V3{{half - ch, 0, half}, {half, 0, half - ch}, {half, 0, -half + ch}, {half - ch, 0, -half},
			{-half + ch, 0, -half}, {-half, 0, -half + ch}, {-half, 0, half - ch}, {-half + ch, 0, half}}
	}
	lo, hi := oct(plateHalf, plateChamfer), oct(plateHalf-plateBevel, plateChamfer-plateBevel*0.4)
	at := func(p V3, y float64) V3 { return V3{p.X, y, p.Z} }
	var out []tri
	for i := range lo {
		j := (i + 1) % len(lo)
		side := lo[j].Sub(lo[i]).Cross(V3{0, 1, 0}).Norm().Mul(-1)
		if side.Dot(lo[i]) < 0 {
			side = side.Mul(-1)
		}
		out = append(out, quad(vert{at(lo[i], 0), side}, vert{at(lo[j], 0), side},
			vert{at(lo[j], plateTop-plateBevel), side}, vert{at(lo[i], plateTop-plateBevel), side}, partPlate, matPlate)...)
		bev := side.Add(V3{0, 1, 0}).Norm()
		out = append(out, quad(vert{at(lo[i], plateTop-plateBevel), bev}, vert{at(lo[j], plateTop-plateBevel), bev},
			vert{at(hi[j], plateTop), bev}, vert{at(hi[i], plateTop), bev}, partPlate, matPlate)...)
		up := V3{0, 1, 0}
		out = append(out, tri{[3]vert{{V3{0, plateTop, 0}, up}, {at(hi[j], plateTop), up}, {at(hi[i], plateTop), up}}, partPlate, matPlate})
	}
	// screw holes: dark discs just above the top
	for _, c := range []V3{{15.5, 0, 15.5}, {-15.5, 0, 15.5}, {15.5, 0, -15.5}, {-15.5, 0, -15.5}} {
		out = append(out, flatDisc(V3{c.X, plateTop + 0.05, c.Z}, 1.9, 16, partPlate, matPlateHole)...)
	}
	return out
}

func flatDisc(c V3, r float64, n int, part, mat int) []tri {
	var out []tri
	up := V3{0, 1, 0}
	for i := 0; i < n; i++ {
		a0, a1 := 2*math.Pi*float64(i)/float64(n), 2*math.Pi*float64(i+1)/float64(n)
		p0 := c.Add(V3{r * math.Sin(a0), 0, r * math.Cos(a0)})
		p1 := c.Add(V3{r * math.Sin(a1), 0, r * math.Cos(a1)})
		out = append(out, tri{[3]vert{{c, up}, {p1, up}, {p0, up}}, part, mat})
	}
	return out
}

// disc is the turntable: a light cylinder with a darker ring at its foot.
func disc() []tri {
	const n = 48
	var out []tri
	ring := func(r, y0, y1 float64, mat int) {
		for i := 0; i < n; i++ {
			a0, a1 := 2*math.Pi*float64(i)/n, 2*math.Pi*float64(i+1)/n
			n0, n1 := V3{math.Sin(a0), 0, math.Cos(a0)}, V3{math.Sin(a1), 0, math.Cos(a1)}
			out = append(out, quad(vert{V3{r * n0.X, y0, r * n0.Z}, n0}, vert{V3{r * n1.X, y0, r * n1.Z}, n1},
				vert{V3{r * n1.X, y1, r * n1.Z}, n1}, vert{V3{r * n0.X, y1, r * n0.Z}, n0}, partDisc, mat)...)
		}
	}
	ring(discR+0.8, plateTop, plateTop+1.2, matDiscRing)
	out = append(out, flatDisc(V3{0, plateTop + 1.2, 0}, discR+0.8, n, partDisc, matDiscRing)...)
	ring(discR, plateTop+1.2, discTop, matDisc)
	out = append(out, flatDisc(V3{0, discTop, 0}, discR, n, partDisc, matDisc)...)
	return out
}
