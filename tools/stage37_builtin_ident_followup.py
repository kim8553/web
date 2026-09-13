#!/usr/bin/env python3
# Apply after stage37_latest_client_legacy_bag_rich_object_patch.py.
# Current FxNet2 derives built-in read-only Ident from VIEW_ADD.objectIndex;
# historical 0x0765/UniqueID must not be negotiated/emitted as Ident.
from pathlib import Path
import sys


def r1(text, old, new, label):
    n = text.count(old)
    if n != 1:
        raise SystemExit(f"{label}: expected 1 anchor, found {n}")
    return text.replace(old, new, 1)


def main():
    if len(sys.argv) != 2:
        raise SystemExit("usage: stage37_builtin_ident_followup.py <buildtree>")
    p = Path(sys.argv[1]) / "cmd" / "protocol-probe"

    f = p / "main.go"
    s = f.read_text(encoding="utf-8")
    s = r1(s, '\thaveBagIdent := false\n', '', 'remove haveBagIdent')
    s = r1(s, '''\t\tif field.Name == "Ident" && field.Type == clientdata.WireString {\n\t\t\thaveBagIdent = true\n\t\t}\n''', '', 'remove Ident discovery')
    s = r1(s, '''\tif !haveBagIdent {\n\t\tfields = append(fields, clientdata.FieldSpec{Index: 0x0765, Name: "Ident", Type: clientdata.WireString})\n\t}\n''', '', 'remove Ident append')
    f.write_text(s, encoding="utf-8")

    f = p / "latest_client_current_bag_wire.go"
    s = f.read_text(encoding="utf-8")
    s = r1(s, 'var haveItemType, haveViewID, haveIdent bool', 'var haveItemType, haveViewID bool', 'frame flags')
    s = r1(s, '''\t\tcase 229: // append-only Ident/string\n\t\t\thaveIdent = property.text != nil && *property.text != ""\n''', '', 'frame Ident case')
    s = r1(s, '''\tif !haveItemType || !haveViewID || !haveIdent {\n\t\treturn nil, fmt.Errorf("latest-client bag object missing required fields ItemType=%v ViewID=%v Ident=%v", haveItemType, haveViewID, haveIdent)\n\t}\n''', '''\tif !haveItemType || !haveViewID {\n\t\treturn nil, fmt.Errorf("latest-client bag object missing required fields ItemType=%v ViewID=%v", haveItemType, haveViewID)\n\t}\n''', 'frame required fields')
    f.write_text(s, encoding="utf-8")

    f = p / "latest_client_bag_view_ordinal_compat.go"
    s = f.read_text(encoding="utf-8")
    s = r1(s, '''\tvar ident string\n\tvar haveIdent bool\n''', '', 'normalizer Ident vars')
    s = r1(s, '''\t\tcase 229, 0x0765:\n\t\t\tif property.text != nil {\n\t\t\t\tident = *property.text\n\t\t\t\thaveIdent = ident != ""\n\t\t\t}\n''', '', 'normalizer Ident case')
    s = r1(s, '''\t// Emit in ascending negotiated ordinal order. Existing 0..227 positions are\n\t// unchanged; ViewID and Ident are the two append-only fields.\n\tresult := make([]serverViewProperty, 0, 7)\n''', '''\t// Emit in ascending negotiated ordinal order. Existing 0..227 positions are\n\t// unchanged; ViewID is the only append-only field. Ident is built-in and\n\t// derived by FxNet2 from VIEW_ADD.objectIndex.\n\tresult := make([]serverViewProperty, 0, 6)\n''', 'normalizer comment/capacity')
    s = r1(s, '''\tif haveIdent {\n\t\tresult = append(result, viewString(229, ident))\n\t}\n''', '', 'normalizer Ident emit')
    f.write_text(s, encoding="utf-8")

    f = p / "latest_client_player_property_ordinal_compat_test.go"
    s = f.read_text(encoding="utf-8")
    s = r1(s, 'latestClientPlayerWirePropertyTableCount != 230', 'latestClientPlayerWirePropertyTableCount != 229', 'table count condition')
    s = r1(s, 'want 230 after append-only ViewID+Ident', 'want 229 after append-only ViewID', 'table count message')
    f.write_text(s, encoding="utf-8")

    f = p / "latest_client_current_bag_wire_test.go"
    s = f.read_text(encoding="utf-8")
    s = r1(s, 'latestClientPlayerWirePropertyTableCount != 230 { t.Fatalf("property table=%d, want 230"', 'latestClientPlayerWirePropertyTableCount != 229 { t.Fatalf("property table=%d, want 229"', 'rich table count')
    s = r1(s, '''\tif got := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name:"Ident", typ:clientdata.WireString}]; got != 229 { t.Fatalf("Ident/string ordinal=%d, want 229", got) }\n''', '''\tif _, ok := latestClientPlayerWirePropertyOrdinals[latestClientPlayerPropertyKey{name:"Ident", typ:clientdata.WireString}]; ok { t.Fatal("Ident must be built-in/read-only and not negotiated") }\n''', 'rich table Ident assertion')
    s = r1(s, '''\tident := "7100568-001-0000000001-0001"\n''', '', 'atomic ident local')
    s = r1(s, 'viewString(0x0765, ident),', 'viewString(0x0765, "7100568-001-0000000001-0001"), // historical UniqueID source; must stay off wire', 'atomic UniqueID input')
    s = r1(s, 'binary.LittleEndian.Uint16(f[5:7]) != 7', 'binary.LittleEndian.Uint16(f[5:7]) != 6', 'atomic property count')
    s = r1(s, 'want := []uint16{7, 105, 106, 110, 205, 228, 229}', 'want := []uint16{7, 105, 106, 110, 205, 228}', 'atomic ordinals')
    s = r1(s, 'case 7, 105, 229:', 'case 7, 105:', 'atomic string ordinals')
    f.write_text(s, encoding="utf-8")

    f = p / "latest_client_bag_view_ordinal_compat_test.go"
    s = f.read_text(encoding="utf-8")
    s = r1(s, 'if len(got) != 7 {\n\t\tt.Fatalf("bag wire property count=%d, want 7", len(got))', 'if len(got) != 6 {\n\t\tt.Fatalf("bag wire property count=%d, want 6", len(got))', 'normalizer test count')
    s = r1(s, 'want := []uint16{7, 105, 106, 110, 205, 228, 229}', 'want := []uint16{7, 105, 106, 110, 205, 228}', 'normalizer test ordinals')
    s = r1(s, '''\tfor _, property := range got {\n\t\tif int(property.index) >= latestClientPlayerWirePropertyTableCount {\n''', '''\tfor _, property := range got {\n\t\tif property.index == 229 { t.Fatal("Ident ordinal 229 must not be emitted") }\n\t\tif int(property.index) >= latestClientPlayerWirePropertyTableCount {\n''', 'normalizer no Ident assertion')
    f.write_text(s, encoding="utf-8")

    for f in [
        p / "main.go",
        p / "latest_client_current_bag_wire.go",
        p / "latest_client_bag_view_ordinal_compat.go",
        p / "latest_client_player_property_ordinal_compat_test.go",
        p / "latest_client_current_bag_wire_test.go",
        p / "latest_client_bag_view_ordinal_compat_test.go",
    ]:
        print(f"patched {f}")


if __name__ == "__main__":
    main()
