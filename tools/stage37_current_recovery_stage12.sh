#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
BUILD="$ROOT/buildtree"
EXPECTED_MANIFEST="936a075a8f0baf477e253c789cf8e756436b576c67629b56901826afb04e1a44"

# Stage12 is read-only. First reproduce Stage11, then inventory every plausible
# regular NPC shop-buy production path. Absence is a finding, not a CI error.
bash tools/stage37_current_recovery_stage11.sh

AUDIT="$(mktemp /tmp/stage12_regular_shop_audit.XXXXXX.go)"
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

type row struct { path, kind, name, source string; line int }

func interesting(s string) bool {
    s = strings.ToLower(s)
    return strings.Contains(s,"shop") || strings.Contains(s,"buy") || strings.Contains(s,"purchase")
}
func sourceRange(src []byte, fset *token.FileSet, n ast.Node) string {
    a,b := fset.Position(n.Pos()).Offset, fset.Position(n.End()).Offset
    if a < 0 || b <= a || b > len(src) { return "" }
    return string(src[a:b])
}
func main() {
    if len(os.Args)!=2 { fmt.Fprintln(os.Stderr,"usage: audit <source-root>"); os.Exit(2) }
    root,_ := filepath.Abs(os.Args[1])
    fset := token.NewFileSet()
    var rows []row
    exactHandlers := 0
    selectorValues := map[string]string{}

    err := filepath.WalkDir(root, func(path string,d fs.DirEntry,walkErr error) error {
        if walkErr!=nil { return walkErr }
        if d.IsDir() { if d.Name()==".git" { return filepath.SkipDir }; return nil }
        if !strings.HasSuffix(path,".go") || strings.HasSuffix(path,"_test.go") { return nil }
        src,err := os.ReadFile(path); if err!=nil { return err }
        f,err := parser.ParseFile(fset,path,src,parser.ParseComments); if err!=nil { return err }
        rel,_ := filepath.Rel(root,path)

        for _,decl := range f.Decls {
            switch x := decl.(type) {
            case *ast.FuncDecl:
                if x.Name==nil { continue }
                if x.Name.Name=="handleShopBuy" { exactHandlers++ }
                if interesting(x.Name.Name) {
                    rows=append(rows,row{rel,"func",x.Name.Name,sourceRange(src,fset,x),fset.Position(x.Pos()).Line})
                }
            case *ast.GenDecl:
                for _,sp := range x.Specs {
                    vs,ok := sp.(*ast.ValueSpec); if !ok { continue }
                    for i,n := range vs.Names {
                        if interesting(n.Name) {
                            rows=append(rows,row{rel,"value",n.Name,sourceRange(src,fset,vs),fset.Position(vs.Pos()).Line})
                        }
                        if n.Name=="clientCustomShopBuy" && i<len(vs.Values) {
                            if lit,ok:=vs.Values[i].(*ast.BasicLit); ok { selectorValues[n.Name]=lit.Value } else { selectorValues[n.Name]=fmt.Sprintf("%T",vs.Values[i]) }
                        }
                    }
                }
            }
        }
        // Record direct references to historically important symbols, even when
        // the declarations themselves are absent from the current source.
        ast.Inspect(f, func(n ast.Node) bool {
            id,ok:=n.(*ast.Ident); if !ok { return true }
            switch id.Name {
            case "clientCustomShopBuy","handleShopBuy","parseShopBuyArguments","verifiedShopBuyMessageID":
                rows=append(rows,row{rel,"ref",id.Name,"",fset.Position(id.Pos()).Line})
            }
            return true
        })
        return nil
    })
    if err!=nil { fmt.Fprintln(os.Stderr,err); os.Exit(2) }

    sort.Slice(rows,func(i,j int)bool{ if rows[i].path!=rows[j].path{return rows[i].path<rows[j].path}; if rows[i].line!=rows[j].line{return rows[i].line<rows[j].line}; if rows[i].kind!=rows[j].kind{return rows[i].kind<rows[j].kind}; return rows[i].name<rows[j].name })
    fmt.Printf("handleShopBuy_decl_count=%d\n",exactHandlers)
    fmt.Printf("clientCustomShopBuy=%s\n",selectorValues["clientCustomShopBuy"])
    if raw:=selectorValues["clientCustomShopBuy"]; raw!="" { if _,err:=strconv.ParseInt(raw,0,64); err!=nil { fmt.Printf("clientCustomShopBuy_parse_note=%v\n",err) } }
    fmt.Printf("shop_buy_candidate_rows=%d\n",len(rows))
    for _,r:=range rows {
        fmt.Printf("candidate=%s:%d kind=%s name=%s\n",r.path,r.line,r.kind,r.name)
        if r.kind=="func" || r.kind=="value" { fmt.Println("candidate_source_begin"); fmt.Println(r.source); fmt.Println("candidate_source_end") }
    }
    if exactHandlers==0 { fmt.Println("FINDING: no handleShopBuy declaration exists in the exact reconstructed 212-file production source") }
    if selectorValues["clientCustomShopBuy"]=="" { fmt.Println("FINDING: no clientCustomShopBuy declaration exists in the exact reconstructed 212-file production source") }
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
if [ "$regular_shop_audit" -ne 0 ]; then exit 97; fi

source_files=$(find "$BUILD" -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' | wc -l | tr -d ' ')
manifest_sha=$(awk '{print $1}' "$ROOT/generated-manifest.sha256")
echo "stage12_source_files=$source_files"
echo "stage12_manifest_sha256=$manifest_sha"
if [ "$source_files" != "212" ]; then exit 94; fi
if [ "$manifest_sha" != "$EXPECTED_MANIFEST" ]; then exit 95; fi
