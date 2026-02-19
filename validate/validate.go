package validate

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	reBIC = regexp.MustCompile(`^[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?$`)
)

type Input struct {
	Scheme              string `json:"scheme"`
	Name                string `json:"name"`
	IBAN                string `json:"iban"`
	BIC                 string `json:"bic"`
	Amount              string `json:"amount"`
	AmountFormat        string `json:"amount_format"`
	Purpose             string `json:"purpose"`
	RemittanceReference string `json:"remittance_reference"`
	RemittanceText      string `json:"remittance_text"`
	Information         string `json:"information"`
}

type Clean struct {
	Scheme              string
	Name                string
	IBAN                string
	BIC                 string
	AmountCents         int64
	Purpose             string
	RemittanceReference string
	RemittanceText      string
	Information         string
}

func CleanAndValidate(in Input) (*Clean, error) {
	scheme := strings.ToLower(strings.TrimSpace(in.Scheme))
	name := strings.TrimSpace(in.Name)
	purpose := strings.TrimSpace(in.Purpose)
	remRef := strings.TrimSpace(in.RemittanceReference)
	remText := strings.TrimSpace(in.RemittanceText)
	info := strings.TrimSpace(in.Information)

	iban := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(in.IBAN), " ", ""))
	bic := strings.ToUpper(strings.TrimSpace(in.BIC))

	if scheme == "" {
		scheme = "epc_sct"
	}
	if scheme != "epc_sct" {
		return nil, fmt.Errorf("unsupported scheme")
	}

	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if iban == "" {
		return nil, fmt.Errorf("iban is required")
	}
	if !ValidIBAN(iban) {
		return nil, fmt.Errorf("invalid iban")
	}

	// BIC is optional in EPC spec, but you asked to verify it.
	if bic == "" {
		return nil, fmt.Errorf("bic is required")
	}
	if !reBIC.MatchString(bic) {
		return nil, fmt.Errorf("invalid bic")
	}

	amtCents, err := parseAmountEUR(in.Amount, in.AmountFormat)
	if err != nil {
		return nil, err
	}
	if amtCents <= 0 {
		return nil, fmt.Errorf("amount must be > 0")
	}
	if amtCents > 99999999999 {
		return nil, fmt.Errorf("amount too large")
	}

	name = truncateRunes(name, 70)
	if purpose == "" {
		purpose = "GDDS"
	}
	purpose = strings.ToUpper(truncateRunes(purpose, 4))
	remRef = truncateRunes(remRef, 25)
	remText = truncateRunes(remText, 140)
	info = truncateRunes(info, 70)

	if remRef != "" && remText != "" {
		return nil, fmt.Errorf("remittance_reference and remittance_text are mutually exclusive")
	}

	return &Clean{
		Scheme:              scheme,
		Name:                name,
		IBAN:                iban,
		BIC:                 bic,
		AmountCents:         amtCents,
		Purpose:             purpose,
		RemittanceReference: remRef,
		RemittanceText:      remText,
		Information:         info,
	}, nil
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	rs := []rune(s)
	return string(rs[:max])
}
