package main

import (
	"log"
	"os"
	"sync/atomic"

	"github.com/local/9yin-go-server/internal/shopwiretrace"
)

const maxDetailedShopWireMessages uint64 = 8192

var detailedShopWireCount atomic.Uint64

// traceShopWireCustom is observation only: never returns a verified selector,
// never mutates a player/store and never replies to the client. The current
// ordinary purchase selector must be independently confirmed before wiring.
func traceShopWireCustom(custom clientCustomMessage, remote string) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 {
		return
	}
	detailed := os.Getenv("NINEYIN_SHOP_WIRE_TRACE") == "1"
	if detailed {
		n := detailedShopWireCount.Add(1)
		if n > maxDetailedShopWireMessages {
			detailed = false
			if n == maxDetailedShopWireMessages+1 {
				log.Printf("%s: SHOP_WIRE_TRACE_DETAIL_LIMIT=%d; subsequent observations are type-only", remote, maxDetailedShopWireMessages)
			}
		}
	}
	values := make([]shopwiretrace.Value, 0, min(len(custom.Values), shopwiretrace.MaxValues))
	for _, value := range custom.Values {
		if len(values) >= shopwiretrace.MaxValues {
			break
		}
		values = append(values, shopwiretrace.Value{Type: value.Type, Int32: value.Int32, Int64: value.Int64, Text: value.Text, Raw: value.Raw})
	}
	// Preserve the actual count for the format's omission accounting without
	// retaining a full copy of large or sensitive client strings.
	message := shopwiretrace.Message{Opcode: custom.Opcode, Selector: custom.Values[0].Int32, Values: values, TotalCount: len(custom.Values)}
	log.Printf("%s: %s", remote, shopwiretrace.Format(message, detailed))
}
