#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
BUILD="$ROOT/buildtree"

# Stage13 remains read-only. First require Stage12 (which itself requires the
# Stage11 binding audit and the fully-passed Stage10 source/build).
bash tools/stage37_current_recovery_stage12.sh

AUDIT="$(mktemp /tmp/stage13_regular_shop_dispatch_audit.XXXXXX.go)"
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

type hit struct { path string; line int; text string }

func main() {
    if len(os.Args) != 2 { fmt.Fprintln(os.Stderr, "usage: audit <source-root>"); os.Exit(2) }
    root, err := filepath.Abs(os.Args[1]); if err != nil { panic(err) }
    fset := token.NewFileSet()
    var handleCalls, selectorRefs, verifierDecls []hit

    err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
        if walkErr != nil { return walkErr }
        if d.IsDir() { if d.Name()==".git" { return filepath.SkipDir }; return nil }
        if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") { return nil }
        src, err := os.ReadFile(path); if err != nil { return err }
        f, err := parser.ParseFile(fset, path, src, 0); if err != nil { return err }
        rel, _ := filepath.Rel(root, path)

        for _, decl := range f.Decls {
            fd, ok := decl.(*ast.FuncDecl)
            if ok && fd.Name != nil && fd.Name.Name == "verifiedShopBuyMessageID" {
                p := fset.Position(fd.Pos())
                start, end := p.Offset, fset.Position(fd.End()).Offset
                text := ""
                if start >= 0 && end > start && end <= len(src) { text = string(src[start:end]) }
                verifierDecls = append(verifierDecls, hit{rel,p.Line,text})
            }
        }

        ast.Inspect(f, func(n ast.Node) bool {
            switch x := n.(type) {
            case *ast.CallExpr:
                if sel, ok := x.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "handleShopBuy" {
                    p := fset.Position(x.Pos())
                    handleCalls = append(handleCalls, hit{rel,p.Line,"selector call"})
                }
            case *ast.Ident:
                if x.Name == "clientCustomShopBuy" {
                    p := fset.Position(x.Pos())
                    selectorRefs = append(selectorRefs, hit{rel,p.Line,"identifier"})
                }
            }
            return true
        })
        return nil
    })
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(2) }

    sort.Slice(handleCalls, func(i,j int) bool { if handleCalls[i].path==handleCalls[j].path { return handleCalls[i].line<handleCalls[j].line }; return handleCalls[i].path<handleCalls[j].path })
    sort.Slice(selectorRefs, func(i,j int) bool { if selectorRefs[i].path==selectorRefs[j].path { return selectorRefs[i].line<selectorRefs[j].line }; return selectorRefs[i].path<selectorRefs[j].path })

    fmt.Printf("handleShopBuy_non_test_call_count=%d\n", len(handleCalls))
    for _, h := range handleCalls { fmt.Printf("handleShopBuy_call=%s:%d\n", h.path, h.line) }
    fmt.Printf("clientCustomShopBuy_non_test_ref_count=%d\n", len(selectorRefs))
    for _, h := range selectorRefs { fmt.Printf("clientCustomShopBuy_ref=%s:%d\n", h.path, h.line) }
    fmt.Printf("verifiedShopBuyMessageID_decl_count=%d\n", len(verifierDecls))
    for _, h := range verifierDecls {
        fmt.Printf("verifiedShopBuyMessageID_decl=%s:%d\n", h.path, h.line)
        fmt.Println("verifier_source_begin")
        fmt.Println(h.text)
        fmt.Println("verifier_source_end")
    }

    if len(handleCalls) == 0 {
        fmt.Fprintln(os.Stderr, "no non-test handleShopBuy dispatch call found")
        os.Exit(1)
    }
}
EOF

set +e
go run "$AUDIT" "$BUILD" > "$ROOT/regular_shop_dispatch_audit.log" 2>&1
regular_shop_dispatch_audit=$?
set -e
echo "$regular_shop_dispatch_audit" > "$ROOT/regular_shop_dispatch_audit.exit"
rm -f "$AUDIT"

echo "regular_shop_dispatch_audit=$regular_shop_dispatch_audit"
cat "$ROOT/regular_shop_dispatch_audit.log"
if [ "$regular_shop_dispatch_audit" -ne 0 ]; then exit 98; fi

source_files=$(find "$BUILD" -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' | wc -l | tr -d ' ')
manifest_sha=$(awk '{print $1}' "$ROOT/generated-manifest.sha256")
echo "stage13_source_files=$source_files"
echo "stage13_manifest_sha256=$manifest_sha"
