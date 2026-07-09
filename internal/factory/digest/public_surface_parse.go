// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// symbolInfo holds symbol data including its signature for modification detection.
type symbolInfo struct {
	Key       symbolKey
	Signature string
}

// parseExportsFromBytes extracts exported symbols from Go source bytes.
func parseExportsFromBytes(content []byte, pkgName string) (map[symbolKey]symbolInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// Use ast.FileExports to filter to exported declarations
	if !ast.FileExports(node) {
		return make(map[symbolKey]symbolInfo), nil
	}

	exports := make(map[symbolKey]symbolInfo)

	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.IsExported() {
				sig := fmt.Sprintf("func %s%s", d.Name.Name, typeString(d.Type))
				if d.Recv != nil && len(d.Recv.List) > 0 {
					receiver := receiverTypeName(d.Recv.List[0].Type)
					key := symbolKey{Package: pkgName, Name: d.Name.Name, Kind: "method", Receiver: receiver}
					exports[key] = symbolInfo{Key: key, Signature: sig}
				} else {
					key := symbolKey{Package: pkgName, Name: d.Name.Name, Kind: "func"}
					exports[key] = symbolInfo{Key: key, Signature: sig}
				}
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.IsExported() {
						sig := fmt.Sprintf("type %s %s", s.Name.Name, typeString(s.Type))
						key := symbolKey{Package: pkgName, Name: s.Name.Name, Kind: "type"}
						exports[key] = symbolInfo{Key: key, Signature: sig}

						switch st := s.Type.(type) {
						case *ast.StructType:
							for _, field := range st.Fields.List {
								if field.Names != nil && len(field.Names) > 0 && field.Names[0].IsExported() {
									fieldSig := fmt.Sprintf("field %s %s", field.Names[0].Name, typeString(field.Type))
									fieldKey := symbolKey{Package: pkgName, Name: s.Name.Name + "." + field.Names[0].Name, Kind: "field", Receiver: s.Name.Name}
									exports[fieldKey] = symbolInfo{Key: fieldKey, Signature: fieldSig}
								}
							}
						case *ast.InterfaceType:
							for _, method := range st.Methods.List {
								if len(method.Names) > 0 && method.Names[0].IsExported() {
									methodSig := fmt.Sprintf("method %s%s", method.Names[0].Name, typeString(method.Type))
									methodKey := symbolKey{Package: pkgName, Name: method.Names[0].Name, Kind: "interface_method", Receiver: s.Name.Name}
									exports[methodKey] = symbolInfo{Key: methodKey, Signature: methodSig}
								}
							}
						}
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if name.IsExported() {
							var kind string
							var sig string
							if d.Tok == token.CONST {
								kind = "const"
								sig = fmt.Sprintf("const %s", name.Name)
								if s.Values != nil && len(s.Values) > 0 {
									sig += " = " + exprString(s.Values[0])
								}
							} else {
								kind = "var"
								sig = fmt.Sprintf("var %s", name.Name)
								if s.Values != nil && len(s.Values) > 0 {
									sig += " = " + exprString(s.Values[0])
								}
							}
							key := symbolKey{Package: pkgName, Name: name.Name, Kind: kind}
							exports[key] = symbolInfo{Key: key, Signature: sig}
						}
					}
				}
			}
		}
	}

	return exports, nil
}

// typeString returns a string representation of a type.
func typeString(t ast.Expr) string {
	if t == nil {
		return "()"
	}
	switch v := t.(type) {
	case *ast.FuncType:
		params := typeStringFromFields(v.Params)
		results := ""
		if v.Results != nil {
			results = typeStringFromFields(v.Results)
		}
		return fmt.Sprintf("(%s) %s", params, results)
	case *ast.Ident:
		return v.Name
	case *ast.StarExpr:
		return "*" + typeString(v.X)
	case *ast.SelectorExpr:
		return typeString(v.X) + "." + v.Sel.Name
	case *ast.ArrayType:
		if v.Len == nil {
			return "[]" + typeString(v.Elt)
		}
		return fmt.Sprintf("[%s]%s", exprString(v.Len), typeString(v.Elt))
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", typeString(v.Key), typeString(v.Value))
	case *ast.ChanType:
		return fmt.Sprintf("chan %s", typeString(v.Value))
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	default:
		return "any"
	}
}

func typeStringFromFields(fields *ast.FieldList) string {
	if fields == nil || len(fields.List) == 0 {
		return ""
	}
	var parts []string
	for _, f := range fields.List {
		if len(f.Names) == 0 {
			parts = append(parts, typeString(f.Type))
		} else {
			for _, n := range f.Names {
				parts = append(parts, fmt.Sprintf("%s %s", n.Name, typeString(f.Type)))
			}
		}
	}
	return strings.Join(parts, ", ")
}

func exprString(e ast.Expr) string {
	if e == nil {
		return ""
	}
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.BasicLit:
		return v.Value
	case *ast.ParenExpr:
		return "(" + exprString(v.X) + ")"
	default:
		return "..."
	}
}

// receiverTypeName extracts the type name from a receiver expression.
func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	case *ast.Ident:
		return t.Name
	}
	return ""
}

// extractCLISymbols extracts CLI command names from cmd/leamas files.
func extractCLISymbols(filePath string) []string {
	var commands []string
	content, err := os.ReadFile(filePath)
	if err != nil {
		return commands
	}
	contentStr := string(content)

	cmdPatterns := []string{
		// Multi-word commands: "leamas digest", "leamas factory verify"
		`"(leamas\s+\w+(?:\s+\w+)?)"`,
		`"(factory\s+\w+(?:\s+\w+)?)"`,
		`"(claim(?:\s+\w+)?)"`,
		`"(evidence(?:\s+\w+)?)"`,
		`"(gate(?:\s+\w+)?)"`,
		`"(digest(?:\s+\w+)?)"`,
		`"(witness(?:\s+\w+)?)"`,
		`"(run(?:\s+\w+)?)"`,
		// Single-word commands in Use field: "leamas", "version", "new"
		`"(?:leamas|version|new|help|digest|gate|claim|evidence|witness|run)"`,
	}

	for _, pattern := range cmdPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(contentStr, -1)
		for _, m := range matches {
			if len(m) > 1 && m[1] != "" {
				cmd := strings.TrimSpace(m[1])
				if isLikelyCommand(cmd) {
					commands = append(commands, cmd)
				}
			}
		}
	}

	commands = deduplicateStrings(commands)
	sort.Strings(commands)
	return commands
}

// isLikelyCommand checks if a string looks like a CLI command name.
func isLikelyCommand(s string) bool {
	if len(s) < 2 || len(s) > 50 {
		return false
	}
	if strings.Contains(s, "://") || strings.Contains(s, "${") {
		return false
	}
	return true
}

// getFileContentAtCommit gets file content at a specific git commit.
func getFileContentAtCommit(repoRoot, filePath, commit string) ([]byte, error) {
	output, err := RunGit(repoRoot, []string{"show", commit + ":" + filePath})
	if err != nil {
		return nil, err
	}
	return []byte(output), nil
}

// getIndexFileContent gets file content from git index (staged).
func getIndexFileContent(repoRoot, filePath string) ([]byte, error) {
	output, err := RunGit(repoRoot, []string{"show", ":" + filePath})
	if err != nil {
		return nil, err
	}
	return []byte(output), nil
}

// getWorktreeFileContent gets current file content from worktree.
func getWorktreeFileContent(repoRoot, filePath string) ([]byte, error) {
	fullPath := filepath.Join(repoRoot, filePath)
	return os.ReadFile(fullPath)
}
