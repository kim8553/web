package shopwiretrace

import (
	"strings"
	"testing"
)

func TestDefaultLogsOnlyTypeShape(t *testing.T) {
	m := Message{Opcode: 0x1e, Selector: 70, Values: []Value{{Type: 2, Int32: 70}, {Type: 6, Text: "Shop_ZH_001"}, {Type: 2, Int32: 3}, {Type: 7, Text: "private password"}}}
	got := Format(m, false)
	if !strings.Contains(got, "selector=70 value_count=4 types=[2,6,2,7]") {
		t.Fatalf("missing shape: %s", got)
	}
	if strings.Contains(got, "Shop_ZH_001") || strings.Contains(got, "password") || strings.Contains(got, "i32=3") {
		t.Fatalf("default leaked values: %s", got)
	}
}

func TestOptInPreservesShopIDButNotArbitraryText(t *testing.T) {
	m := Message{Opcode: 0x27, Selector: 345, Values: []Value{{Type: 2, Int32: 345}, {Type: 6, Text: "Shop_ZH_001"}, {Type: 2, Int32: 2}, {Type: 2, Int32: 7}, {Type: 6, Text: "password=secret\n"}, {Type: 8, Raw: [8]byte{1, 2, 3}}}}
	got := Format(m, true)
	for _, token := range []string{"selector=345", "1:shopid=Shop_ZH_001", "2:i32=2", "3:i32=7", "4:textlen=16,sha256_8=", "5:object_sha256_8="} {
		if !strings.Contains(got, token) {
			t.Fatalf("missing %q in %s", token, got)
		}
	}
	if strings.Contains(got, "secret") || strings.Contains(got, "password") || strings.Contains(got, "\n") {
		t.Fatalf("sensitive content leaked: %s", got)
	}
}

func TestLongAndUnsafeShopIDMaskedAndLimit(t *testing.T) {
	m := Message{Opcode: 0x1e, Selector: 1, Values: make([]Value, MaxValues+5)}
	for i := range m.Values {
		m.Values[i] = Value{Type: 6, Text: "Shop_BAD\nsecret"}
	}
	got := Format(m, true)
	if !strings.Contains(got, "omitted=5") || strings.Contains(got, "secret") || strings.Contains(got, "shopid=Shop_BAD") {
		t.Fatalf("invalid redaction or cap: %s", got)
	}
	if strings.Count(got, "textlen=") != MaxValues-1 {
		t.Fatalf("wrong detail count: %s", got)
	}
}

func TestEmptyAndInt64Redacted(t *testing.T) {
	if got := Format(Message{Opcode: 0x1e}, false); !strings.Contains(got, "value_count=0 types=[]") {
		t.Fatal(got)
	}
	got := Format(Message{Opcode: 0x1e, Selector: 123, Values: []Value{{Type: 2, Int32: 123}, {Type: 3, Int64: 123456789}}}, true)
	if strings.Contains(got, "123456789") || !strings.Contains(got, "i64=") {
		t.Fatal(got)
	}
}
