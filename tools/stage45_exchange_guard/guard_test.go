// Stage45 source regression tests. These are NOT a purchase implementation,
// official settlement proof, SQL transaction test, or live-client test.
package stage45_exchange_guard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func called(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		if recv, ok := x.X.(*ast.Ident); ok {
			return recv.Name + "." + x.Sel.Name
		}
	}
	return "<dynamic call>"
}

// audit requires the exact observed 4-field request and an unchanged, terminal
// fail-closed purchase branch. New calls need manual review before updating it.
func audit(src []byte) error {
	f, err := parser.ParseFile(token.NewFileSet(), "contract.go", src, 0)
	if err != nil {
		return err
	}
	var request *ast.StructType
	var decode, handler *ast.FuncDecl
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if ok && ts.Name.Name == "shopExchangeBuyRequest" {
					request, _ = ts.Type.(*ast.StructType)
				}
			}
		case *ast.FuncDecl:
			if d.Name.Name == "parseShopExchangeBuyRequest" {
				decode = d
			}
			if d.Name.Name == "handleShopExchangeContract" {
				handler = d
			}
		}
	}
	if request == nil || request.Fields == nil || decode == nil || decode.Body == nil || handler == nil || handler.Body == nil {
		return fmt.Errorf("missing request, decoder, or handler")
	}
	expected := []string{"ShopID", "Page", "Position", "Count"}
	if len(request.Fields.List) != len(expected) {
		return fmt.Errorf("unexpected 0x4F request field count")
	}
	for i, field := range request.Fields.List {
		if len(field.Names) != 1 || field.Names[0].Name != expected[i] {
			return fmt.Errorf("0x4F wire fields changed at %d", i)
		}
	}
	hasFive := false
	ast.Inspect(decode.Body, func(n ast.Node) bool {
		b, ok := n.(*ast.BinaryExpr)
		if !ok || b.Op != token.NEQ {
			return true
		}
		lit, ok := b.Y.(*ast.BasicLit)
		if !ok {
			return true
		}
		v, err := strconv.Atoi(lit.Value)
		if err != nil || v != 5 {
			return true
		}
		c, ok := b.X.(*ast.CallExpr)
		if !ok || called(c.Fun) != "len" || len(c.Args) != 1 {
			return true
		}
		s, ok := c.Args[0].(*ast.SelectorExpr)
		if ok && s.Sel.Name == "Values" {
			hasFive = true
		}
		return true
	})
	if !hasFive {
		return fmt.Errorf("five-value 0x4F wire guard absent")
	}
	var purchase *ast.IfStmt
	ast.Inspect(handler.Body, func(n ast.Node) bool {
		s, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		a, ok := s.Init.(*ast.AssignStmt)
		if !ok || len(a.Rhs) != 1 {
			return true
		}
		c, ok := a.Rhs[0].(*ast.CallExpr)
		if ok && called(c.Fun) == "parseShopExchangeBuyRequest" {
			purchase = s
		}
		return true
	})
	if purchase == nil || purchase.Body == nil || len(purchase.Body.List) == 0 {
		return fmt.Errorf("purchase branch absent")
	}
	tail, ok := purchase.Body.List[len(purchase.Body.List)-1].(*ast.ReturnStmt)
	if !ok || len(tail.Results) != 2 {
		return fmt.Errorf("purchase branch must terminate without settlement")
	}
	a, aok := tail.Results[0].(*ast.Ident)
	b, bok := tail.Results[1].(*ast.Ident)
	if !aok || !bok || a.Name != "true" || b.Name != "nil" {
		return fmt.Errorf("purchase terminal refusal changed")
	}
	allowed := map[string]bool{
		"resolveDefaultCurrentShopExchangeBuyAuthority": true,
		"loadDefaultCurrentShopConditionAuthority":      true,
		"evaluateExchangeConditionDetails":              true,
		"summarizeCurrentShopExchangeConditionResults":  true,
		"log.Printf": true, "make": true, "len": true, "evaluator.evaluate": true,
	}
	hasRefusal := false
	var bad string
	ast.Inspect(purchase.Body, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			name := called(c.Fun)
			if !allowed[name] {
				bad = name
			}
			if name == "log.Printf" && len(c.Args) > 0 {
				if lit, ok := c.Args[0].(*ast.BasicLit); ok {
					s, e := strconv.Unquote(lit.Value)
					if e == nil && strings.Contains(s, "blocked: native eligibility/cost/bind/commit path unresolved") {
						hasRefusal = true
					}
				}
			}
		}
		switch n.(type) {
		case *ast.GoStmt, *ast.DeferStmt, *ast.FuncLit, *ast.SendStmt:
			bad = "unreviewed side-effect-capable operation"
		}
		return true
	})
	if bad != "" {
		return fmt.Errorf("unreviewed 0x4F operation: %s", bad)
	}
	if !hasRefusal {
		return fmt.Errorf("explicit settlement refusal missing")
	}
	return nil
}

const fixture = `package main
 type shopExchangeBuyRequest struct { ShopID string; Page int32; Position int32; Count int32 }
 func parseShopExchangeBuyRequest(custom clientCustomMessage) { if len(custom.Values) != 5 { return } }
 func handleShopExchangeContract() {
 if request, matched, err := parseShopExchangeBuyRequest(custom); matched {
 authorized, selected, selectErr := resolveDefaultCurrentShopExchangeBuyAuthority(request)
 if selectErr != nil { log.Printf("blocked"); return true, nil }
 if !selected { log.Printf("blocked"); return true, nil }
 authority, conditionErr := loadDefaultCurrentShopConditionAuthority()
 if conditionErr != nil { log.Printf("blocked"); return true, nil }
 details, supported, observationErr := evaluateExchangeConditionDetails()
 if observationErr != nil { log.Printf("blocked"); return true, nil }
 results := make([]bool,len(details))
 observation := summarizeCurrentShopExchangeConditionResults(results,supported)
 log.Printf("blocked: native eligibility/cost/bind/commit path unresolved",observation)
 return true,nil
 }
 return false,nil
 }`

func TestSyntheticRefusalAndMutationRejection(t *testing.T) {
	if err := audit([]byte(fixture)); err != nil {
		t.Fatalf("synthetic baseline: %v", err)
	}
	for _, call := range []string{"grantBagItems()", "link.WriteFrame(frame)", "tx.Commit()", "store.Save()"} {
		altered := strings.Replace(fixture, "return true,nil\n }", call+"\n return true,nil\n }", 1)
		if err := audit([]byte(altered)); err == nil {
			t.Errorf("accepted unreviewed call %s", call)
		}
	}
	altered := strings.Replace(fixture, "Count int32 }", "Count int32; RequestID int64 }", 1)
	if err := audit([]byte(altered)); err == nil {
		t.Error("accepted invented wire request ID")
	}
	altered = strings.Replace(fixture, "return true,nil\n }", "return false,nil\n }", 1)
	if err := audit([]byte(altered)); err == nil {
		t.Error("accepted missing terminal refusal")
	}
}

func TestActualStage44ContractRemainsFailClosed(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	path := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "server", "cmd", "protocol-probe", "latest_client_shop_exchange_contract.go"))
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read actual checked-out production source %s: %v", path, err)
	}
	if err := audit(src); err != nil {
		t.Fatalf("production source gate: %v", err)
	}
}
