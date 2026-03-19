package exitcheck

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer настраивает анализатор для проверки вызовов os.Exit.
var Analyzer = &analysis.Analyzer{
	Name:     "exitcheck",
	Doc:      "check for direct os.Exit calls in main function of main package",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// run реализует логику анализатора.
func run(pass *analysis.Pass) (interface{}, error) {
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Фильтр для поиска только функций
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspector.Preorder(nodeFilter, func(node ast.Node) {
		fn := node.(*ast.FuncDecl)

		// Проверяем, что это функция main в пакете main
		if fn.Name.Name != "main" || pass.Pkg.Name() != "main" {
			return
		}

		// Проверяем, что функция main принимает 0 параметров и 0 возвращаемых значений
		if fn.Type.Params != nil && len(fn.Type.Params.List) > 0 {
			return
		}
		if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
			return
		}

		// Рекурсивно обходим тело функции
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			// Ищем вызовы функций
			callExpr, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			// Ищем селекторные выражения (например, os.Exit)
			selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			// Проверяем идентификатор пакета
			ident, ok := selExpr.X.(*ast.Ident)
			if !ok {
				return true
			}

			// Проверяем, что это os.Exit
			if ident.Name == "os" && selExpr.Sel.Name == "Exit" {
				// Проверяем импорт пакета os
				for _, imp := range pass.Pkg.Imports() {
					if imp.Path() == "os" {
						pass.Reportf(callExpr.Pos(),
							"прямой вызов os.Exit в функции main запрещен. "+
								"Используйте возврат из main или panic вместо os.Exit")
					}
				}
			}

			return true
		})
	})

	return nil, nil
}
