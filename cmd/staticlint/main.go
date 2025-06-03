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
	"github.com/Totarae/URLShortener/cmd/staticlint/noexit"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"honnef.co/go/tools/staticcheck"
)

func main() {
	var analyzers []*analysis.Analyzer

	// стандартные анализаторы
	analyzers = append(analyzers,
		shadow.Analyzer,
		structtag.Analyzer,
		nilness.Analyzer,
		fieldalignment.Analyzer,
		printf.Analyzer,
	)

	// SA-анализаторы staticcheck
	for _, info := range staticcheck.Analyzers {
		if info.Analyzer != nil && len(info.Analyzer.Name) >= 2 && info.Analyzer.Name[:2] == "SA" {
			analyzers = append(analyzers, info.Analyzer)
		}
	}

	// отдельные не-SA анализаторы
	for _, name := range []string{"S1000", "U1000"} {
		if a := findStaticcheckAnalyzer(name); a != nil {
			analyzers = append(analyzers, a)
		}
	}

	// внешние и собственные
	analyzers = append(analyzers, bodyclose.Analyzer)
	analyzers = append(analyzers, noexit.NewAnalyzer())

	multichecker.Main(analyzers...)
}

func findStaticcheckAnalyzer(name string) *analysis.Analyzer {
	for _, info := range staticcheck.Analyzers {
		if info.Analyzer != nil && info.Analyzer.Name == name {
			return info.Analyzer
		}
	}
	return nil
}
