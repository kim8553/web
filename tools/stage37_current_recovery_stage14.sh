#!/usr/bin/env bash
set -euo pipefail
ROOT="$(pwd)"
BUILD="$ROOT/buildtree"

# Read-only topology audit for rebuilding regular NPC shop purchase on the
# current bag/currency architecture. It intentionally does not touch deferred GM
# grant code and does not invent a client purchase message ID.
bash tools/stage37_current_recovery_stage12.sh

AUDIT="$(mktemp /tmp/stage14_shop_topology.XXXXXX.go)"
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

type hit struct{ path, kind, name, src string; line int }
var wanted = map[string]bool{
  "bagItem":true,"bagStoreIface":true,"bagStore":true,"mysqlBagStore":true,
  "currencySnapshot":true,"currencyStore":true,"mysqlCurrencyStore":true,
  "enrichBagItem":true,"grantBagItems":true,"openBagStore":true,"openBagStoreAt":true,
  "openCurrencyStore":true,"openCurrencyStoreAt":true,
  "shopWallet":true,"setShopCapital":true,"setWallet":true,"walletUpdate":true,
  "sceneLifecycle":true,"playerActor":true,
}
func nodeSrc(src []byte,fset *token.FileSet,n ast.Node) string { a,b:=fset.Position(n.Pos()).Offset,fset.Position(n.End()).Offset; if a<0||b<=a||b>len(src){return ""}; return string(src[a:b]) }
func main(){
 if len(os.Args)!=2 { os.Exit(2) }; root,_:=filepath.Abs(os.Args[1]); fset:=token.NewFileSet(); var hits []hit
 err:=filepath.WalkDir(root,func(path string,d fs.DirEntry,e error)error{
  if e!=nil{return e}; if d.IsDir(){if d.Name()==".git"{return filepath.SkipDir};return nil}; if !strings.HasSuffix(path,".go")||strings.HasSuffix(path,"_test.go"){return nil}
  raw,e:=os.ReadFile(path);if e!=nil{return e}; f,e:=parser.ParseFile(fset,path,raw,0);if e!=nil{return e}; rel,_:=filepath.Rel(root,path)
  for _,decl:=range f.Decls{
   switch x:=decl.(type){
   case *ast.FuncDecl:
    n:=x.Name.Name; recv:=""; if x.Recv!=nil&&len(x.Recv.List)>0 { recv=nodeSrc(raw,fset,x.Recv.List[0].Type) }
    if wanted[n] || (n=="Load"||n=="Save")&&(strings.Contains(recv,"BagStore")||strings.Contains(strings.ToLower(recv),"currency")) {
      hits=append(hits,hit{rel,"func",recv+"."+n,nodeSrc(raw,fset,x),fset.Position(x.Pos()).Line})
    }
   case *ast.GenDecl:
    for _,sp:=range x.Specs{
     ts,ok:=sp.(*ast.TypeSpec); if ok&&wanted[ts.Name.Name] { hits=append(hits,hit{rel,"type",ts.Name.Name,nodeSrc(raw,fset,ts),fset.Position(ts.Pos()).Line}) }
    }
   }
  }
  // Important current wiring references in main/scene source.
  if strings.HasSuffix(rel,"main.go") || strings.HasSuffix(rel,"scene_lifecycle.go") {
   lines:=strings.Split(string(raw),"\n"); keys:=[]string{"openBagStore","openCurrencyStore","bagStore","currencyStore","setWallet","shopWallet","activeShopID","role_currency"}
   for i,line:=range lines { for _,k:=range keys { if strings.Contains(line,k) { hits=append(hits,hit{rel,"ref",k,strings.TrimSpace(line),i+1}); break } } }
  }
  return nil
 }); if err!=nil { panic(err) }
 sort.Slice(hits,func(i,j int)bool{if hits[i].path!=hits[j].path{return hits[i].path<hits[j].path};if hits[i].line!=hits[j].line{return hits[i].line<hits[j].line};return hits[i].name<hits[j].name})
 fmt.Printf("topology_hits=%d\n",len(hits)); for _,h:=range hits { fmt.Printf("topology=%s:%d kind=%s name=%s\n",h.path,h.line,h.kind,h.name); fmt.Println("topology_source_begin");fmt.Println(h.src);fmt.Println("topology_source_end") }
}
EOF
set +e
go run "$AUDIT" "$BUILD" > "$ROOT/regular_shop_topology.log" 2>&1
rc=$?
set -e
rm -f "$AUDIT"
echo "$rc" > "$ROOT/regular_shop_topology.exit"
echo "regular_shop_topology=$rc"
cat "$ROOT/regular_shop_topology.log"
if [ "$rc" -ne 0 ]; then exit 99; fi
