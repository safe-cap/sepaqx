package qr

import (
	"fmt"
	"strings"
)

func BuildEPCPayload(name, iban, bic string, amountCents int64, purpose, remittanceRef, remittanceText, info string) (string, error) {
	if name == "" || iban == "" || bic == "" || amountCents <= 0 {
		return "", fmt.Errorf("missing required fields")
	}
	if remittanceRef != "" && remittanceText != "" {
		return "", fmt.Errorf("remittance reference and text are mutually exclusive")
	}

	amountStr := "EUR" + fmt.Sprintf("%d.%02d", amountCents/100, amountCents%100)

	lines := []string{
		"BCD",
		"001",
		"1",
		"SCT",
		bic,
		name,
		iban,
		amountStr,
		purpose,
		remittanceRef,
		remittanceText,
		info,
	}

	return strings.Join(lines, "\n"), nil
}
