package robot3d

import (
	"math"
	"sync"
)

// The robot in millimetres: M5Stack's structure files (m5stack/, MIT) placed by M5Stack's drawing
// (54 wide, 70.5 high, 61.5 deep; the plate 8 high), and the CoreS3 in front of the main body
// drawn here (54 x 54 x 15.5, the screen 320x240 = 40.8 x 30.6 in its black glass front).
//
// The robot's space: the yaw axis at x = 0, z = 0 on the ground (y = 0); the robot faces +Z.
const (
	robotH      = 70.5
	headBottom  = robotH - 54 // the head (CoreS3 + main body) is 54 high
	coreFront   = 33.6        // the yaw axis is 33.6 behind the front
	coreD       = 15.5        // the CoreS3's depth
	bodyFront   = coreFront - coreD
	coreR       = 3.5 // the CoreS3's rounded edges
	screenW     = 40.8
	screenH     = 30.6
	screenCY    = 1.0 // the screen's centre above the CoreS3's centre
	rim         = 1.4 // the shell's rim around the glass front
	servoBottom = 8.5 // the servo body stands on the plate (8 high)
	pitchAxisY  = servoBottom + 32.3
)

// coreCentre is the CoreS3's centre at rest.
var coreCentre = V3{0, headBottom + 27, coreFront - coreD/2}

// pivot is the pitch axis (along X) at rest: the servo's horn, over the yaw axis.
var pivot = V3{0, pitchAxisY, 0}

// Parts move differently: the plate stays, the servo body turns with yaw, the head (main body
// and CoreS3) with yaw and pitch.
const (
	partPlate = iota
	partServo
	partBody
	partCore
)

// Materials.
const (
	matPlate = iota
	matServo
	matServoCover
	matBody
	matBar       // the light guide bars of the LEDs
	matBackPanel // the upper back: a sticker and two ports
	matCore
	matPitchServo // the pitch servo (Feetech SCS0009, black), in the servo body's pocket
)

// The pitch servo, drawn (M5Stack's files leave the servos out): the pocket the servo body
// leaves on the robot's left, round the pitch axis, between its side and the main body's boss.
const (
	pitchServoX0, pitchServoX1 = 11.0, 22.0
	pitchServoY0, pitchServoY1 = 36.0, 47.5
	pitchServoZ0, pitchServoZ1 = -17.0, 10.0
)

// The LED bars' slot (one each side, at the top) and the upper back's panel.
const (
	barZ0, barZ1 = -16.3, 13.0
	barY0, barY1 = robotH - 1.8, robotH - 0.4
	backPanelZ   = bodyFront - 46.7 + 1.6
	backPanelY0  = robotH - 29 // its lower edge, from the photo of the back: the servo body shows below it
)

type vert struct{ p, n V3 } // at rest, in the robot's space

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
		// STL anchors: the turntable's centre (base, servo body), the main body's front.
		m = append(m, stlPart("StackChan-Base.stl", 106.2, -320.2, -32.6, V3{0, 0, 0}, partPlate, matPlate)...)
		m = append(m, stlPart("StackChan-ServoBody.stl", 229.45, -320.9, -22.3, V3{0, servoBottom, 0}, partServo, matServo)...)
		m = append(m, stlPart("StackChan-ServoSideCover.stl", 229.45, -320.9, -22.3, V3{0, servoBottom, 0}, partServo, matServoCover)...)
		m = append(m, stlPart("StackChan-ServoCover.stl", 229.45, -320.9, -22.3, V3{0, servoBottom, 0}, partServo, matServo)...) // the servo body's top, inside the head
		m = append(m, stlPart("StackChan-MainBody.stl", 475.7, -352.0, -17.7, V3{0, headBottom, bodyFront}, partBody, matBody)...)
		add := func(tris []tri, at V3) {
			for _, t := range tris {
				for k := range t.v {
					t.v[k].p = t.v[k].p.Add(at)
				}
				m = append(m, t)
			}
		}
		add(roundedBox(V3{27, 27, coreD / 2}, coreR, 6, partCore, matCore), coreCentre)
		add(roundedBox(V3{(pitchServoX1 - pitchServoX0) / 2, (pitchServoY1 - pitchServoY0) / 2, (pitchServoZ1 - pitchServoZ0) / 2}, 0.8, 2, partServo, matPitchServo),
			V3{(pitchServoX0 + pitchServoX1) / 2, (pitchServoY0 + pitchServoY1) / 2, (pitchServoZ0 + pitchServoZ1) / 2})
		for _, x := range []float64{-26.1, 26.1} {
			add(roundedBox(V3{0.75, (barY1 - barY0) / 2, (barZ1 - barZ0) / 2}, 0.6, 2, partBody, matBar),
				V3{x, (barY0 + barY1) / 2, (barZ0 + barZ1) / 2})
		}
		add(roundedBox(V3{25.6, (robotH - 1.2 - backPanelY0) / 2, 0.5}, 0.4, 1, partBody, matBackPanel),
			V3{0, (robotH - 1.2 + backPanelY0) / 2, backPanelZ})
		meshTris = m
	})
	return meshTris
}

// transforms gives each part's matrix (rest to world) for a pose.
func transforms(yaw, pitch float64) [4]M4 {
	ry := rotY(rad(yaw)) // +yaw: to the robot's left (+X), counter-clockwise seen from above
	head := ry.Mul(translate(pivot)).Mul(rotX(-rad(pitch))).Mul(translate(pivot.Mul(-1)))
	return [4]M4{identity(), ry, head, head}
}

func quad(a, b, c, d vert, part, mat int) []tri {
	return []tri{{[3]vert{a, b, c}, part, mat}, {[3]vert{a, c, d}, part, mat}}
}

// roundedBox is a box of half sizes h around the origin with edges rounded by r: a grid on each
// face, pushed out from the inner box, denser where the edges bend.
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
						c[k] = point(V3{a[0], a[1], a[2]})
					}
					if s < 0 { // counter-clockwise seen from outside, as the + side
						c[1], c[3] = c[3], c[1]
					}
					out = append(out, quad(c[0], c[1], c[2], c[3], part, mat)...)
				}
			}
		}
	}
	return out
}
