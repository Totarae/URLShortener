// Package main запускает multichecker.
//
// Он включает:
// - стандартные анализаторы go/analysis/passes
// - все SA-анализаторы staticcheck
// - один не-SA анализатор (S1000)
// - ещё один не-SA анализатор (U1000)
// - два публичных анализатора: bodyclose и nilerr
// - собственный анализатор noexit (запрещает os.Exit в main)
//
// Запуск:
//
//	go run ./cmd/staticlint ./...
package main

import (
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"honnef.co/go/tools/staticcheck"

	"github.com/Totarae/URLShortener/cmd/staticlint/noexit"
)

func main() {
	var analyzers []*analysis.Analyzer

	analyzers = append(analyzers,
		shadow.Analyzer,
		structtag.Analyzer,
		nilness.Analyzer,
		fieldalignment.Analyzer,
		printf.Analyzer,
	)

	// SA-анализаторы
	for _, a := range staticcheck.Analyzers {
		if a.Analyzer.Name[:2] == "SA" {
			analyzers = append(analyzers, a.Analyzer)
		}
	}

	// не-SA:
	if a := findAnalyzer("S1000"); a != nil {
		analyzers = append(analyzers, a) // упрощения
	}
	if a := findAnalyzer("U1000"); a != nil {
		analyzers = append(analyzers, a) // неиспользуемые параметры
	}

	// публичный анализатор (не из staticcheck)
	analyzers = append(analyzers, bodyclose.Analyzer)

	// собственный анализатор
	analyzers = append(analyzers, noexit.NewAnalyzer())

	multichecker.Main(analyzers...)
}

func findAnalyzer(name string) *analysis.Analyzer {
	for _, a := range staticcheck.Analyzers {
		if a.Analyzer.Name == name {
			return a.Analyzer
		}
	}
	return nil
}
