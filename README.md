# s-w42-eu-assets

Assets for the Stackchan projects on [s.w42.eu](https://s.w42.eu).

## robot3d: the robot in 3D, with any screen

A simplified 3D model of an M5Stack Stackchan robot, built in code from photos of a real one, and
a small renderer, all in Go (standard library only, no GPU, no cgo; the same pixels on every
machine). Put any picture on the robot's screen (a screenshot taken over USB, a frame of a
recording, a mock-up), turn its head, move the camera round it, light its LEDs.

```bash
GOPRIVATE=github.com/mj41/* go install github.com/mj41/s-w42-eu-assets/cmd/robot3d@latest   # a private repo

s-w42-eu-usb screenshot -o screen.jpg                  # the robot's screen over USB (320x240)
robot3d -screen screen.jpg -o robot.png                # the robot showing it (800x800, transparent)
robot3d -screen screen.jpg -az 30 -el 12 -pitch 25 -yaw 10 -o look-up.png
robot3d -screen screen.jpg -leds "#ff6a00*12" -bg "#f4f6ff" -o orange.png
robot3d -screen screen.jpg -frames 72 -o spin/%03d.png # the camera once round the robot
ffmpeg -framerate 24 -i spin/%03d.png -pix_fmt yuv420p spin.mp4
```

| Flag | |
|---|---|
| `-screen` | the screen's picture, JPEG or PNG, drawn 4:3; empty: the screen is off |
| `-brightness` | the screen, 0..1 |
| `-leds` | 12 colours, `#rrggbb` comma separated, numbered as the robot's `leds` command (left 0-5, right 6-11; 0 and 11 at the front); `#rrggbb*12` for all |
| `-yaw`, `-pitch` | the head in degrees, as the robot's `look` command: yaw + to the robot's left, pitch up from level (the robot allows 5..85) |
| `-az`, `-el` | the camera round the robot in degrees: 0, 0 is straight in front; + azimuth to the robot's left |
| `-zoom`, `-fov` | 1 fits the robot in any pose, so a moving head keeps a steady frame |
| `-size`, `-bg`, `-no-shadow`, `-ss` | the picture's size, its background (transparent if empty), the soft shadow on the ground, supersampling |
| `-frames`, `-turn` | frames of the camera going round by `-turn` degrees; `-o` is then a pattern |

From Go:

```go
img := robot3d.Render(robot3d.Options{Width: 800, Height: 800, Screen: screen, Pitch: 20, Azimuth: 25, Elevation: 10})
```

The model: the head is the CoreS3 cube (54 x 54 x 52 mm, rounded edges) with the screen
(40.8 x 30.6 mm) in its black glass front, the red ring and the sensors' dots under it; on its
sides the CoreS3 section (vents on the left; power button, USB-C and Grove port on the right),
the label, three holes and the LED bar; at the back the ports, and the open bottom with the servo.
Under the head the turntable (yaw) on the dark chamfered plate. Details are drawn, not modelled,
and labels are drawn without text. The head pitches about a point low at its back, as on the
robot.

Stack-chan is a registered trademark of Shinya Ishikawa; M5Stack is a trademark of M5Stack
Technology. This project is independent of both.

## License

Apache License 2.0, see [LICENSE](LICENSE).
