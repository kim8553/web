# Current authority / provenance boundary

## Editable source baseline

- User archive: `9yin-go-server1(3).rar`
- SHA256: `ddc2f6bc078660a40eedf077d43befd6402ed59478ccde987d44171c1d7aefa5`
- Go source files in archive: 112
- Baseline Windows EXE (`build/9yin-game-native-menu.exe`) SHA256:
  `7acc63911cfa36ca5ac6f99d9e9c079ae5f80e9066252f7be5dad47f3cbe89d2`
- Baseline EXE Go toolchain observed by `go version -m`: Go 1.23.2.

This baseline is an implementation starting point, not official-current behavior authority.

## Later exact server implementation reference

- `9yin-game-native-menu.exe`
- SHA256: `fee2df844b00e7df07508b9e67f64f0da9df20f00ace09befed790ef28b0d71c`
- Exact `.text` SHA256:
  `f8c1d46bccd6629e0a974604df6e937cc24917c4cf4bbfa2a4df25420627cc93`

Use this binary to recover later Go implementation deltas where evidence exists. Do not treat
its behavior as official Snail gameplay semantics.

## Exact current Snail client authority

- `fxgame.exe` SHA256:
  `c7f0366026452d525dc0c74b8499aac30fdb69189591420978e12c7094b5fde3`
- `FxGameLogic.dll` SHA256:
  `3bfa3832c04291d9d7a44f6bc13cceb609aad55ba90e2ad937cc77221089ef96`
- `fxnet2.dll` SHA256:
  `0443d9f401fdcc6a92a391869c898780ffac93399913dc9a82c7cac8cf6c3fde`
- `fxcore.dll` SHA256:
  `ce0da0f52a80db5033e177a59be868c44b0baeca4a71226b0ccb2aca76cae724`

Only these exact current binaries plus matching current-client Lua/resources are authority for
current-client wire behavior and client-side semantics.

## Current proven 211 / 212 boundary carried into this source track

- 211 (`0xD3`) exact client wire layout is V0..V9 without target and V0..V13 with target.
- V1 is skill ConfigID; V2..V4 current visual XYZ; V5 is `visual_rotation[1]` (do not rename
  it to literal AngleY); V6..V8 are `CurSkillTargetX/Y/Z`; V9 is a condition-dependent
  auxiliary string; V10 is the 8-byte target identity; V11..V13 are target visual XYZ.
- 212 (`0xD4`) carries current visual XYZ, `visual_rotation[1]`, and int32 flag.
- The exact official enum names/semantics of the 212 flag remain unproven. Preserve observe-only.

Static proof does not establish live current-client E2E success.
