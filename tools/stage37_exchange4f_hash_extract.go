//go:build ignore

// A CI-only AST extractor: exercise unchanged production hash-verifier bodies
// using synthetic local files, never guessed replacement hashes or game data.
package main

import (
 "bytes"
 "fmt"
 "go/ast"
 "go/format"
 "go/parser"
 "go/token"
 "os"
)

func main() {
 if len(os.Args)!=3 { panic("usage: go run stage37_exchange4f_hash_extract.go SOURCE OUTPUT") }
 fset:=token.NewFileSet()
 file,err:=parser.ParseFile(fset,os.Args[1],nil,0)
 if err!=nil { panic(err) }
 need:=map[string]bool{"fileSHA256Hex":true,"requireFileSHA256":true}
 var out bytes.Buffer
 out.WriteString("package main\nimport (\"crypto/sha256\";\"encoding/hex\";\"fmt\";\"os\";\"strings\")\n")
 for _,decl:=range file.Decls {
  fn,ok:=decl.(*ast.FuncDecl);if !ok || !need[fn.Name.Name] {continue}
  if err:=format.Node(&out,fset,fn);err!=nil {panic(err)}
  out.WriteString("\n")
  delete(need,fn.Name.Name)
 }
 if len(need)!=0 { panic(fmt.Sprintf("missing exact production hash functions %v",need)) }
 normalized,err:=format.Source(out.Bytes());if err!=nil {panic(err)}
 if err:=os.WriteFile(os.Args[2],normalized,0600);err!=nil {panic(err)}
}
