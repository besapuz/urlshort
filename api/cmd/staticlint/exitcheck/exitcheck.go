package exitcheck

import (
	"golang.org/x/tools/go/analysis"
)

// GetAnalyzer возвращает настроенный анализатор для проверки os.Exit.
func GetAnalyzer() *analysis.Analyzer {
	return Analyzer
}
