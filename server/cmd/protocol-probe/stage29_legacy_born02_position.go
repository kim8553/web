package main

import "github.com/local/9yin-go-server/internal/role"

// stage29LegacyBorn02Position preserves the 2026-09-13 LIVE position discriminator.
// Only the exact legacy born02 coordinate window is remapped; every other scene
// and coordinate is passed through. This is not a guessed universal spawn point.
func stage29LegacyBorn02Position(scene role.Scene, position role.Position) (role.Position, bool) {
	if scene.Resource != "born02" ||
		position.X <= 693.907 || position.X >= 693.909 ||
		position.Y <= 24.693 || position.Y >= 24.695 ||
		position.Z <= 404.349 || position.Z >= 404.351 {
		return position, false
	}
	position.X = 905.069
	position.Y = 10.810
	position.Z = 196.980
	position.Orient = 1.610
	return position, true
}
