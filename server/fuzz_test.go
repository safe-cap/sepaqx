package server

import (
	"net/url"
	"testing"

	"github.com/safe-cap/sepaqx/validate"
)

func FuzzQueryParsing(f *testing.F) {
	f.Add("name=Example%20GmbH&iban=DE12500105170648489890&bic=INGDDEFFXXX&amount=49.90")
	f.Add("name=&iban=&bic=&amount=")
	f.Add("name=%FF%FE&amount=1.2")
	f.Fuzz(func(t *testing.T, qs string) {
		q, err := url.ParseQuery(qs)
		if err != nil {
			return
		}
		in := validate.Input{
			Scheme:              q.Get("scheme"),
			Name:                q.Get("name"),
			IBAN:                q.Get("iban"),
			BIC:                 q.Get("bic"),
			Amount:              q.Get("amount"),
			Purpose:             q.Get("purpose"),
			RemittanceReference: q.Get("remittance_reference"),
			RemittanceText:      q.Get("remittance_text"),
			Information:         q.Get("information"),
		}
		_, _ = validate.CleanAndValidate(in)
	})
}
