# NPC creator manifest filename-case fix (2026-09-19)

## Scope and source authority

- Original Go lineage remains user-provided `9yin-go-server1.rar`. Edited base is `kim8553/web` branch `stage37-restored-20260918`, GitHub commit `97ddf19b31ea3b41feab623e8f9d7d65afb27827`, tree `c850bee9159c7c33d6316c345870e8dbbd6aa543`. Its source was reconstructed in an isolated sandbox from the previously verified `d904455` tree `146eb03755b493de4bbfa09bc78c97b52088f069` and a SHA-pinned patch; the resulting tree matched the GitHub tree exactly before changes.
- The bundled `server/resources/modern/share/creator/npc_creator` files were the inputs for this regression. This does **not** establish that those files exactly match the user's currently installed latest client or Google Drive `res` packages. No client protocol, shop sale price or gameplay behavior is inferred from them.

## Reproduction

- On Linux, three existing full-suite tests failed: `TestCurrentCitySceneRegistryMaterializesModernCreatorCatalog`, `TestShenJiHuiCatalogSkipsMissingOptionalCreators`, `TestYanYuZhuangCatalogSkipsEmptyZeroAmountCreatorPlaceholder`. Their `file.ini` entries such as `CommonNpc.xml` refer to installed `commonnpc.xml`; case-sensitive `os.Stat` silently skipped those entries and reported no creator.
- A read-only inventory of 288 bundled scene `file.ini` manifests found 1,477 case-only filename differences, 98 absent optional XML entries and zero case-colliding actual filenames. These are counts for this source tree only, not an audit of every current-client resource.
- New tests for case-only filename resolution and ambiguity rejection both **FAIL before** the change and **PASS after** it.

## Minimum code change

- `sceneCreatorPaths` continues to honor an exact file path first, but when that path does not exist, it resolves a simple manifest XML basename against actual directory entries using case-insensitive equality. No invented creator path is returned: it must exist on disk. Missing optional XML entries stay skipped. If more than one actual filename matches the same basename case-insensitively, return an explicit ambiguity error rather than choose one. Deduplication still uses the final resolved path.
- Original manifest content, bundled XML, other NPC creation logic, shop purchases, currencies, database schema and any C2S IDs are unchanged. This fixes host-filesystem case handling, not NPC gameplay correctness.

## Verified checks and limitations

- Exact base source tree matched `c850bee9159c7c33d6316c345870e8dbbd6aa543` before edit.
- Isolated sandbox with Go 1.23.2, verified vendored dependencies and a temporary alternate module file outside the repository: `go test ./... -count=1` **PASS** after change (previously **FAIL** on three tests); targeted shop/bag/NPC/creator tests **PASS**; `go vet ./...` **PASS**; `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/protocol-probe` **PASS**. Original tracked `go.mod` remains unchanged.
- Windows binary was built but not launched in the user's environment. User MySQL, latest-client live purchase/sale, scene arrival, Bag/View and relogin **NOT RUN**. The NPC sale packet and settlement remain unknown; GM web grant and mode3 Exchange remain deferred.
