package validate

import "testing"

func TestParseAmountEUR_Table(t *testing.T) {
	SetAmountLenientOCR(false)
	tests := []struct {
		in      string
		format  string
		want    int64
		wantErr bool
	}{
		{"1", "", 100, false},
		{"1.2", "", 120, false},
		{"1,2", "", 120, false},
		{"1.23", "", 123, false},
		{"001.00", "", 100, false},
		{"30.12", "", 3012, false},
		{"30,12", "", 3012, false},
		{"EUR 30.12", "", 3012, false},
		{"30,12 €", "", 3012, false},
		{"49.90", "eur_dot", 4990, false},
		{"49,90", "eur_comma", 4990, false},
		{"1 234,50", "eur_grouped_space_comma", 123450, false},
		{"1.234,50", "eur_grouped_dot_comma", 123450, false},
		{"49,90", "eur_dot", 0, true},
		{"49.90", "eur_comma", 0, true},
		{"1.234,50", "eur_grouped_space_comma", 0, true},
		{"1 234,50", "eur_grouped_dot_comma", 0, true},
		{"49.90", "unknown_profile", 0, true},
		{"0", "", 0, false},
		{"$30.12", "", 0, true},
		{"USD 30.12", "", 0, true},
		{"", "", 0, true},
		{"1.234", "", 0, true},
		{"-1", "", 0, true},
		{"abc", "", 0, true},
		{"999999999999", "", 99999999999900, false}, // format ok; higher-level validator rejects amount too large
	}

	for _, tt := range tests {
		got, err := parseAmountEUR(tt.in, tt.format)
		if (err != nil) != tt.wantErr {
			t.Fatalf("parseAmountEUR(%q, %q) err=%v wantErr=%v", tt.in, tt.format, err, tt.wantErr)
		}
		if !tt.wantErr && got != tt.want {
			t.Fatalf("parseAmountEUR(%q, %q)=%d want %d", tt.in, tt.format, got, tt.want)
		}
	}
}

func TestParseAmountEUR_LenientOCR(t *testing.T) {
	SetAmountLenientOCR(true)
	defer SetAmountLenientOCR(false)

	tests := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"EUR 1 234,50", 123450, false},
		{"1.234,50 €", 123450, false},
		{"1,234.50 EUR", 123450, false},
		{"1O,5", 1050, false},  // OCR O -> 0
		{"US$ 10.00", 0, true}, // non-EUR stays rejected
		{"GBP 10,00", 0, true}, // non-EUR stays rejected
		{"EUR10USD", 0, true},  // conflicting markers -> rejected
		{"nonsense", 0, true},
	}

	for _, tt := range tests {
		got, err := parseAmountEUR(tt.in, "")
		if (err != nil) != tt.wantErr {
			t.Fatalf("parseAmountEUR(%q) err=%v wantErr=%v", tt.in, err, tt.wantErr)
		}
		if !tt.wantErr && got != tt.want {
			t.Fatalf("parseAmountEUR(%q)=%d want %d", tt.in, got, tt.want)
		}
	}
}

func TestParseAmountEUR_AutoLenientProfile(t *testing.T) {
	SetAmountLenientOCR(false)
	if _, err := parseAmountEUR("1O,5", "auto_eur_lenient"); err == nil {
		t.Fatalf("expected error when auto_eur_lenient is used while AMOUNT_LENIENT_OCR is disabled")
	}

	SetAmountLenientOCR(true)
	defer SetAmountLenientOCR(false)
	got, err := parseAmountEUR("1O,5", "auto_eur_lenient")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 1050 {
		t.Fatalf("got %d, want 1050", got)
	}
}

func TestParseAmountEUR_EquivalentFormsProperty(t *testing.T) {
	SetAmountLenientOCR(true)
	defer SetAmountLenientOCR(false)

	cases := []struct {
		amount string
		format string
		want   int64
	}{
		{"49.90", "eur_dot", 4990},
		{"49,90", "eur_comma", 4990},
		{"1 234,50", "eur_grouped_space_comma", 123450},
		{"1.234,50", "eur_grouped_dot_comma", 123450},
		{"EUR 1 234,50", "auto_eur_lenient", 123450},
	}

	for _, tc := range cases {
		got, err := parseAmountEUR(tc.amount, tc.format)
		if err != nil {
			t.Fatalf("parseAmountEUR(%q, %q) unexpected error: %v", tc.amount, tc.format, err)
		}
		if got != tc.want {
			t.Fatalf("parseAmountEUR(%q, %q)=%d want %d", tc.amount, tc.format, got, tc.want)
		}
	}
}

func TestParseAmountEUR_NoisyUnicodeEdgeCases(t *testing.T) {
	SetAmountLenientOCR(true)
	defer SetAmountLenientOCR(false)

	reject := []struct {
		amount string
		format string
	}{
		{"💸49,90", "eur_comma"},
		{"INV#2026 EUR49,90 TOTAL", "eur_comma"},
		{"EUR10USD", ""},
		{"€ 10 $", "auto_eur_lenient"},
	}
	for _, tc := range reject {
		if _, err := parseAmountEUR(tc.amount, tc.format); err == nil {
			t.Fatalf("expected rejection for amount=%q format=%q", tc.amount, tc.format)
		}
	}
}
