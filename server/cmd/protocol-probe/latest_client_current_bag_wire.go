package main

import (
	"encoding/binary"
	"fmt"

	"github.com/local/9yin-go-server/internal/clientdata"
)

func latestClientCurrentBagContainer(category int32) (uint16, bool) {
	switch category {
	case 1:
		return 2, true
	case 2:
		return 121, true
	case 3:
		return 123, true
	case 4:
		return 125, true
	default:
		return 0, false
	}
}

func latestClientCurrentBagOrdinal(name string, typ clientdata.WireType) (uint16, error) {
	ordinal, ok := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name: name, typ: typ}]
	if !ok {
		return 0, fmt.Errorf("latest-client bag property %s/%s is not negotiated", name, typ)
	}
	return ordinal, nil
}

func latestClientViewAddSingle(viewID, objectIndex uint16, property serverViewProperty) ([]byte, error) {
	msg := make([]byte, 7)
	msg[0] = 0x18
	binary.LittleEndian.PutUint16(msg[1:3], viewID)
	binary.LittleEndian.PutUint16(msg[3:5], objectIndex)
	binary.LittleEndian.PutUint16(msg[5:7], 1)
	if err := appendViewProperty(&msg, property); err != nil {
		return nil, err
	}
	return msg, nil
}

func latestClientViewObjectPropertySingle(viewID, objectIndex uint16, property serverViewProperty) ([]byte, error) {
	msg := make([]byte, 12)
	msg[0] = 0x10
	msg[1] = 0x01
	binary.LittleEndian.PutUint32(msg[2:6], uint32(viewID))
	binary.LittleEndian.PutUint32(msg[6:10], uint32(objectIndex))
	binary.LittleEndian.PutUint16(msg[10:12], 1)
	if err := appendViewProperty(&msg, property); err != nil {
		return nil, err
	}
	return msg, nil
}

func latestClientViewAddCounted(viewID, objectIndex uint16, properties []serverViewProperty) ([]byte, error) {
	if len(properties) > 0xffff {
		return nil, fmt.Errorf("latest-client VIEW_ADD property count=%d", len(properties))
	}
	msg := make([]byte, 7)
	msg[0] = 0x18
	binary.LittleEndian.PutUint16(msg[1:3], viewID)
	binary.LittleEndian.PutUint16(msg[3:5], objectIndex)
	binary.LittleEndian.PutUint16(msg[5:7], uint16(len(properties)))
	for _, property := range properties {
		if err := appendViewProperty(&msg, property); err != nil {
			return nil, err
		}
	}
	return msg, nil
}

func latestClientCurrentBagFrames(viewID, objectIndex uint16, category int32, properties []serverViewProperty) ([][]byte, error) {
	wireProperties := latestClientBagWireProperties(properties)
	if len(wireProperties) == 0 {
		return nil, fmt.Errorf("latest-client bag object has no negotiated properties")
	}

	var haveItemType, haveViewID bool
	for _, property := range wireProperties {
		switch property.index {
		case 205: // ItemType/int32 in the original negotiated table
			haveItemType = property.int32 != nil
		case 228: // append-only ViewID/int32
			if property.int32 != nil {
				haveViewID = true
				if *property.int32 != category {
					return nil, fmt.Errorf("latest-client bag ViewID=%d does not match category=%d", *property.int32, category)
				}
			}
		}
	}
	if !haveItemType || !haveViewID {
		return nil, fmt.Errorf("latest-client bag object missing required fields ItemType=%v ViewID=%v", haveItemType, haveViewID)
	}

	add, err := latestClientViewAddCounted(viewID, objectIndex, wireProperties)
	if err != nil {
		return nil, err
	}
	return [][]byte{add}, nil
}
