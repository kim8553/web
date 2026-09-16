from pathlib import Path

root = Path('buildtree/cmd/protocol-probe')
contract_path = root / 'latest_client_shop_exchange_contract.go'
main_path = root / 'main.go'

contract = contract_path.read_text(encoding='utf-8')
old_import = 'import (\n\t"fmt"\n\t"log"\n)'
new_import = 'import (\n\t"fmt"\n\t"log"\n\n\t"github.com/local/9yin-go-server/internal/exchangebinding"\n)'
if contract.count(old_import) != 1:
    raise SystemExit(f'contract import block match count={contract.count(old_import)}')
contract = contract.replace(old_import, new_import, 1)

old_sig = 'func handleShopExchangeContract(link sceneMessageConnection, player *playerActor, itemCatalog *itemCatalog, custom clientCustomMessage, remote string) (bool, error) {'
new_sig = 'func handleShopExchangeContract(link sceneMessageConnection, player *playerActor, itemCatalog *itemCatalog, persistenceReady bool, custom clientCustomMessage, remote string) (bool, error) {'
if contract.count(old_sig) != 1:
    raise SystemExit(f'contract signature match count={contract.count(old_sig)}')
contract = contract.replace(old_sig, new_sig, 1)

needle = '\t\tlog.Printf("%s: current shop exchange buy preflight shop=%s page=%d pos=%d count=%d exchange_data=%d output=%s x%d conditions_supported=%t conditions_satisfied=%t properties_supported=%t properties_satisfied=%t materials_satisfied=%t capacity_supported=%t capacity_satisfied=%t exchange_bind_preview=%d blocked: final persisted BindStatus/atomic mutation unresolved",'
if contract.count(needle) != 1:
    raise SystemExit(f'preflight log anchor match count={contract.count(needle)}')
gate = '''\t\t// Dry-run only: exact-current final persisted BindStatus is still unresolved,\n\t\t// therefore the zero-value binding decision must keep this gate closed.\n\t\t// No mutation or persistence helper is invoked from this handler.\n\t\tvar bindingDecision exchangebinding.Decision\n\t\tmutationGate := currentShopExchangeMutationGate(preflight, bindingDecision, persistenceReady)\n\t\tlog.Printf("%s: current shop exchange mutation gate allowed=%t blocks=%v persistence_ready=%t",\n\t\t\tremote, mutationGate.Allowed(), mutationGate.Blocks(), persistenceReady)\n'''
contract = contract.replace(needle, gate + needle, 1)
contract_path.write_text(contract, encoding='utf-8')

main = main_path.read_text(encoding='utf-8')
old_call = 'handleShopExchangeContract(link, player, itemCatalog, custom, conn.RemoteAddr().String())'
new_call = 'handleShopExchangeContract(link, player, itemCatalog, bagStore != nil && selectedRoleID(selected) != 0, custom, conn.RemoteAddr().String())'
if main.count(old_call) != 1:
    raise SystemExit(f'main call match count={main.count(old_call)}')
main = main.replace(old_call, new_call, 1)
main_path.write_text(main, encoding='utf-8')

print('stage8 exchange gate dry-run wiring applied')
