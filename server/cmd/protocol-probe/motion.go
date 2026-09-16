package main

import (
	"encoding/binary"
	"math"
)

func parseMotionPosition(msg []byte) (x float32, y float32, z float32, orient float32, ok bool) {
	if len(msg) != 67 {
		return 0, 0, 0, 0, false
	}
	if msg[38] != 4 || msg[43] != 4 || msg[48] != 4 || msg[53] != 4 {
		return 0, 0, 0, 0, false
	}
	x = math.Float32frombits(binary.LittleEndian.Uint32(msg[39:43]))
	y = math.Float32frombits(binary.LittleEndian.Uint32(msg[44:48]))
	z = math.Float32frombits(binary.LittleEndian.Uint32(msg[49:53]))
	orient = math.Float32frombits(binary.LittleEndian.Uint32(msg[54:58]))
	if !float32Finite(x) || !float32Finite(y) || !float32Finite(z) || !float32Finite(orient) {
		return 0, 0, 0, 0, false
	}
	if math.Abs(float64(x)) > 10000 || math.Abs(float64(y)) > 3000 || math.Abs(float64(z)) > 10000 || math.Abs(float64(orient)) > 20 {
		return 0, 0, 0, 0, false
	}
	return x, y, z, orient, true
}
func float32Finite(value float32) bool {
	v := float64(value)
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
