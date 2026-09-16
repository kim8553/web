#!/usr/bin/env python3
from pathlib import Path

path = Path("buildtree/cmd/protocol-probe/zz_recovered_overlay.go")
text = path.read_text(encoding="utf-8")
start_marker = "func handleShopBuyCustom("
end_marker = "\nfunc shopCatalogItems("
start = text.find(start_marker)
if start < 0:
    raise SystemExit("Stage21: handleShopBuyCustom not found")
end = text.find(end_marker, start)
if end < 0:
    raise SystemExit("Stage21: shopCatalogItems boundary not found")
old = text[start:end]

required_unsafe = [
    "player.addGold(",
    "player.addSilver(",
    "player.addSilverCard(",
    "player.addBagItem(",
    "writeFrames(",
    "persistBagEquip(",
    "currencyStore.Save(",
]
missing = [token for token in required_unsafe if token not in old]
if missing:
    raise SystemExit(f"Stage21: recovered unsafe handler drift; missing anchors: {missing}")
if "custom.Values[0].Int32 != 0x46" not in old:
    raise SystemExit("Stage21: recovered candidate selector anchor 0x46 missing")

brace = old.find("{")
if brace < 0:
    raise SystemExit("Stage21: function opening brace missing")
signature = old[:brace + 1]
replacement = signature + r'''
	// Stage21 fail-closed safety barrier. The recovered compatibility handler
	// used candidate selector 0x46/70, but exact-current ordinary-shop wire
	// authority is still not present in the canonical repository. The previous
	// body mutated live currency/bag, published frames before persistence, and
	// saved bag/currency independently. Never mutate on an unverified selector.
	if len(custom.Values) < 1 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != 0x46 {
		return false, nil
	}
	log.Printf("%s: ordinary shop candidate selector=0x46 blocked: exact-current wire authority unavailable; legacy mutation route disabled", remote)
	// Return false so this candidate does not claim authoritative handling; the
	// caller currently ignores the boolean and may continue normal dispatch.
	return false, nil
}'''

new_text = text[:start] + replacement + text[end:]
path.write_text(new_text, encoding="utf-8")
print("stage21 disabled recovered unverified ordinary-shop mutation route")
