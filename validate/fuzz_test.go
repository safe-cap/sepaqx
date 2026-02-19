package validate

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func FuzzCleanAndValidateJSON(f *testing.F) {
	f.Add(`{"name":"Example GmbH","iban":"DE12500105170648489890","bic":"INGDDEFFXXX","amount":"49.90"}`)
	f.Add(`{"name":"","iban":"","bic":"","amount":""}`)
	f.Add(`{"name":"A","iban":"DE12500105170648489890","bic":"INGDDEFFXXX","amount":"1.2","purpose":"GDDS"}`)
	f.Fuzz(func(t *testing.T, body string) {
		var in Input
		// decode via JSON in server; this fuzz just drives validation robustness
		_ = jsonUnmarshal(body, &in)
		_, _ = CleanAndValidate(in)
	})
}

func FuzzParseAmountEUR(f *testing.F) {
	seeds := []string{"1", "1.2", "1,2", "1.23", "001.00", "0", "999999999999", "abc", "-1", "1.234"}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = parseAmountEUR(s, "")
	})
}

func FuzzParseAmountEURWithFormat(f *testing.F) {
	formats := []string{
		"",
		"auto",
		"eur_dot",
		"eur_comma",
		"eur_grouped_space_comma",
		"eur_grouped_dot_comma",
		"auto_eur_lenient",
		"custom_profile",
	}
	amounts := []string{
		"49.90",
		"49,90",
		"1 234,50",
		"1.234,50",
		"EUR 49.90",
		"49,90 €",
		"1O,5",
		"EUR10USD",
		"US$ 10.00",
		"GBP 10,00",
		"nonsense",
	}
	for _, a := range amounts {
		for _, fm := range formats {
			f.Add(a, fm)
		}
	}

	f.Fuzz(func(t *testing.T, amount, format string) {
		_, _ = parseAmountEUR(amount, format)
	})
}

func FuzzParseAmountEURConflictingCurrencyMarkers(f *testing.F) {
	conflicts := []string{
		"EUR10USD",
		"USD 10 €",
		"€ 10 $",
		"US$ EUR 10.00",
		"EUR 10 GBP",
		"CHF 10 €",
	}
	for _, s := range conflicts {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = parseAmountEUR(s, "")
		_, _ = parseAmountEUR(s, "auto_eur_lenient")
	})
}

func FuzzParseAmountEURNoisyUnicode(f *testing.F) {
	seeds := []string{
		"ＥＵＲ １２３,４５",              // fullwidth latin/digits
		"EUR\u00a0123,45",         // non-breaking space
		"EUR\u2009123,45",         // thin space
		"€\u2007123,45",           // figure space
		"EUR123,45руб",            // glued non-latin suffix
		"USＤ 10.00",               // mixed-width USD
		"💸49,90",                  // emoji prefix
		"INV#2026 EUR49,90 TOTAL", // glued free text
		"EUR1O,5",                 // OCR O->0 candidate
		"1\u206649,90\u2069",      // isolate controls
	}
	formats := []string{
		"",
		"auto",
		"eur_dot",
		"eur_comma",
		"eur_grouped_space_comma",
		"eur_grouped_dot_comma",
		"auto_eur_lenient",
	}
	for _, s := range seeds {
		for _, fm := range formats {
			f.Add(s, fm)
		}
	}
	f.Fuzz(func(t *testing.T, amount, format string) {
		_, _ = parseAmountEUR(amount, format)
	})
}

func FuzzParseAmountEURProfileAgreement(f *testing.F) {
	f.Add(int64(49), uint8(90))
	f.Add(int64(1234), uint8(50))
	f.Add(int64(1), uint8(5))

	f.Fuzz(func(t *testing.T, whole int64, frac uint8) {
		if whole < 0 {
			whole = -whole
		}
		whole = whole % 1000000000000 // parser allows up to 12 digits before separator
		c := int64(frac % 100)
		want := whole*100 + c

		dot := fmt.Sprintf("%d.%02d", whole, c)
		comma := fmt.Sprintf("%d,%02d", whole, c)

		spaceGrouped := comma
		dotGrouped := comma
		if whole >= 1000 {
			intPart := fmt.Sprintf("%d", whole)
			var groups []string
			for len(intPart) > 3 {
				groups = append([]string{intPart[len(intPart)-3:]}, groups...)
				intPart = intPart[:len(intPart)-3]
			}
			groups = append([]string{intPart}, groups...)
			spaceGrouped = strings.Join(groups, " ") + fmt.Sprintf(",%02d", c)
			dotGrouped = strings.Join(groups, ".") + fmt.Sprintf(",%02d", c)
		}

		got, err := parseAmountEUR(dot, "eur_dot")
		if err != nil || got != want {
			t.Fatalf("eur_dot mismatch for %q: got=%d err=%v want=%d", dot, got, err, want)
		}

		got, err = parseAmountEUR(comma, "eur_comma")
		if err != nil || got != want {
			t.Fatalf("eur_comma mismatch for %q: got=%d err=%v want=%d", comma, got, err, want)
		}

		got, err = parseAmountEUR(spaceGrouped, "eur_grouped_space_comma")
		if err != nil || got != want {
			t.Fatalf("eur_grouped_space_comma mismatch for %q: got=%d err=%v want=%d", spaceGrouped, got, err, want)
		}

		got, err = parseAmountEUR(dotGrouped, "eur_grouped_dot_comma")
		if err != nil || got != want {
			t.Fatalf("eur_grouped_dot_comma mismatch for %q: got=%d err=%v want=%d", dotGrouped, got, err, want)
		}
	})
}

func FuzzParseAmountEURAutoLenientAgreement(f *testing.F) {
	f.Add(int64(1234), uint8(50))
	f.Add(int64(10), uint8(5))

	f.Fuzz(func(t *testing.T, whole int64, frac uint8) {
		if whole < 0 {
			whole = -whole
		}
		whole = whole % 1000000000000
		c := int64(frac % 100)
		want := whole*100 + c

		SetAmountLenientOCR(true)
		defer SetAmountLenientOCR(false)

		noisy := fmt.Sprintf("INV EUR %d,%02d TOTAL", whole, c)
		got, err := parseAmountEUR(noisy, "auto_eur_lenient")
		if err != nil || got != want {
			t.Fatalf("auto_eur_lenient mismatch for %q: got=%d err=%v want=%d", noisy, got, err, want)
		}
	})
}

func FuzzValidIBAN(f *testing.F) {
	seeds := []string{
		"DE12500105170648489890",
		"GB82WEST12345698765432",
		"FR1420041010050500013M02606",
		"INVALID",
		"",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_ = ValidIBAN(s)
	})
}

// jsonUnmarshal is a tiny helper to avoid importing encoding/json in every fuzz target body.
func jsonUnmarshal(s string, v any) error {
	return json.Unmarshal([]byte(s), v)
}
