// Package noexit содержит пользовательский анализатор,
// который запрещает прямой вызов os.Exit в функции main пакета main.
package noexit

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// Analyzer представляет анализатор, запрещающий использовать os.Exit в функции main.
var Analyzer = &analysis.Analyzer{
	Name: "noexit",
	Doc:  "запрещает использовать os.Exit в функции main пакета main",
	Run:  run,
}

// NewAnalyzer возвращает анализатор noexit.
func NewAnalyzer() *analysis.Analyzer {
	return Analyzer
}

func run(pass *analysis.Pass) (interface{}, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" || fn.Recv != nil {
				continue
			}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "os" && sel.Sel.Name == "Exit" {
					obj := pass.TypesInfo.Uses[sel.Sel]
					if fn, ok := obj.(*types.Func); ok && fn.FullName() == "os.Exit" {
						pass.Reportf(call.Pos(), "вызов os.Exit в функции main запрещён")
					}
				}
				return true
			})
		}
	}
	return nil, nil
}
