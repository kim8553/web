package main

import (
	"fmt"

	"github.com/local/9yin-go-server/internal/clientdata"
)

// latestClientViewPropertyFields is the exact ordinal/type table Stage37 sends
// in opcode 0x09. The latest FxNet2 ServerCreateView and ServerViewAdd handlers
// both dispatch their property lists through the same decoder, which rejects an
// index >= this table length and decodes the value using the type at that
// ordinal. Validate locally so a stale global property ID cannot reach the
// client as a property error.
var latestClientViewPropertyFields = sceneVisiblePropertyFields(clientdata.VisibleNPCModernV1().Fields)

func latestClientServerViewPropertyType(property serverViewProperty) (clientdata.WireType, bool) {
	switch {
	case property.byte1 != nil:
		return clientdata.WireByte, true
	case property.text != nil:
		return clientdata.WireString, true
	case property.int32 != nil:
		return clientdata.WireInt32, true
	case property.real32 != nil:
		return clientdata.WireFloat32, true
	default:
		return 0, false
	}
}

func latestClientValidateViewWireProperties(viewID uint16, operation string, properties []serverViewProperty) error {
	for _, property := range properties {
		if int(property.index) >= len(latestClientViewPropertyFields) {
			return fmt.Errorf("%s view %d property ordinal %d outside negotiated count %d", operation, viewID, property.index, len(latestClientViewPropertyFields))
		}
		got, ok := latestClientServerViewPropertyType(property)
		if !ok {
			return fmt.Errorf("%s view %d property ordinal %d has unsupported wire value", operation, viewID, property.index)
		}
		want := latestClientViewPropertyFields[property.index].Type
		if got != want {
			return fmt.Errorf("%s view %d property ordinal %d wire type %s does not match negotiated %s (%s)", operation, viewID, property.index, got, want, latestClientViewPropertyFields[property.index].Name)
		}
	}
	return nil
}
