#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
BUILD="$ROOT/buildtree"

# Stage15 is deliberately read-only with respect to production gameplay code.
# It records exactly what the current reconstructed server accepts for ordinary
# NPC shop purchases, whether the exact current-client authority files are
# present in this repository, and whether the existing handler mutates live
# state before durable persistence.  It must not promote a historical selector
# or a server-side literal into an exact-current client contract.
bash tools/stage37_current_recovery_stage14.sh

AUDIT="$(mktemp /tmp/stage15_regular_shop_wire.XXXXXX.go)"
cat > "$AUDIT" <<'EOF'
package main

import (
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
    "io/fs"
    "os"
    "path/filepath"
    "sort"
    "strings"
)

type funcHit struct {
    path string
    line int
    src  string
}

func source(raw []byte, fset *token.FileSet, n ast.Node) string {
    a := fset.Position(n.Pos()).Offset
    b := fset.Position(n.End()).Offset
    if a < 0 || b <= a || b > len(raw) { return "" }
    return string(raw[a:b])
}

func main() {
    if len(os.Args) != 3 {
        fmt.Fprintln(os.Stderr, "usage: audit <repo-root> <source-root>")
        os.Exit(2)
    }
    repoRoot, _ := filepath.Abs(os.Args[1])
    sourceRoot, _ := filepath.Abs(os.Args[2])

    fset := token.NewFileSet()
    var handlers []funcHit
    err := filepath.WalkDir(sourceRoot, func(path string, d fs.DirEntry, walkErr error) error {
        if walkErr != nil { return walkErr }
        if d.IsDir() {
            if d.Name() == ".git" { return filepath.SkipDir }
            return nil
        }
        if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") { return nil }
        raw, err := os.ReadFile(path); if err != nil { return err }
        f, err := parser.ParseFile(fset, path, raw, parser.ParseComments); if err != nil { return err }
        rel, _ := filepath.Rel(sourceRoot, path)
        for _, decl := range f.Decls {
            fd, ok := decl.(*ast.FuncDecl)
            if !ok || fd.Name == nil || fd.Name.Name != "handleShopBuyCustom" { continue }
            handlers = append(handlers, funcHit{rel, fset.Position(fd.Pos()).Line, source(raw, fset, fd)})
        }
        return nil
    })
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(2) }
    sort.Slice(handlers, func(i,j int) bool { if handlers[i].path == handlers[j].path { return handlers[i].line < handlers[j].line }; return handlers[i].path < handlers[j].path })

    fmt.Printf("handleShopBuyCustom_decl_count=%d\n", len(handlers))
    if len(handlers) != 1 {
        fmt.Fprintln(os.Stderr, "expected exactly one non-test handleShopBuyCustom declaration")
        os.Exit(1)
    }
    h := handlers[0]
    fmt.Printf("handler=%s:%d\n", h.path, h.line)

    candidate46 := strings.Contains(h.src, "Int32 != 0x46") || strings.Contains(h.src, "Int32 != 70")
    payloadShape := strings.Contains(h.src, "custom.Values[1].Type != 6") &&
        strings.Contains(h.src, "custom.Values[2].Type != 2") &&
        strings.Contains(h.src, "custom.Values[3].Type != 2") &&
        strings.Contains(h.src, "custom.Values[4].Type != 2") &&
        strings.Contains(h.src, "shopID := custom.Values[1].Text") &&
        strings.Contains(h.src, "page := custom.Values[2].Int32") &&
        strings.Contains(h.src, "pos := custom.Values[3].Int32") &&
        strings.Contains(h.src, "amount := custom.Values[4].Int32")

    fmt.Printf("server_candidate_selector_0x46=%t\n", candidate46)
    fmt.Printf("server_candidate_payload_shopid_page_pos_amount=%t\n", payloadShape)
    if candidate46 { fmt.Println("candidate_selector=0x46") } else { fmt.Println("candidate_selector=UNKNOWN") }
    if payloadShape { fmt.Println("candidate_payload=shopid:string,page:int32,pos:int32,amount:int32") } else { fmt.Println("candidate_payload=UNKNOWN") }

    debit := strings.Contains(h.src, "player.addGold(") || strings.Contains(h.src, "player.addSilver(") || strings.Contains(h.src, "player.addSilverCard(")
    bagApply := strings.Contains(h.src, "player.addBagItem(")
    clientWrite := strings.Contains(h.src, "writeFrames(")
    bagPersist := strings.Contains(h.src, "persistBagEquip(")
    currencyPersist := strings.Contains(h.src, "currencyStore.Save(")
    fmt.Printf("existing_live_currency_mutation=%t\n", debit)
    fmt.Printf("existing_live_bag_mutation=%t\n", bagApply)
    fmt.Printf("existing_client_frame_write=%t\n", clientWrite)
    fmt.Printf("existing_bag_persistence=%t\n", bagPersist)
    fmt.Printf("existing_currency_persistence=%t\n", currencyPersist)

    writeAt := strings.Index(h.src, "writeFrames(")
    bagPersistAt := strings.Index(h.src, "persistBagEquip(")
    currencyPersistAt := strings.Index(h.src, "currencyStore.Save(")
    clientBeforePersistence := writeAt >= 0 && bagPersistAt >= 0 && currencyPersistAt >= 0 && writeAt < bagPersistAt && writeAt < currencyPersistAt
    separatePersistence := bagPersist && currencyPersist && !strings.Contains(h.src, "BeginTx(") && !strings.Contains(h.src, "BeginTxx(") && !strings.Contains(h.src, ".Begin(")
    fmt.Printf("client_frames_before_bag_and_currency_persistence=%t\n", clientBeforePersistence)
    fmt.Printf("bag_currency_persistence_separate_in_handler=%t\n", separatePersistence)

    // Presence is not verification: even when an authority asset appears in the
    // repository, a later stage must actually analyze its sender/xref before it
    // may set current_client_selector_verified=1.
    wanted := map[string]bool{
        "fxgame.exe": true,
        "fxgamelogic.dll": true,
        "fxnet2.dll": true,
        "fxcore.dll": true,
        "form_shop.lua": true,
        "lua64.package": true,
    }
    var assets []string
    _ = filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, walkErr error) error {
        if walkErr != nil { return walkErr }
        if d.IsDir() {
            if d.Name() == ".git" || filepath.Clean(path) == filepath.Clean(sourceRoot) { return filepath.SkipDir }
            return nil
        }
        if wanted[strings.ToLower(d.Name())] {
            rel, _ := filepath.Rel(repoRoot, path)
            assets = append(assets, rel)
        }
        return nil
    })
    sort.Strings(assets)
    fmt.Printf("exact_current_authority_asset_hits=%d\n", len(assets))
    for _, p := range assets { fmt.Printf("authority_asset=%s\n", p) }
    fmt.Println("current_client_selector_verified=0")
    if len(assets) == 0 {
        fmt.Println("wire_gate_status=BLOCKED_NO_EXACT_CURRENT_CLIENT_AUTHORITY_IN_REPO")
    } else {
        fmt.Println("wire_gate_status=BLOCKED_AUTHORITY_PRESENT_BUT_SENDER_XREF_NOT_YET_PROVEN")
    }

    // These are findings, not a protocol failure. The audit exits zero so the
    // rest of Stage37 build/race/vet evidence remains independently meaningful.
    if !candidate46 || !payloadShape {
        fmt.Fprintln(os.Stderr, "canonical ordinary-shop candidate changed; inspect before continuing")
        os.Exit(3)
    }
    if !(debit && bagApply && clientWrite && bagPersist && currencyPersist) {
        fmt.Fprintln(os.Stderr, "canonical ordinary-shop mutation topology changed; inspect before continuing")
        os.Exit(4)
    }
}
EOF

set +e
go run "$AUDIT" "$ROOT" "$BUILD" > "$ROOT/regular_shop_wire_gate.log" 2>&1
rc=$?
set -e
rm -f "$AUDIT"
echo "$rc" > "$ROOT/regular_shop_wire_gate.exit"
echo "regular_shop_wire_gate=$rc"
cat "$ROOT/regular_shop_wire_gate.log"
if [ "$rc" -ne 0 ]; then exit 97; fi

source_files=$(find "$BUILD" -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' | wc -l | tr -d ' ')
manifest_sha=$(awk '{print $1}' "$ROOT/generated-manifest.sha256")
echo "stage15_source_files=$source_files"
echo "stage15_manifest_sha256=$manifest_sha"
