package main

import (
	"testing"

	symbollistusecase "stock_backend/internal/feature/symbollist/usecase"
)

func TestShouldFailExit(t *testing.T) {
	testCases := []struct {
		name      string
		result    symbollistusecase.LogoIngestResult
		threshold float64
		want      bool
	}{
		{
			name:      "all symbols succeed",
			result:    symbollistusecase.LogoIngestResult{Total: 10, Succeeded: 10, Failed: 0},
			threshold: 0.2,
			want:      false,
		},
		{
			name:      "failure rate equals threshold",
			result:    symbollistusecase.LogoIngestResult{Total: 10, Succeeded: 8, Failed: 2},
			threshold: 0.2,
			want:      false,
		},
		{
			name:      "failure rate exceeds threshold",
			result:    symbollistusecase.LogoIngestResult{Total: 10, Succeeded: 7, Failed: 3},
			threshold: 0.2,
			want:      true,
		},
		{
			name:      "no symbols",
			result:    symbollistusecase.LogoIngestResult{Total: 0},
			threshold: 0.2,
			want:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldFailExit(tc.result, tc.threshold); got != tc.want {
				t.Errorf("shouldFailExit(%+v, %v)=%v, want %v", tc.result, tc.threshold, got, tc.want)
			}
		})
	}
}
