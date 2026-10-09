package robot3d

import "image/color"

// Part is a piece of the model at rest (yaw 0, pitch 0) in the robot's space (millimetres, Y up,
// facing +Z, the yaw axis through x = 0, z = 0), for other renderers (e.g. glTF for a browser).
type Part struct {
	Name  string     // plate, servo, servo-cover, body, led-bar-left, led-bar-right, back-panel, core
	Joint string     // what moves it: "base" (nothing), "yaw" (turns about the yaw axis), "head" (yaw, then pitch about PitchPivot)
	Color color.RGBA // its base colour (details such as the screen, ports and labels are drawn, not modelled)
	// Triangles, three positions each, and a normal per position.
	Positions []V3
	Normals   []V3
}

// PitchPivot is a point on the head's pitch axis (the axis is along X): +pitch lifts the front.
// The yaw axis is the Y axis; +yaw turns the head to the robot's left (+X).
func PitchPivot() V3 { return pivot }

// Screen is the screen's rectangle at rest: its centre on the CoreS3's front, its width and
// height (320 x 240 pixels, the picture's top left at Centre - (Width/2, -Height/2)).
func Screen() (centre V3, width, height float64) {
	return V3{coreCentre.X, coreCentre.Y + screenCY, coreFront}, screenW, screenH
}

// Parts gives the model's parts at rest.
func Parts() []Part {
	type key struct{ part, mat int }
	names := map[key]string{
		{partPlate, matPlate}: "plate", {partServo, matServo}: "servo", {partServo, matServoCover}: "servo-cover",
		{partBody, matBody}: "body", {partBody, matBackPanel}: "back-panel", {partCore, matCore}: "core",
	}
	joints := map[int]string{partPlate: "base", partServo: "yaw", partBody: "head", partCore: "head"}
	colours := map[int]rgb{matPlate: colPlate, matServo: colDisc, matServoCover: colServo, matBody: colShell,
		matBar: colBarOff, matBackPanel: colShell, matCore: colShell}
	byName := map[string]*Part{}
	var order []string
	for _, t := range mesh() {
		name := names[key{t.part, t.mat}]
		if t.mat == matBar {
			name = "led-bar-right"
			if t.v[0].p.X > 0 {
				name = "led-bar-left"
			}
		}
		p := byName[name]
		if p == nil {
			c := colours[t.mat]
			p = &Part{Name: name, Joint: joints[t.part], Color: color.RGBA{q8(c.R), q8(c.G), q8(c.B), 255}}
			byName[name] = p
			order = append(order, name)
		}
		for _, v := range t.v {
			p.Positions = append(p.Positions, v.p)
			p.Normals = append(p.Normals, v.n)
		}
	}
	out := make([]Part, 0, len(order))
	for _, n := range order {
		out = append(out, *byName[n])
	}
	return out
}
