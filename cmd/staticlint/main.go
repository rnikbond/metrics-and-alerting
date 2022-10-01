// Package main предназначен для создания анализатора.
//
// Анализатор состоит из:
// Анализатора спецификаторов формативания текста (printf)
// Анализатора затененных переменных (shadow)
// Анализатора структурных тегов (structtag)
// Анализатора проверяет передачу указателя на структуру при декодировании (unmarshal)
// Анализатора излишней проверки переменной bool с константой (bools)
// Анализатора упакованных ошибок (errorsas)
// Анализатор проверяет наличие ошибок с помощью ответов HTTP (httpresponse)
// Анализатор проверяет бесполезные сравнения с nil (nilfunc)
// Анализатора кода, который никогда не будет выполнен (unreachable)
// Анализатора вызова os.Exit() из функции main (exitchecker)
//
// Для добавления классов анализоторов из пакета statickcheck,
// нужно создать одноименный с бинарным файлом анализатора JSON конфигурационный файл и перечислить
// в нем классы.

package main

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"metrics-and-alerting/pkg/exitchecker"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

//go:embed staticlint_default.json
var defaultConfig []byte

// ConfigChecks содержит названия анализаторов из пакета staticcheck
type ConfigChecks struct {
	AnalyzerNames []string `json:"staticlint"`
}

// configAnalyzers Возвращает указатель на структуру, которая содержит список имен анализаторов.
// Структура заполняется из JSON файла конфигурации.
// Файл конфигурации должен иметь такое же название, как и бинарный файл, только иметь расширение ".json".
// Поиск файла конфигурации производится в той директории, где находится исполняемый файл.
func configAnalyzers() (*ConfigChecks, error) {
	fullPath, err := os.Executable()
	if err != nil {
		return nil, err
	}

	configPath := strings.TrimSuffix(fullPath, filepath.Ext(fullPath)) + ".json"

	data, err := os.ReadFile(configPath)
	if err != nil {
		data = defaultConfig
	}

	var cfg ConfigChecks
	if err = json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func main() {

	analyzers := []*analysis.Analyzer{
		printf.Analyzer,               // Анализатор спецификаторов формативания текста
		shadow.Analyzer,               // Анализатор затененных переменных
		structtag.Analyzer,            // Анализатор структурных тегов
		unmarshal.Analyzer,            // Анализатор проверяет передачу указателя на структуру при декодировании
		bools.Analyzer,                // Анализатор излишней проверки переменной bool с константой
		errorsas.Analyzer,             // Анализатор упакованных ошибок
		httpresponse.Analyzer,         // Анализатор проверяет наличие ошибок с помощью ответов HTTP
		nilfunc.Analyzer,              // Анализатор проверяет бесполезные сравнения с nil
		unreachable.Analyzer,          // Анализатор кода, который никогда не будет выполнен
		exitchecker.ExitCheckAnalyzer, // Анализатор вызова os.Exit() из функции main
	}

	conf, errConf := configAnalyzers()
	if errConf != nil {
		panic(errConf)
	}

	confAnalyzers := make(map[string]bool)
	for _, v := range conf.AnalyzerNames {
		confAnalyzers[v] = true
	}

	// Добавление анализаторов из tools/staticcheck, которые указаны в конфигурационном файле
	for _, v := range staticcheck.Analyzers {
		if confAnalyzers[v.Analyzer.Name] {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	// Добавление анализаторов из tools/simple, которые указаны в конфигурационном файле
	for _, v := range simple.Analyzers {
		if confAnalyzers[v.Analyzer.Name] {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	// добавляем анализаторы из tools/stylecheck, которые указаны в файле конфигурации
	for _, v := range stylecheck.Analyzers {
		if confAnalyzers[v.Analyzer.Name] {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	multichecker.Main(
		analyzers...,
	)
}
