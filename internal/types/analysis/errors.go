// internal/analysis/errors.go
package analysis

import "fmt"

// AnalysisError - ошибка анализа
type AnalysisError struct {
	Message string
}

func (e *AnalysisError) Error() string {
	return e.Message
}

// Ошибки анализа
var (
	ErrLowConfidence    = &AnalysisError{Message: "low confidence"}
	ErrNoChange         = &AnalysisError{Message: "no price change"}
	ErrInvalidSymbol    = &AnalysisError{Message: "invalid symbol"}
	ErrInsufficientData = &AnalysisError{Message: "insufficient data"}
	ErrAnalysisFailed   = &AnalysisError{Message: "analysis failed"}
)

// NewAnalysisError создает новую ошибку анализа
func NewAnalysisError(format string, args ...interface{}) *AnalysisError {
	return &AnalysisError{
		Message: fmt.Sprintf(format, args...),
	}
}

// ErrorWithContext добавляет контекст к ошибке
func (e *AnalysisError) WithContext(context string) *AnalysisError {
	return &AnalysisError{
		Message: fmt.Sprintf("%s: %s", context, e.Message),
	}
}

// ErrorWithSymbol добавляет символ к ошибке
func (e *AnalysisError) WithSymbol(symbol string) *AnalysisError {
	return &AnalysisError{
		Message: fmt.Sprintf("%s: %s", symbol, e.Message),
	}
}
