#!/usr/bin/env python3
"""Extract exact source blocks for resource-independent Stage27 tests.

This does not stub a game service or substitute old client resources. It copies
unaltered production declarations into a temporary package for unit tests.
"""
from pathlib import Path
import re
import sys

source = Path(sys.argv[1])
test = Path(sys.argv[2])
destination = Path(sys.argv[3])
destination.mkdir(parents=True, exist_ok=True)


def source_block(path: Path, needle: str) -> str:
    data = path.read_text()
    assert data.count(needle) == 1, (str(path), needle, data.count(needle))
    start = data.index(needle)
    opening = data.index("{", start)
    depth = 0
    for index in range(opening, len(data)):
        if data[index] == "{":
            depth += 1
        elif data[index] == "}":
            depth -= 1
            if depth == 0:
                return data[start : index + 1]
    raise ValueError(f"unterminated Go declaration: {needle}")


shop = source.parent / "shop_catalog.go"
view = source.parent / "latest_client_shop_view_contract.go"
page_size = re.findall(r"^const currentShopPageSize int32 = 500$", view.read_text(), re.M)
assert len(page_size) == 1
chunks = [
    "package main",
    page_size[0],
    source_block(shop, "type shopCatalogItem struct {"),
    source_block(view, "func currentShopViewObjectIndex(item shopCatalogItem) (uint16, bool) {"),
    source_block(source, "type stage27ShopDisplayRows struct {"),
    source_block(source, "func summarizeStage27ShopDisplayRows(items []shopCatalogItem) stage27ShopDisplayRows {"),
]
(destination / "stage27_isolated.go").write_text("\n\n".join(chunks) + "\n")
(destination / "stage27_isolated_test.go").write_bytes(test.read_bytes())
print("stage27_isolation=EXTRACTED_VERBATIM_FROM_VERIFIED_SOURCE")
