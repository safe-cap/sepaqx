package validate

import (
	"strings"
	"testing"
)

func TestCleanAndValidate_Table(t *testing.T) {
	tests := []struct {
		name    string
		in      Input
		wantErr bool
	}{
		{
			name: "valid_basic",
			in: Input{
				Name:   "Example GmbH",
				IBAN:   "DE12500105170648489890",
				BIC:    "INGDDEFFXXX",
				Amount: "49.90",
			},
			wantErr: false,
		},
		{
			name: "unicode_name",
			in: Input{
				Name:   "Müller & Сервис",
				IBAN:   "DE12500105170648489890",
				BIC:    "INGDDEFFXXX",
				Amount: "1",
			},
			wantErr: false,
		},
		{
			name: "purpose_too_long_truncates",
			in: Input{
				Name:    "Example GmbH",
				IBAN:    "DE12500105170648489890",
				BIC:     "INGDDEFFXXX",
				Amount:  "1",
				Purpose: "ABCDE",
			},
			wantErr: false,
		},
		{
			name: "amount_invalid_format",
			in: Input{
				Name:   "Example GmbH",
				IBAN:   "DE12500105170648489890",
				BIC:    "INGDDEFFXXX",
				Amount: "1.234",
			},
			wantErr: true,
		},
		{
			name: "amount_zero",
			in: Input{
				Name:   "Example GmbH",
				IBAN:   "DE12500105170648489890",
				BIC:    "INGDDEFFXXX",
				Amount: "0",
			},
			wantErr: true,
		},
		{
			name: "amount_too_large",
			in: Input{
				Name:   "Example GmbH",
				IBAN:   "DE12500105170648489890",
				BIC:    "INGDDEFFXXX",
				Amount: "999999999999",
			},
			wantErr: true,
		},
		{
			name: "mutual_exclusion",
			in: Input{
				Name:                "Example GmbH",
				IBAN:                "DE12500105170648489890",
				BIC:                 "INGDDEFFXXX",
				Amount:              "1",
				RemittanceReference: "RF18539007547034",
				RemittanceText:      "Text",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CleanAndValidate(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected err=%v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestCleanAndValidate_TruncationAndDefaults(t *testing.T) {
	long := strings.Repeat("A", 100)
	in := Input{
		Name:        long,
		IBAN:        "DE12500105170648489890",
		BIC:         "INGDDEFFXXX",
		Amount:      "1",
		Purpose:     "abcdE",
		Information: long,
	}
	cleaned, err := CleanAndValidate(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cleaned.Name) != 70 {
		t.Fatalf("expected name truncated to 70, got %d", len(cleaned.Name))
	}
	if cleaned.Purpose != "ABCD" {
		t.Fatalf("expected purpose uppercased+truncated to ABCD, got %q", cleaned.Purpose)
	}
	if len(cleaned.Information) != 70 {
		t.Fatalf("expected information truncated to 70, got %d", len(cleaned.Information))
	}
	if cleaned.RemittanceReference != "" || cleaned.RemittanceText != "" {
		t.Fatalf("expected empty remittance fields")
	}
}
