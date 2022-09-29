package exitchecker

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

var ExitCheckAnalyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "check call os.Exit in main()",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {

	for _, file := range pass.Files {
		if file.Name.String() != "main" {
			continue
		}

		inMain := false

		ast.Inspect(file, func(node ast.Node) bool {

			// Обработка только пакета main
			if file.Name.Name != "main" {
				return true
			}

			if fnDecl, ok := node.(*ast.FuncDecl); ok {
				// запоминаю, что нахожусь в функции main
				inMain = fnDecl.Name.Name == "main"
			}

			if !inMain {
				return true
			}

			if callExpr, ok := node.(*ast.CallExpr); ok {

				if s, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
					if s.Sel.Name == "Exit" {
						fmt.Printf("%v: calling os.Exit from main\n", pass.Fset.Position(callExpr.Fun.Pos()))
						return false
					}
				}
			}

			return true
		})
	}

	// реализация будет ниже
	return nil, nil
}
