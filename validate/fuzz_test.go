package validate

import (
	"encoding/json"
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
		_, _ = parseAmountEUR(s)
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
