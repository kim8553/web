#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
BUILD="$ROOT/buildtree"

# Reproduce and verify the Stage11 safety audit first. Stage12 is read-only:
# it inventories the exact reconstructed regular NPC shop-buy path so it can be
# repaired independently from mode3 exchange and independently from deferred GM
# grant functionality.
bash tools/stage37_current_recovery_stage11.sh

AUDIT="$ROOT/stage12_regular_shop_audit_tmp.go"
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
    "strconv"
    "strings"
)

type handlerInfo struct {
    path string
    start int
    end int
    source string
    calls []string
}

func callName(expr ast.Expr) string {
    switch x := expr.(type) {
    case *ast.Ident:
        return x.Name
    case *ast.SelectorExpr:
        left := callName(x.X)
        if left == "" { return x.Sel.Name }
        return left + "." + x.Sel.Name
    case *ast.ParenExpr:
        return callName(x.X)
    }
    return ""
}

func main() {
    if len(os.Args) != 2 {
        fmt.Fprintln(os.Stderr, "usage: audit <source-root>")
        os.Exit(2)
    }
    root, _ := filepath.Abs(os.Args[1])
    fset := token.NewFileSet()
    var handlers []handlerInfo
    constValues := map[string]string{}

    err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
        if walkErr != nil { return walkErr }
        if d.IsDir() {
            if d.Name() == ".git" { return filepath.SkipDir }
            return nil
        }
        if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") { return nil }
        src, err := os.ReadFile(path)
        if err != nil { return err }
        f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
        if err != nil { return err }

        for _, decl := range f.Decls {
            if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.CONST {
                for _, spec := range gd.Specs {
                    vs, ok := spec.(*ast.ValueSpec); if !ok { continue }
                    for i, n := range vs.Names {
                        if n.Name != "clientCustomShopBuy" || i >= len(vs.Values) { continue }
                        switch v := vs.Values[i].(type) {
                        case *ast.BasicLit:
                            constValues[n.Name] = v.Value
                        default:
                            constValues[n.Name] = fmt.Sprintf("%T", v)
                        }
                    }
                }
            }
            fd, ok := decl.(*ast.FuncDecl)
            if !ok || fd.Name == nil || fd.Name.Name != "handleShopBuy" || fd.Body == nil { continue }
            startOff := fset.Position(fd.Pos()).Offset
            endOff := fset.Position(fd.End()).Offset
            if startOff < 0 || endOff > len(src) || endOff <= startOff { return fmt.Errorf("bad function offsets for %s", path) }
            callSet := map[string]bool{}
            ast.Inspect(fd.Body, func(n ast.Node) bool {
                call, ok := n.(*ast.CallExpr); if !ok { return true }
                if name := callName(call.Fun); name != "" { callSet[name] = true }
                return true
            })
            calls := make([]string, 0, len(callSet))
            for name := range callSet { calls = append(calls, name) }
            sort.Strings(calls)
            rel, _ := filepath.Rel(root, path)
            handlers = append(handlers, handlerInfo{path: rel, start: fset.Position(fd.Pos()).Line, end: fset.Position(fd.End()).Line, source: string(src[startOff:endOff]), calls: calls})
        }
        return nil
    })
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(2) }
    if len(handlers) != 1 {
        fmt.Fprintf(os.Stderr, "expected exactly one handleShopBuy, found %d\n", len(handlers))
        os.Exit(1)
    }
    h := handlers[0]
    fmt.Printf("handler=%s:%d-%d\n", h.path, h.start, h.end)
    fmt.Printf("clientCustomShopBuy=%s\n", constValues["clientCustomShopBuy"])
    fmt.Println("calls:")
    for _, c := range h.calls { fmt.Println("  " + c) }

    checks := []struct{name, needle string}{
        {"uses_quest_state", "questState"},
        {"uses_quest_store", "questStore"},
        {"uses_persist_quest", "persistQuestLocked"},
        {"uses_lookup_gm_item", "lookupGMItem"},
        {"uses_gm_item_inventory_view", "gmItemInventoryView"},
        {"uses_append_inventory_stack", "appendInventoryStackLocked"},
        {"uses_bag_store", "bagStore"},
        {"uses_player_bag_items", "bagItems"},
        {"uses_shop_exchange_commit", "commitShopExchangeReplacementPersistenceFirst"},
    }
    fmt.Println("flags:")
    for _, c := range checks {
        fmt.Printf("  %s=%t\n", c.name, strings.Contains(h.source, c.needle))
    }
    fmt.Println("source_begin")
    fmt.Println(h.source)
    fmt.Println("source_end")

    // Make sure the discovered selector really is the expected integer literal
    // when represented directly. We only report other representations; we do
    // not invent a value.
    if raw := constValues["clientCustomShopBuy"]; raw != "" {
        if _, err := strconv.ParseInt(raw, 0, 64); err != nil {
            fmt.Printf("clientCustomShopBuy_parse_note=%v\n", err)
        }
    }
}
EOF

set +e
go run "$AUDIT" "$BUILD" > "$ROOT/regular_shop_audit.log" 2>&1
regular_shop_audit=$?
set -e
echo "$regular_shop_audit" > "$ROOT/regular_shop_audit.exit"
rm -f "$AUDIT"
echo "regular_shop_audit=$regular_shop_audit"
cat "$ROOT/regular_shop_audit.log"
if [ "$regular_shop_audit" -ne 0 ]; then
    exit 97
fi

# Read-only audit: Stage10/11 production source identity must remain unchanged.
source_files=$(find "$BUILD" -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' | wc -l | tr -d ' ')
manifest_sha=$(awk '{print $1}' "$ROOT/generated-manifest.sha256")
echo "stage12_source_files=$source_files"
echo "stage12_manifest_sha256=$manifest_sha"
