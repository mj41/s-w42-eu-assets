package robot3d

import (
	"math"
)

// rgb is a linear-ish colour, 0..1 a channel.
type rgb struct{ R, G, B float64 }

func (a rgb) add(b rgb) rgb             { return rgb{a.R + b.R, a.G + b.G, a.B + b.B} }
func (a rgb) mul(s float64) rgb         { return rgb{a.R * s, a.G * s, a.B * s} }
func (a rgb) mulc(b rgb) rgb            { return rgb{a.R * b.R, a.G * b.G, a.B * b.B} }
func (a rgb) lerp(b rgb, t float64) rgb { return a.add(b.add(a.mul(-1)).mul(t)) }

func hex(v uint32) rgb {
	return rgb{float64(v>>16&0xff) / 255, float64(v>>8&0xff) / 255, float64(v&0xff) / 255}
}

var (
	colShell    = hex(0xe4e5e7) // the head's light grey shell
	colGlass    = hex(0x0d0e11) // the black glass front
	colVent     = hex(0x55595e)
	colPortDark = hex(0x1c1d20)
	colPortBlue = hex(0x2f9be0)
	colServo    = hex(0x50555b) // the open back: the servo inside
	colBarOff   = hex(0xeef3f5) // the LED bar, unlit
	colRing     = hex(0xc4432c) // the red ring under the screen
	colPlate    = hex(0x5b6067)
	colHole     = hex(0x2c2f33)
	colDisc     = hex(0xc9ccd0)
	colDiscRing = hex(0x8d9298)
)

// lights: a key light from the upper left front, a fill from the right, a soft one from behind,
// a sky above.
var (
	keyDir  = V3{-0.45, 0.8, 0.55}.Norm()
	fillDir = V3{0.75, 0.25, 0.45}.Norm()
	backDir = V3{0.2, 0.5, -0.85}.Norm()
)

// surface is what a point looks like before the lights: its colour, shine, and light of its own.
type surface struct {
	albedo   rgb
	spec     float64 // specular strength
	shine    float64 // specular exponent
	emissive rgb     // light of its own (the screen, lit LEDs)
	glass    bool    // reflects the room
}

// scene holds what a frame's shading needs.
type scene struct {
	screen     *texture
	brightness float64
	leds       [12]*rgb
}

// head shading by the point's place on the head (head space: centre at 0, +Z the front).
func (s *scene) head(p, n V3) surface {
	sf := surface{albedo: colShell, spec: 0.18, shine: 24}
	switch {
	case n.Z > 0.97: // the front
		gx, gy := headW/2-rim, headH/2-rim
		if roundRect(p.X, p.Y, gx, gy, headR-rim) <= 0 {
			sf = surface{albedo: colGlass, spec: 0.9, shine: 90, glass: true}
			sx, sy := p.X+screenW/2, (screenCY+screenH/2)-p.Y // from the screen's top left
			if sx >= 0 && sx <= screenW && sy >= 0 && sy <= screenH {
				if s.screen != nil {
					sf.emissive = s.screen.sample(sx/screenW, sy/screenH).mul(s.brightness)
				}
			} else {
				s.frontMarks(p, &sf)
			}
		}
	case n.X > 0.97 || n.X < -0.97: // the sides
		s.side(p, n.X > 0, &sf)
	case n.Z < -0.97: // the back: a sticker and ports at the top, open below (the servo inside)
		switch {
		case p.Y < 4:
			sf = surface{albedo: hex(0x2f3338), spec: 0.05, shine: 10}
			if math.Abs(p.X) < 14 && p.Y < 1 { // the servo
				sf.albedo, sf.spec = colServo, 0.15
				if math.Abs(p.X) > 12.5 || p.Y > -0.6 {
					sf.albedo = colServo.mul(0.75)
				}
			}
		case box(p.X, p.Y, -13, 19.5, 6, 26):
			sf.albedo, sf.spec = hex(0xf2f2f0), 0.2
		case box(p.X, p.Y, 9, 15.5, 16.5, 18.5):
			sf.albedo = colPortBlue
		case box(p.X, p.Y, -13.4, 15.3, -6.7, 18.5):
			sf.albedo = colPortDark
		}
	}
	return sf
}

// frontMarks: under the screen, the red ring and the sensors' dots.
func (s *scene) frontMarks(p V3, sf *surface) {
	const y = -headH/2 + 6.4
	d := math.Hypot(p.X, p.Y-y)
	if d < 1.9 && d > 1.25 {
		sf.albedo, sf.glass = colRing, false
		sf.emissive = colRing.mul(0.25)
		return
	}
	for _, dot := range [][2]float64{{-14.5, 0.55}, {5.5, 0.3}, {7.5, 0.3}, {13.5, 0.5}} {
		if math.Hypot(p.X-dot[0], p.Y-y) < dot[1] {
			sf.albedo = hex(0x2a2c31)
		}
	}
}

// side: the front 16 mm is the CoreS3 module (a seam behind it): vents on the robot's left (+X),
// the power button, USB-C and the red Grove port on its right. Behind it on both sides: the
// label (a light sticker with a dark end, drawn without its text), three round holes, and the
// LED bar along the top; on the left (+X) LED 0 is at the front, on the right LED 11.
func (s *scene) side(p V3, left bool, sf *surface) {
	z, y := p.Z, p.Y
	const seam = headD/2 - 16
	switch {
	case math.Abs(z-seam) < 0.25 && math.Abs(y) < headH/2-headR:
		sf.albedo = colShell.mul(0.7)
	case z > seam && left: // vents: a hex grid of small holes
		if z > seam+1.8 && z < headD/2-3.5 && y > -4 && y < 17 {
			const step = 2.1
			row := math.Round(y / (step * 0.866))
			off := 0.0
			if int(row)%2 != 0 {
				off = step / 2
			}
			cz := math.Round((z-off)/step)*step + off
			if math.Hypot(z-cz, y-row*step*0.866) < 0.5 {
				sf.albedo, sf.spec = colVent, 0
			}
		}
	case z > seam: // the power button, USB-C, the Grove port
		const cz = seam + 7
		if d := roundRect(z-cz, y-16, 2.6, 2.2, 1); d <= 0 {
			sf.albedo = colShell.mul(0.88)
			if d > -0.35 {
				sf.albedo = colShell.mul(0.6)
			}
		}
		if roundRect(z-cz, y-3, 1.3, 4.2, 1.2) <= 0 {
			sf.albedo, sf.spec = colPortDark, 0.2
		}
		if d := roundRect(z-cz, y+11, 2.4, 3.2, 0.4); d <= 0 {
			sf.albedo = hex(0xa83a3c)
			if d < -0.8 {
				sf.albedo = hex(0x3a1416)
			}
		}
	default: // the body: label, holes
		const l0, l1 = seam - 6.5, seam - 0.8
		if z >= l0 && z <= l1 && y > -headH/2+3 && y < headH/2-4 {
			sf.albedo, sf.spec = hex(0xf3f3f1), 0.25
			if (left && y > 9) || (!left && y < -14) {
				sf.albedo = colPortDark
			}
		}
		for _, hz := range []float64{-12.6, -8.1, -3.6} {
			d := math.Hypot(z-hz, y-14.5)
			if d < 2.0 {
				sf.albedo = colShell.mul(0.72)
				if d < 1.35 {
					sf.albedo = hex(0x2b3138)
				}
			}
		}
	}
	// the LED bar
	const z0, z1, y0, y1 = -20.5, -2.5, 20.6, 22.0
	if z > z0-3 && z < z1+3 && y > y0-3 && y < y1+3 {
		k := int(clamp((z1-z)/((z1-z0)/6), 0, 5)) // 0 at the front
		idx := k
		if !left {
			idx = 11 - k
		}
		inBar := z >= z0 && z <= z1 && y >= y0 && y <= y1
		if inBar {
			sf.albedo, sf.spec, sf.shine = colBarOff, 0.3, 40
		}
		if c := s.leds[idx]; c != nil {
			dz := math.Max(0, math.Max(z0-z, z-z1))
			dy := math.Max(0, math.Max(y0-y, y-y1))
			if inBar {
				sf.emissive = sf.emissive.add(c.mul(0.9)).add(rgb{0.25, 0.25, 0.25}.mulc(*c))
				sf.albedo = colBarOff.lerp(*c, 0.6)
			} else {
				sf.emissive = sf.emissive.add(c.mul(0.35 * math.Exp(-math.Hypot(dz, dy)/1.1)))
			}
		}
	}
}

func box(x, y, x0, y0, x1, y1 float64) bool { return x >= x0 && x <= x1 && y >= y0 && y <= y1 }

// roundRect is the signed distance to a rectangle of half sizes hx, hy with corners rounded by r.
func roundRect(x, y, hx, hy, r float64) float64 {
	qx, qy := math.Abs(x)-hx+r, math.Abs(y)-hy+r
	return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
}

// light shades a surface: n the world normal, v the direction to the camera.
func light(sf surface, n, v V3) rgb {
	amb := 0.52 + 0.06*n.Y
	diff := amb + 0.4*math.Max(0, n.Dot(keyDir)) + 0.22*math.Max(0, n.Dot(fillDir)) + 0.2*math.Max(0, n.Dot(backDir))
	c := sf.albedo.mul(diff)
	for _, l := range []struct {
		d V3
		i float64
	}{{keyDir, 1}, {fillDir, 0.35}} {
		h := l.d.Add(v).Norm()
		c = c.add(rgb{1, 1, 1}.mul(sf.spec * l.i * math.Pow(math.Max(0, n.Dot(h)), sf.shine)))
	}
	if sf.glass { // the room in the glass: brighter above, more at grazing angles
		r := n.Mul(2 * n.Dot(v)).Sub(v)
		f := 0.05 + 0.6*math.Pow(1-math.Max(0, n.Dot(v)), 4)
		env := hex(0x3a3d44).lerp(hex(0xd8dde6), smoothstep(-0.2, 0.9, r.Y))
		c = c.add(env.mul(f * 0.55))
	}
	return c.add(sf.emissive)
}

// texture is an image as floats, sampled bilinearly.
type texture struct {
	w, h int
	px   []rgb
}

func (t *texture) sample(u, v float64) rgb {
	x := clamp(u*float64(t.w)-0.5, 0, float64(t.w-1))
	y := clamp(v*float64(t.h)-0.5, 0, float64(t.h-1))
	x0, y0 := int(x), int(y)
	x1, y1 := min(x0+1, t.w-1), min(y0+1, t.h-1)
	fx, fy := x-float64(x0), y-float64(y0)
	a := t.px[y0*t.w+x0].lerp(t.px[y0*t.w+x1], fx)
	b := t.px[y1*t.w+x0].lerp(t.px[y1*t.w+x1], fx)
	return a.lerp(b, fy)
}

// material is a non-head part's look; p, n in its own space.
func material(mat int, p, n V3) surface {
	switch mat {
	case matPlate:
		sf := surface{albedo: colPlate, spec: 0.12, shine: 20}
		if n.Z > 0.9 && math.Abs(p.X) < 2 && p.Y < 6.5 { // the notch between its front feet
			sf.albedo = colPlate.mul(0.6)
		}
		if n.Z < -0.9 && math.Abs(p.X) < 4.3 && math.Abs(p.Y-4.5) < 1.5 { // USB-C at the back
			sf.albedo = colPortDark
		}
		return sf
	case matPlateHole:
		return surface{albedo: colHole}
	case matDisc:
		return surface{albedo: colDisc, spec: 0.25, shine: 30}
	case matDiscRing:
		return surface{albedo: colDiscRing, spec: 0.15, shine: 20}
	}
	return surface{albedo: colShell}
}
