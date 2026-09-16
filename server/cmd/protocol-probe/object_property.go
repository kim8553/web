package main

import (
	"encoding/binary"
	"fmt"
	"github.com/local/9yin-go-server/internal/clientdata"
)

func sceneObjectProperties(objectID, ownerID uint32, isViewObject byte, properties []clientdata.IndexedProperty) ([]byte, error) {
	if isViewObject == 0 {
		properties = latestClientSceneObjectWireProperties(properties)
	}
	if len(properties) > 0xffff {
		return nil, fmt.Errorf("object property count %d overflows u16", len(properties))
	}
	msg := make([]byte, 12, 12+len(properties)*8)
	msg[0] = 0x10
	msg[1] = isViewObject
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], uint16(len(properties)))
	for _, property := range properties {
		before := len(msg)
		msg = appendNPCProperty(msg, property)
		if len(msg) == before {
			return nil, fmt.Errorf("empty encode for property index %d", property.Index)
		}
	}
	return msg, nil
}
