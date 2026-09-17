//go:build ignore

// This CI-only extractor prints selected, unmodified production declarations.
// Running the entire protocol-probe test binary without the current client's
// external resources panics at unrelated skill catalog package initialization.
// The generated isolated package tests the actual parsed production functions;
// it is NOT a substitute for full-server, live, or E2E tests.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	source := flag.String("source", "", "protocol-probe production source directory")
	output := flag.String("output", "", "isolated output Go file")
	flag.Parse()
	if *source == "" || *output == "" {
		fail(fmt.Errorf("-source and -output are required"))
	}
	wanted := map[string][]string{
		"custom_c2s.go": {
			"clientCustomMessage", "clientCustomValue",
		},
		"shop_catalog.go": {
			"shopCatalogItem", "loadShopCatalogSection",
		},
		"latest_client_shop_view_contract.go": {
			"currentShopAuthoredCoordinates", "currentShopListing",
		},
		"latest_client_shop_exchange_config.go": {
			"shopExchangeDefinition", "loadShopExchangeDefinition", "validateShopExchangeDefinition",
			"validateExchangePairList", "validateExchangeIntList",
		},
		"latest_client_shop_condition_capability.go": {
			"conditionCapabilityClass", "exchangeConditionSpec", "exchangeConditionCapability",
			"shopExchangeConditionCapabilityAudit",
		},
		"latest_client_shop_exchange_contract.go": {
			"clientCustomExchangeFromShop", "shopExchangeBuyRequest", "currentShopExchangeBuySelection",
			"currentShopExchangeBuyAuthority", "parseShopExchangeBuyRequest",
			"resolveCurrentShopExchangeBuySelection", "resolveAuthorizedCurrentShopExchangeBuyAuthority",
		},
	}
	var out bytes.Buffer
	out.WriteString("package main\n\nimport (\"bufio\"; \"fmt\"; \"os\"; \"strconv\"; \"strings\")\n\n")
	files := make([]string, 0, len(wanted))
	for file := range wanted {
		files = append(files, file)
	}
	sort.Strings(files)
	for _, name := range files {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filepath.Join(*source, name), nil, 0)
		if err != nil {
			fail(fmt.Errorf("parse %s: %w", name, err))
		}
		remaining := make(map[string]bool, len(wanted[name]))
		for _, symbol := range wanted[name] {
			remaining[symbol] = true
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if !remaining[d.Name.Name] {
					continue
				}
				printDecl(&out, fset, d)
				delete(remaining, d.Name.Name)
			case *ast.GenDecl:
				if d.Tok != token.TYPE && d.Tok != token.CONST {
					continue
				}
				for _, spec := range d.Specs {
					if typ, ok := spec.(*ast.TypeSpec); ok && remaining[typ.Name.Name] {
						printDecl(&out, fset, &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{typ}})
						delete(remaining, typ.Name.Name)
					}
					if vals, ok := spec.(*ast.ValueSpec); ok {
						for _, ident := range vals.Names {
							if remaining[ident.Name] {
								// Keep the complete original declaration for grouped constants.
								printDecl(&out, fset, d)
								for _, part := range d.Specs {
									if group, ok := part.(*ast.ValueSpec); ok {
										for _, entry := range group.Names {
											delete(remaining, entry.Name)
										}
									}
								}
								break
							}
							}
						}
					}
			}
		}
		if len(remaining) != 0 {
			fail(fmt.Errorf("%s: missing production declarations: %v", name, remaining))
		}
	}
	formatted, err := format.Source(out.Bytes())
	if err != nil {
		fail(fmt.Errorf("format extracted declarations: %w", err))
	}
	if err := os.WriteFile(*output, formatted, 0o600); err != nil {
		fail(err)
	}
}

func printDecl(out *bytes.Buffer, fset *token.FileSet, decl ast.Decl) {
	if err := format.Node(out, fset, decl); err != nil {
		fail(err)
	}
	out.WriteString("\n\n")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
