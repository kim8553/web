package main

import "github.com/local/9yin-go-server/internal/clientdata"

type latestClientPlayerPropertyKey struct {
	name string
	typ  clientdata.WireType
}

var latestClientPlayerWirePropertyOrdinals, latestClientPlayerWirePropertyTableCount = func() (map[latestClientPlayerPropertyKey]uint16, int) {
	fields := sceneVisiblePropertyFields(clientdata.VisibleNPCModernV1().Fields)
	ordinals := make(map[latestClientPlayerPropertyKey]uint16, len(fields))
	for ordinal, field := range fields {
		key := latestClientPlayerPropertyKey{name: field.Name, typ: field.Type}
		// The 0x09 encoder transmits fields in slice order and does not serialize
		// FieldSpec.Index.  When the same name exists with a different wire type
		// (for example Force), the exact wire type disambiguates it.  For an exact
		// duplicate, keep the first negotiated ordinal.
		if _, exists := ordinals[key]; exists {
			continue
		}
		ordinals[key] = uint16(ordinal)
	}
	return ordinals, len(fields)
}()

// latestClientSceneObjectWireProperties converts server/global property IDs to
// ordinal indexes of the exact table Stage37 sends in opcode 0x09. FxNet2 uses
// that negotiated table for non-view ServerObjectProperty updates regardless of
// whether the target object is the main player or an NPC. Only exact
// name+wire-type matches survive; unsupported or type-mismatched legacy fields
// are dropped instead of being guessed onto a current ordinal.
func latestClientSceneObjectWireProperties(properties []clientdata.IndexedProperty) []clientdata.IndexedProperty {
	if len(properties) == 0 {
		return nil
	}
	result := make([]clientdata.IndexedProperty, 0, len(properties))
	seen := make(map[uint16]struct{}, len(properties))
	for _, property := range properties {
		ordinal, ok := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name: property.Name, typ: property.Value.Type}]
		if !ok {
			continue
		}
		if _, duplicate := seen[ordinal]; duplicate {
			continue
		}
		property.Index = ordinal
		result = append(result, property)
		seen[ordinal] = struct{}{}
	}
	return result
}

// latestClientPlayerWireProperties is retained for player-construction call
// sites; player and NPC non-view updates share the same negotiated table.
func latestClientPlayerWireProperties(properties []clientdata.IndexedProperty) []clientdata.IndexedProperty {
	return latestClientSceneObjectWireProperties(properties)
}
