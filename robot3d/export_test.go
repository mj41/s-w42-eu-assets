package robot3d

import "testing"

// Surface gives the renderer's details: the glass at the screen, the power button and USB-C on
// the CoreS3's right, the back panel's blue port; the plain shell elsewhere.
func TestSurface(t *testing.T) {
	shell, _ := Surface("core", V3{0, coreCentre.Y, coreCentre.Z}, V3{0, 1, 0})
	centre, _, _ := Screen()
	if _, glass := Surface("core", centre, V3{0, 0, 1}); !glass {
		t.Error("the screen's centre is not glass")
	}
	right := V3{-1, 0, 0}
	for name, p := range map[string]V3{
		"power button": coreCentre.Add(V3{-27, 14, 0.4}),
		"USB-C":        coreCentre.Add(V3{-27, -0.7, 0.4}),
	} {
		if c, _ := Surface("core", p, right); c == shell {
			t.Errorf("%s: the plain shell", name)
		}
	}
	if c, _ := Surface("back-panel", V3{12, robotH - 10, backPanelZ - 0.5}, V3{0, 0, -1}); c.B <= c.R {
		t.Errorf("the blue port: %v", c)
	}
	if c, glass := Surface("core", coreCentre.Add(V3{-27, -20, 0.4}), right); glass || c != shell {
		t.Errorf("the right side below the ports: %v %v, want the shell %v", c, glass, shell)
	}
}
