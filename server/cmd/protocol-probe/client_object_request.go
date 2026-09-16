package main

import "encoding/binary"

type clientObjectRequest struct {
	Sequence uint32
	ObjectID uint32
	OwnerID  uint32
	FuncID   int32
}

func parseClientObjectRequest(message []byte) (clientObjectRequest, bool) {
	if len(message) < 29 || message[0] != 0x07 {
		return clientObjectRequest{}, false
	}
	request := clientObjectRequest{Sequence: binary.LittleEndian.Uint32(message[17:]), ObjectID: binary.LittleEndian.Uint32(message[21:]), OwnerID: binary.LittleEndian.Uint32(message[25:]), FuncID: int32(binary.LittleEndian.Uint32(message[29:]))}
	if request.ObjectID == 0 || request.OwnerID == 0 {
		return clientObjectRequest{}, false
	}
	return request, true
}
