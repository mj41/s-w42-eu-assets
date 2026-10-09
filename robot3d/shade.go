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
	colDisc     = hex(0xc9ccd0)
	colMotor    = hex(0x1d1e21) // the servos themselves: black
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

// core shades the CoreS3 (p, n in the robot's rest space): the glass front with the screen, the
// red ring and the sensors' dots; vents on the robot's left (+X), the power button, USB-C and the
// Grove port on its right (sizes from M5Stack's drawing).
func (s *scene) core(p, n V3) surface {
	sf := surface{albedo: colShell, spec: 0.18, shine: 24}
	p = p.Sub(coreCentre)
	switch {
	case n.Z > 0.97: // the front
		if roundRect(p.X, p.Y, 27-rim, 27-rim, coreR-rim) <= 0 {
			sf = surface{albedo: colGlass, spec: 0.9, shine: 90, glass: true}
			sx, sy := p.X+screenW/2, (screenCY+screenH/2)-p.Y // from the screen's top left
			if sx >= 0 && sx <= screenW && sy >= 0 && sy <= screenH {
				if s.screen != nil {
					sf.emissive = s.screen.sample(sx/screenW, sy/screenH).mul(s.brightness)
				}
			} else {
				frontMarks(p, &sf)
			}
		}
	case n.X > 0.97 && p.Z > -coreD/2+1.5 && p.Z < coreD/2-2: // left: vents, a hex grid of holes
		if p.Y > -6 && p.Y < 19 {
			const step = 2.1
			row := math.Round(p.Y / (step * 0.866))
			off := 0.0
			if int(row)%2 != 0 {
				off = step / 2
			}
			cz := math.Round((p.Z-off)/step)*step + off
			if math.Hypot(p.Z-cz, p.Y-row*step*0.866) < 0.5 {
				sf.albedo, sf.spec = colVent, 0
			}
		}
	case n.X < -0.97: // right: the power button, USB-C, the Grove port
		const cz = 0.4
		if d := roundRect(p.Z-cz, p.Y-14, 3, 2.6, 1.2); d <= 0 {
			sf.albedo = colShell.mul(0.9)
			if d > -0.35 {
				sf.albedo = colShell.mul(0.6)
			}
		}
		if roundRect(p.Z-cz, p.Y+0.7, 1.5, 4.4, 1.4) <= 0 {
			sf.albedo, sf.spec = colPortDark, 0.2
		}
		if d := roundRect(p.Z-cz, p.Y+13.5, 2.5, 4, 0.4); d <= 0 {
			sf.albedo = hex(0xa83a3c)
			if d < -0.8 {
				sf.albedo = hex(0x3a1416)
			}
		}
	}
	return sf
}

// frontMarks: under the screen, the red ring and the sensors' dots (p from the CoreS3's centre).
func frontMarks(p V3, sf *surface) {
	const y = -27 + 6.7
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

// body shades M5Stack's main body (p, n in the robot's rest space): on each side the label (a
// light sticker with a dark end, drawn without its text) and the LED bar along the top (on the
// left, +X, LED 0 at the front; on the right LED 11); at the back a sticker and two ports.
func (s *scene) body(p, n V3, mat int) surface {
	sf := surface{albedo: colShell, spec: 0.18, shine: 24}
	top := robotH
	switch {
	case mat == matBar:
		s.ledBar(p.Z, p.X > 0, &sf)
	case math.Abs(n.X) > 0.97 && math.Abs(p.X) > 26:
		left := p.X > 0
		if p.Z > 10.9 && p.Z < 16.2 && p.Y > headBottom+9.5 && p.Y < top-5.5 {
			sf.albedo, sf.spec = hex(0xf3f3f1), 0.25
			if (left && p.Y > top-18) || (!left && p.Y < headBottom+20) {
				sf.albedo = colPortDark
			}
		}
		s.barGlow(p.Z, p.Y, p.X > 0, &sf)
	case mat == matTopBoard && n.Y > 0.9: // from the left (+X): port C, the IR window, port B
		switch {
		case box(p.X, p.Z, 13.5, 13.6, 18.5, 17.2), box(p.X, p.Z, -18.5, 13.6, -13.5, 17.2):
			sf.albedo, sf.spec = colPortDark, 0.1
		case box(p.X, p.Z, 1.5, 14.2, 5.0, 16.6):
			sf.albedo, sf.spec = hex(0x111216), 0.6
		}
	case mat == matBackPanel && n.Z < -0.9:
		switch {
		case box(p.X, p.Y, -13, top-7, 6, top-1):
			sf.albedo, sf.spec = hex(0xf2f2f0), 0.2
		case box(p.X, p.Y, 9, top-11.5, 16.5, top-8.5):
			sf.albedo = colPortBlue
		case box(p.X, p.Y, -13.4, top-11.7, -6.7, top-8.5):
			sf.albedo = colPortDark
		}
	}
	return sf
}

// led is the LED under a point of a bar: six along it, on the left (+X) LED 0 at the front, on
// the right LED 11.
func (s *scene) led(z float64, left bool) *rgb {
	k := int(clamp((barZ1-z)/((barZ1-barZ0)/6), 0, 5))
	if !left {
		k = 11 - k
	}
	return s.leds[k]
}

// ledBar shades a light guide bar: milky when off, its LED's colour when on.
func (s *scene) ledBar(z float64, left bool, sf *surface) {
	sf.albedo, sf.spec, sf.shine = colBarOff, 0.3, 40
	if c := s.led(z, left); c != nil {
		sf.albedo, sf.spec = c.mul(0.35), 0.15 // the light, not the plastic
		sf.emissive = c.mul(0.75)
	}
}

// barGlow: a lit LED's light on the shell around its bar.
func (s *scene) barGlow(z, y float64, left bool, sf *surface) {
	dz := math.Max(0, math.Max(barZ0-z, z-barZ1))
	dy := math.Max(0, math.Max(barY0-y, y-barY1))
	if d := math.Hypot(dz, dy); d < 4 {
		if c := s.led(clamp(z, barZ0, barZ1), left); c != nil {
			sf.emissive = sf.emissive.add(c.mul(0.35 * math.Exp(-d/1.1)))
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

// material is the plate's and the servo's look.
func material(mat int) surface {
	switch mat {
	case matPlate:
		return surface{albedo: colPlate, spec: 0.12, shine: 20}
	case matServo:
		return surface{albedo: colDisc, spec: 0.25, shine: 30}
	case matServoCover:
		return surface{albedo: colServo, spec: 0.15, shine: 20}
	case matPitchServo:
		return surface{albedo: colMotor, spec: 0.2, shine: 30}
	}
	return surface{albedo: colShell}
}
