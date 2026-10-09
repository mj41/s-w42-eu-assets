# s-w42-eu-assets

Assets for the Stackchan projects: robot3d (a 3D model of the robot drawn in Go, with any screen).

- Go only, standard library only (robot3d is imported by other tools, e.g. a video recorder).
- Photos from a phone carry GPS: never commit them as they are (`photos-*/` is ignored).
- Never name things "stackchan-..." (trademark); labels on the model are drawn without text.
- robot3d/m5stack/*.stl are M5Stack's files (MIT), unchanged; keep their LICENSE next to them.
- Check a change against the photos: render the same view and put them side by side.
- `GOWORK=off go test ./...` (the parent directory has a go.work without this module).
