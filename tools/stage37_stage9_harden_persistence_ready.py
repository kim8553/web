from pathlib import Path

main_path = Path('buildtree/cmd/protocol-probe/main.go')
main = main_path.read_text(encoding='utf-8')
old = 'handleShopExchangeContract(link, player, itemCatalog, bagStore != nil && selectedRoleID(selected) != 0, custom, conn.RemoteAddr().String())'
new = 'handleShopExchangeContract(link, player, itemCatalog, currentShopExchangePersistenceReady(selectedRoleID(selected), bagStore), custom, conn.RemoteAddr().String())'
if main.count(old) != 1:
    raise SystemExit(f'stage9 main persistence-ready match count={main.count(old)}')
main = main.replace(old, new, 1)
main_path.write_text(main, encoding='utf-8')
print('stage9 typed-nil-safe persistence readiness wired')
