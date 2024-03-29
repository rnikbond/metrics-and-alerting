// Package exitchecker - анализатор, который запрещает прямой вызов os.Exit() из функции main
package exitchecker

import (
	"go/ast"
	"os"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var ExitCheckAnalyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "check call os.Exit in func main() package main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {

	mainNodes := make([]ast.Node, 0)
	tmpDir := os.TempDir()
	tmpDir = strings.ReplaceAll(tmpDir, "Temp", "")

	for _, file := range pass.Files {

		// Обработка только пакета main
		if file.Name.String() != "main" {
			continue
		}

		/*
			Под Windows почему-то лезет в какую-то папку "C:\Users\user\AppData\Local\go-build\5f\5"
			и в ней какой-то файл, который выглядит так:
			***********************************************************************************
				// Code generated by 'go test'. DO NOT EDIT.
				....
				func main() {
					m := testing.MainStart(testdeps.TestDeps{}, tests, benchmarks, examples)
					os.Exit(m.Run())
				}
			***********************************************************************************

			Не понял, как его обойти и почему анализатор туда загялдывает.
			Поэтому пока такой костыль.
		*/
		// BUG(rnikbond): Разобраться, почему в pass.Files попадают файлы из какой-то временной директории
		if fullPath := pass.Fset.Position(file.Pos()).String(); strings.Contains(fullPath, tmpDir) {
			continue
		}

		// Сохранение nodes с объявлением функции main
		ast.Inspect(file, func(node ast.Node) bool {
			if fnDecl, ok := node.(*ast.FuncDecl); ok {
				if fnDecl.Name.Name == "main" {
					mainNodes = append(mainNodes, node)
				}
			}

			return true
		})
	}

	// Поиск в функциях main вызова os.Exit
	for _, node := range mainNodes {
		ast.Inspect(node, func(node ast.Node) bool {
			if callExpr, ok := node.(*ast.CallExpr); ok {
				if s, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
					if s.Sel.Name == "Exit" {
						pass.Reportf(s.Pos(), "you call Exit from main, but you do it without respect")
					}
				}
			}

			return true
		})
	}

	return nil, nil
}
