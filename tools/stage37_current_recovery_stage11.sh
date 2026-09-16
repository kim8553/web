#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
BUILD="$ROOT/buildtree"
EXPECTED_MANIFEST="936a075a8f0baf477e253c789cf8e756436b576c67629b56901826afb04e1a44"

# Reproduce the fully-passed Stage10 source/build first. Stage11 adds no
# production source: it audits that unresolved final binding cannot be turned
# into a known value from non-test production code.
bash tools/stage37_current_recovery_stage10.sh

# Go ignores explicit source files whose basenames start with '.' or '_'. Keep
# the temporary audit source in /tmp with a normal basename.
AUDIT="$(mktemp /tmp/stage11_exchangebinding_constructor_audit.XXXXXX.go)"
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

const bindingImport = "github.com/local/9yin-go-server/internal/exchangebinding"

func main() {
    if len(os.Args) != 2 {
        fmt.Fprintln(os.Stderr, "usage: audit <source-root>")
        os.Exit(2)
    }
    root, err := filepath.Abs(os.Args[1])
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(2)
    }

    fset := token.NewFileSet()
    var violations []string
    err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
        if walkErr != nil {
            return walkErr
        }
        if d.IsDir() {
            if d.Name() == ".git" {
                return filepath.SkipDir
            }
            return nil
        }
        if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
            return nil
        }

        file, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
        if parseErr != nil {
            return fmt.Errorf("parse imports %s: %w", path, parseErr)
        }
        aliases := map[string]bool{}
        for _, imp := range file.Imports {
            p, unquoteErr := strconv.Unquote(imp.Path.Value)
            if unquoteErr != nil || p != bindingImport {
                continue
            }
            alias := "exchangebinding"
            if imp.Name != nil {
                alias = imp.Name.Name
            }
            switch alias {
            case ".":
                rel, _ := filepath.Rel(root, path)
                violations = append(violations, rel+": dot-import of exchangebinding is forbidden in production code")
                continue
            case "_":
                continue
            default:
                aliases[alias] = true
            }
        }
        if len(aliases) == 0 {
            return nil
        }

        full, parseErr := parser.ParseFile(fset, path, nil, 0)
        if parseErr != nil {
            return fmt.Errorf("parse source %s: %w", path, parseErr)
        }
        ast.Inspect(full, func(n ast.Node) bool {
            call, ok := n.(*ast.CallExpr)
            if !ok {
                return true
            }
            sel, ok := call.Fun.(*ast.SelectorExpr)
            if !ok {
                return true
            }
            id, ok := sel.X.(*ast.Ident)
            if !ok || !aliases[id.Name] {
                return true
            }
            if sel.Sel.Name != "Bound" && sel.Sel.Name != "Unbound" {
                return true
            }
            pos := fset.Position(call.Pos())
            rel, _ := filepath.Rel(root, pos.Filename)
            violations = append(violations, fmt.Sprintf("%s:%d:%d: forbidden direct exchangebinding.%s() in production code", rel, pos.Line, pos.Column, sel.Sel.Name))
            return true
        })
        return nil
    })
    if err != nil {
        fmt.Fprintln(os.Stderr, "audit error:", err)
        os.Exit(2)
    }

    sort.Strings(violations)
    if len(violations) != 0 {
        for _, v := range violations {
            fmt.Fprintln(os.Stderr, v)
        }
        os.Exit(1)
    }
    fmt.Println("PASS: no non-test production code constructs authoritative exchangebinding Bound/Unbound decisions")
}
EOF

set +e
go run "$AUDIT" "$BUILD" > "$ROOT/binding_constructor_audit.log" 2>&1
binding_constructor_audit=$?
set -e
echo "$binding_constructor_audit" > "$ROOT/binding_constructor_audit.exit"
rm -f "$AUDIT"

echo "binding_constructor_audit=$binding_constructor_audit"
cat "$ROOT/binding_constructor_audit.log"

# Stage11 must be a pure safety audit over the exact Stage10 production source.
source_files=$(find "$BUILD" -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' | wc -l | tr -d ' ')
manifest_sha=$(awk '{print $1}' "$ROOT/generated-manifest.sha256")
echo "stage11_source_files=$source_files"
echo "stage11_manifest_sha256=$manifest_sha"
if [ "$source_files" != "212" ]; then
    echo "Stage11 source count drift: got $source_files, want 212" >&2
    exit 94
fi
if [ "$manifest_sha" != "$EXPECTED_MANIFEST" ]; then
    echo "Stage11 manifest drift: got $manifest_sha, want $EXPECTED_MANIFEST" >&2
    exit 95
fi
if [ "$binding_constructor_audit" -ne 0 ]; then
    exit 96
fi

# GM item grant is intentionally outside the active Stage11 work scope.
# No source mutation is performed here.
