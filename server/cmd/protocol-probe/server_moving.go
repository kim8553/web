package main

import (
	"encoding/binary"
	worldcore "github.com/local/9yin-go-server/internal/world"
	"math"
)

type MotionState struct {
	DestX, DestY, DestZ, DestOrient    float32
	Motion0, Motion1, Motion2, Motion3 float32
	Extra                              uint32
}

func motionFromDest(dest worldcore.Transform) MotionState {
	return MotionState{DestX: dest.X, DestY: dest.Y, DestZ: dest.Z, DestOrient: dest.Orient}
}
func serverMoving(objectID, ownerID uint32, state MotionState) []byte {
	payload := make([]byte, 45)
	payload[0] = 0x20
	binary.LittleEndian.PutUint32(payload[1:], objectID)
	binary.LittleEndian.PutUint32(payload[5:], ownerID)
	binary.LittleEndian.PutUint32(payload[9:], math.Float32bits(state.DestX))
	binary.LittleEndian.PutUint32(payload[13:], math.Float32bits(state.DestY))
	binary.LittleEndian.PutUint32(payload[17:], math.Float32bits(state.DestZ))
	binary.LittleEndian.PutUint32(payload[21:], math.Float32bits(state.DestOrient))
	binary.LittleEndian.PutUint32(payload[25:], math.Float32bits(state.Motion0))
	binary.LittleEndian.PutUint32(payload[29:], math.Float32bits(state.Motion1))
	binary.LittleEndian.PutUint32(payload[33:], math.Float32bits(state.Motion2))
	binary.LittleEndian.PutUint32(payload[37:], math.Float32bits(state.Motion3))
	binary.LittleEndian.PutUint32(payload[41:], state.Extra)
	return payload
}
