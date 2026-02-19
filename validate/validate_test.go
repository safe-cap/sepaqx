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
			name: "scheme_default_epc_sct",
			in: Input{
				Name:   "Example GmbH",
				IBAN:   "DE12500105170648489890",
				BIC:    "INGDDEFFXXX",
				Amount: "49.90",
			},
			wantErr: false,
		},
		{
			name: "scheme_explicit_epc_sct",
			in: Input{
				Scheme: "epc_sct",
				Name:   "Example GmbH",
				IBAN:   "DE12500105170648489890",
				BIC:    "INGDDEFFXXX",
				Amount: "49.90",
			},
			wantErr: false,
		},
		{
			name: "scheme_unsupported",
			in: Input{
				Scheme: "pix",
				Name:   "Example GmbH",
				IBAN:   "DE12500105170648489890",
				BIC:    "INGDDEFFXXX",
				Amount: "49.90",
			},
			wantErr: true,
		},
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
			name: "unicode_name_cyrillic",
			in: Input{
				Name:   "Сервис Москва",
				IBAN:   "DE12500105170648489890",
				BIC:    "INGDDEFFXXX",
				Amount: "1",
			},
			wantErr: false,
		},
		{
			name: "unicode_name_latin_diacritics",
			in: Input{
				Name:   "Muller & François AG",
				IBAN:   "DE12500105170648489890",
				BIC:    "INGDDEFFXXX",
				Amount: "1",
			},
			wantErr: false,
		},
		{
			name: "unicode_name_greek",
			in: Input{
				Name:   "Αθηνα Tech EE",
				IBAN:   "DE12500105170648489890",
				BIC:    "INGDDEFFXXX",
				Amount: "1",
			},
			wantErr: false,
		},
		{
			name: "unicode_name_japanese",
			in: Input{
				Name:   "東京株式会社",
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
			name: "amount_format_explicit_comma_valid",
			in: Input{
				Name:         "Example GmbH",
				IBAN:         "DE12500105170648489890",
				BIC:          "INGDDEFFXXX",
				Amount:       "49,90",
				AmountFormat: "eur_comma",
			},
			wantErr: false,
		},
		{
			name: "amount_format_mismatch_invalid",
			in: Input{
				Name:         "Example GmbH",
				IBAN:         "DE12500105170648489890",
				BIC:          "INGDDEFFXXX",
				Amount:       "49,90",
				AmountFormat: "eur_dot",
			},
			wantErr: true,
		},
		{
			name: "amount_format_unsupported",
			in: Input{
				Name:         "Example GmbH",
				IBAN:         "DE12500105170648489890",
				BIC:          "INGDDEFFXXX",
				Amount:       "49.90",
				AmountFormat: "custom_profile",
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
	if cleaned.Scheme != "epc_sct" {
		t.Fatalf("expected default scheme epc_sct, got %q", cleaned.Scheme)
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

func TestCleanAndValidate_UnicodeTruncationByRunes(t *testing.T) {
	name := strings.Repeat("Ж", 80)
	info := strings.Repeat("あ", 90)

	cleaned, err := CleanAndValidate(Input{
		Name:        name,
		IBAN:        "DE12500105170648489890",
		BIC:         "INGDDEFFXXX",
		Amount:      "1",
		Information: info,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len([]rune(cleaned.Name)); got != 70 {
		t.Fatalf("expected name truncated to 70 runes, got %d", got)
	}
	if got := len([]rune(cleaned.Information)); got != 70 {
		t.Fatalf("expected information truncated to 70 runes, got %d", got)
	}
}
