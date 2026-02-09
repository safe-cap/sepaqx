package validate

import "testing"

func TestParseAmountEUR_Table(t *testing.T) {
	tests := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"1", 100, false},
		{"1.2", 120, false},
		{"1,2", 120, false},
		{"1.23", 123, false},
		{"001.00", 100, false},
		{"0", 0, false},
		{"", 0, true},
		{"1.234", 0, true},
		{"-1", 0, true},
		{"abc", 0, true},
		{"999999999999", 99999999999900, false}, // format ok; higher-level validator rejects amount too large
	}

	for _, tt := range tests {
		got, err := parseAmountEUR(tt.in)
		if (err != nil) != tt.wantErr {
			t.Fatalf("parseAmountEUR(%q) err=%v wantErr=%v", tt.in, err, tt.wantErr)
		}
		if !tt.wantErr && got != tt.want {
			t.Fatalf("parseAmountEUR(%q)=%d want %d", tt.in, got, tt.want)
		}
	}
}
