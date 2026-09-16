package main

// serverDeleteViewCompat encodes the historically verified SERVER_DELETE_VIEW
// layout: opcode 0x16 followed by a little-endian uint16 view ID.  It is used
// only by the latest-client live-bag refresh A/B and is not exact-authority code.
func serverDeleteViewCompat(viewID uint16) []byte {
	return []byte{0x16, byte(viewID), byte(viewID >> 8)}
}
