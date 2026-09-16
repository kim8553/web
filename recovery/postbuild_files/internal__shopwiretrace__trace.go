// Package shopwiretrace formats read-only, bounded observations of decoded
// client custom messages. A wire observation never authorizes a handler.
package shopwiretrace

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

const MaxValues = 12

type Value struct {
	Type  byte
	Int32 int32
	Int64 int64
	Text  string
	Raw   [8]byte
}

type Message struct {
	Opcode     byte
	Selector   int32
	Values     []Value // Includes the first selector value.
	TotalCount int     // Original count when Values is only a prefix.
}

// Format never exposes arbitrary decoded text or object identifiers. Detailed
// mode is an explicit, local opt-in and is limited to a finite prefix of values.
func Format(message Message, detailed bool) string {
	types := make([]string, 0, min(len(message.Values), MaxValues))
	details := make([]string, 0, min(len(message.Values), MaxValues))
	for i, v := range message.Values {
		if i >= MaxValues {
			break
		}
		types = append(types, fmt.Sprintf("%d", v.Type))
		if !detailed || i == 0 {
			continue
		}
		switch v.Type {
		case 2:
			details = append(details, fmt.Sprintf("%d:i32=%d", i, v.Int32))
		case 3:
			details = append(details, fmt.Sprintf("%d:i64=%s", i, fingerprint(fmt.Sprintf("%d", v.Int64))))
		case 6, 7:
			if safeShopID(v.Text) {
				details = append(details, fmt.Sprintf("%d:shopid=%s", i, v.Text))
			} else {
				details = append(details, fmt.Sprintf("%d:textlen=%d,sha256_8=%s", i, len(v.Text), fingerprint(v.Text)))
			}
		case 8:
			details = append(details, fmt.Sprintf("%d:object_sha256_8=%s", i, fingerprint(string(v.Raw[:]))))
		default:
			details = append(details, fmt.Sprintf("%d:type=%d", i, v.Type))
		}
	}
	count := len(message.Values)
	if message.TotalCount > count {
		count = message.TotalCount
	}
	suffix := ""
	if count > MaxValues {
		suffix = fmt.Sprintf(" omitted=%d", count-MaxValues)
	}
	line := fmt.Sprintf("SHOP_WIRE_OBSERVE opcode=0x%02X selector=%d value_count=%d types=[%s]%s", message.Opcode, message.Selector, count, strings.Join(types, ","), suffix)
	if detailed {
		line += fmt.Sprintf(" detail=[%s]", strings.Join(details, ";"))
	}
	return line
}

func fingerprint(s string) string {
	hash := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", hash[:8])
}

func safeShopID(s string) bool {
	if len(s) < len("Shop_")+1 || len(s) > 80 || !strings.EqualFold(s[:len("Shop_")], "Shop_") {
		return false
	}
	for _, b := range []byte(s) {
		if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_' || b == '-' {
			continue
		}
		return false
	}
	return true
}
