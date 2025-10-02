package service

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestValidateTableTennisScore(t *testing.T) {
	tests := []struct {
		name       string
		setsA      int32
		setsB      int32
		setsToPlay int32
		wantError  bool
		errorCode  codes.Code
	}{
		// Best-of-3 tests
		{"Best-of-3 valid 2-0", 2, 0, 3, false, codes.OK},
		{"Best-of-3 valid 2-1", 2, 1, 3, false, codes.OK},
		{"Best-of-3 valid 0-2", 0, 2, 3, false, codes.OK},
		{"Best-of-3 valid 1-2", 1, 2, 3, false, codes.OK},
		{"Best-of-3 invalid tie", 1, 1, 3, true, codes.InvalidArgument},
		{"Best-of-3 invalid insufficient sets", 1, 0, 3, true, codes.InvalidArgument},
		{"Best-of-3 invalid too many sets", 3, 0, 3, true, codes.InvalidArgument},

		// Best-of-5 tests
		{"Best-of-5 valid 3-0", 3, 0, 5, false, codes.OK},
		{"Best-of-5 valid 3-1", 3, 1, 5, false, codes.OK},
		{"Best-of-5 valid 3-2", 3, 2, 5, false, codes.OK},
		{"Best-of-5 valid 0-3", 0, 3, 5, false, codes.OK},
		{"Best-of-5 valid 1-3", 1, 3, 5, false, codes.OK},
		{"Best-of-5 valid 2-3", 2, 3, 5, false, codes.OK},
		{"Best-of-5 invalid tie", 2, 2, 5, true, codes.InvalidArgument},
		{"Best-of-5 invalid insufficient sets", 2, 0, 5, true, codes.InvalidArgument},
		{"Best-of-5 invalid too many sets loser", 4, 3, 5, true, codes.InvalidArgument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTableTennisScore(tt.setsA, tt.setsB, tt.setsToPlay)
			
			if tt.wantError {
				if err == nil {
					t.Errorf("validateTableTennisScore() expected error but got none")
					return
				}
				if status.Code(err) != tt.errorCode {
					t.Errorf("validateTableTennisScore() error code = %v, want %v", status.Code(err), tt.errorCode)
				}
			} else {
				if err != nil {
					t.Errorf("validateTableTennisScore() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateTableTennisScoreEdgeCases(t *testing.T) {
	// Test that validation works for unusual best-of-X scenarios
	tests := []struct {
		name       string
		setsA      int32
		setsB      int32
		setsToPlay int32
		wantError  bool
	}{
		{"Best-of-7 valid 4-0", 4, 0, 7, false},
		{"Best-of-7 valid 4-3", 4, 3, 7, false},
		{"Best-of-7 invalid tie", 3, 3, 7, true},
		{"Best-of-7 invalid insufficient", 3, 0, 7, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTableTennisScore(tt.setsA, tt.setsB, tt.setsToPlay)
			
			if tt.wantError && err == nil {
				t.Errorf("validateTableTennisScore() expected error but got none")
			} else if !tt.wantError && err != nil {
				t.Errorf("validateTableTennisScore() unexpected error = %v", err)
			}
		})
	}
}